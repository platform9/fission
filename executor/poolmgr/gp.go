/*
Copyright 2016 The Fission Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package poolmgr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/environments/fetcher"
	fetcherClient "github.com/fission/fission/environments/fetcher/client"
	"github.com/fission/fission/executor/fscache"
	"github.com/fission/fission/executor/util"
)

const POD_PHASE_RUNNING string = "Running"

type (
	GenericPool struct {
		env                    *crd.Environment
		replicas               int32                         // num idle pods
		deployment             *v1beta1.Deployment           // kubernetes deployment
		namespace              string                        // namespace to keep our resources
		podReadyTimeout        time.Duration                 // timeout for generic pods to become ready
		idlePodReapTime        time.Duration                 // pods unused for idlePodReapTime are deleted
		fsCache                *fscache.FunctionServiceCache // cache funcSvc's by function, address and podname
		useSvc                 bool                          // create k8s service for specialized pods
		useIstio               bool
		poolInstanceId         string // small random string to uniquify pod names
		fetcherImage           string
		fetcherImagePullPolicy apiv1.PullPolicy
		runtimeImagePullPolicy apiv1.PullPolicy // pull policy for generic pool to created env deployment
		kubernetesClient       *kubernetes.Clientset
		fissionClient          *crd.FissionClient
		instanceId             string // poolmgr instance id
		labelsForPool          map[string]string
		requestChannel         chan *choosePodRequest
		sharedMountPath        string // used by generic pool when creating env deployment to specify the share volume path for fetcher & env
		sharedSecretPath       string
		sharedCfgMapPath       string
	}

	// serialize the choosing of pods so that choices don't conflict
	choosePodRequest struct {
		newLabels       map[string]string
		responseChannel chan *choosePodResponse
	}
	choosePodResponse struct {
		pod *apiv1.Pod
		error
	}
)

func getImagePullPolicy(policy string) apiv1.PullPolicy {
	switch policy {
	case "Always":
		return apiv1.PullAlways
	case "Never":
		return apiv1.PullNever
	default:
		return apiv1.PullIfNotPresent
	}
}

func MakeGenericPool(
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	env *crd.Environment,
	initialReplicas int32,
	namespace string,
	fsCache *fscache.FunctionServiceCache,
	instanceId string,
	enableIstio bool) (*GenericPool, error) {

	log.Printf("Creating pool for environment %v", env.Metadata)

	fetcherImage := os.Getenv("FETCHER_IMAGE")
	if len(fetcherImage) == 0 {
		fetcherImage = "fission/fetcher"
	}
	fetcherImagePullPolicy := os.Getenv("FETCHER_IMAGE_PULL_POLICY")
	if len(fetcherImagePullPolicy) == 0 {
		fetcherImagePullPolicy = "IfNotPresent"
	}
	runtimeImagePullPolicy := os.Getenv("RUNTIME_IMAGE_PULL_POLICY")
	if len(runtimeImagePullPolicy) == 0 {
		runtimeImagePullPolicy = "IfNotPresent"
	}

	// TODO: in general we need to provide the user a way to configure pools.  Initial
	// replicas, autoscaling params, various timeouts, etc.
	gp := &GenericPool{
		env:              env,
		replicas:         initialReplicas, // TODO make this an env param instead?
		requestChannel:   make(chan *choosePodRequest),
		fissionClient:    fissionClient,
		kubernetesClient: kubernetesClient,
		namespace:        namespace,
		podReadyTimeout:  5 * time.Minute, // TODO make this an env param?
		idlePodReapTime:  3 * time.Minute, // TODO make this configurable
		fsCache:          fsCache,
		poolInstanceId:   uniuri.NewLen(8),
		instanceId:       instanceId,
		fetcherImage:     fetcherImage,
		useSvc:           false,       // defaults off -- svc takes a second or more to become routable, slowing cold start
		useIstio:         enableIstio, // defaults off -- istio integration requires pod relabeling and it takes a second or more to become routable, slowing cold start
		sharedMountPath:  "/userfunc", // change this may break v1 compatibility, since most of the v1 environments have hard-coded "/userfunc" in loading path
		sharedSecretPath: "/secrets",
		sharedCfgMapPath: "/configs",
	}

	gp.runtimeImagePullPolicy = getImagePullPolicy(runtimeImagePullPolicy)

	gp.fetcherImagePullPolicy = getImagePullPolicy(fetcherImagePullPolicy)
	log.Printf("fetcher image: %v, pull policy: %v", gp.fetcherImage, gp.fetcherImagePullPolicy)

	// Labels for generic deployment/RS/pods.
	gp.labelsForPool = map[string]string{
		"environmentName":                 gp.env.Metadata.Name,
		"environmentUid":                  string(gp.env.Metadata.UID),
		fission.EXECUTOR_INSTANCEID_LABEL: gp.instanceId,
		"executorType":                    fission.ExecutorTypePoolmgr,
	}

	// create the pool
	err := gp.createPool()
	if err != nil {
		return nil, err
	}
	log.Printf("[%v] Deployment created", env.Metadata)

	go gp.choosePodService()

	return gp, nil
}

// choosePodService serializes the choosing of pods
func (gp *GenericPool) choosePodService() {
	for {
		select {
		case req := <-gp.requestChannel:
			pod, err := gp._choosePod(req.newLabels)
			if err != nil {
				req.responseChannel <- &choosePodResponse{error: err}
				continue
			}
			req.responseChannel <- &choosePodResponse{pod: pod}
		}
	}
}

// choosePod picks a ready pod from the pool and relabels it, waiting if necessary.
// returns the pod API object.
func (gp *GenericPool) choosePod(newLabels map[string]string) (*apiv1.Pod, error) {
	req := &choosePodRequest{
		newLabels:       newLabels,
		responseChannel: make(chan *choosePodResponse),
	}
	gp.requestChannel <- req
	resp := <-req.responseChannel
	return resp.pod, resp.error
}

// _choosePod is called serially by choosePodService
func (gp *GenericPool) _choosePod(newLabels map[string]string) (*apiv1.Pod, error) {
	startTime := time.Now()
	for {
		// Retries took too long, error out.
		if time.Since(startTime) > gp.podReadyTimeout {
			log.Printf("[%v] Erroring out, timed out", newLabels)
			return nil, errors.New("timeout: waited too long to get a ready pod")
		}

		// Get pods; filter the ones that are ready
		podList, err := gp.kubernetesClient.CoreV1().Pods(gp.namespace).List(
			metav1.ListOptions{
				LabelSelector: labels.Set(
					gp.deployment.Spec.Selector.MatchLabels).AsSelector().String(),
			})
		if err != nil {
			return nil, err
		}
		readyPods := make([]*apiv1.Pod, 0, len(podList.Items))
		for i := range podList.Items {
			pod := podList.Items[i]

			// If a pod has no IP it's not ready
			if len(pod.Status.PodIP) == 0 || string(pod.Status.Phase) != POD_PHASE_RUNNING {
				continue
			}

			// Wait for all containers in the pod to be ready
			podReady := true
			for _, cs := range pod.Status.ContainerStatuses {
				podReady = podReady && cs.Ready
			}

			// add it to the list of ready pods
			if podReady {
				readyPods = append(readyPods, &pod)
			}
		}
		log.Printf("[%v] found %v ready pods of %v total", newLabels, len(readyPods), len(podList.Items))

		// If there are no ready pods, wait and retry.
		if len(readyPods) == 0 {
			err = gp.waitForReadyPod()
			if err != nil {
				return nil, err
			}
			continue
		}

		// Pick a ready pod.  For now just choose randomly;
		// ideally we'd care about which node it's running on,
		// and make a good scheduling decision.
		chosenPod := readyPods[rand.Intn(len(readyPods))]

		if gp.env.Spec.AllowedFunctionsPerContainer != fission.AllowedFunctionsPerContainerInfinite {
			// Relabel.  If the pod already got picked and
			// modified, this should fail; in that case just
			// retry.
			chosenPod.ObjectMeta.Labels = newLabels
			log.Printf("relabeling pod: [%v]", chosenPod.ObjectMeta.Name)
			_, err = gp.kubernetesClient.CoreV1().Pods(gp.namespace).Update(chosenPod)
			if err != nil {
				log.Printf("failed to relabel pod [%v]: %v", chosenPod.ObjectMeta.Name, err)
				continue
			}
		}
		log.Printf("Chosen pod: %v (in %v)", chosenPod.ObjectMeta.Name, time.Since(startTime))
		return chosenPod, nil
	}
}

func (gp *GenericPool) labelsForFunction(metadata *metav1.ObjectMeta) map[string]string {
	return map[string]string{
		"functionName":                    metadata.Name,
		"functionUid":                     string(metadata.UID),
		"unmanaged":                       "true", // this allows us to easily find pods not managed by the deployment
		fission.EXECUTOR_INSTANCEID_LABEL: gp.instanceId,
	}
}

func (gp *GenericPool) scheduleDeletePod(name string) {
	go func() {
		// The sleep allows debugging or collecting logs from the pod before it's
		// cleaned up.  (We need a better solutions for both those things; log
		// aggregation and storage will help.)
		log.Printf("Error in pod '%v', scheduling cleanup", name)
		// Ignore sleep here if istio feature is enabled, function pod
		// will be deleted after 6 mins (terminationGracePeriodSeconds).
		if !gp.useIstio {
			time.Sleep(5 * time.Minute)
		}
		gp.kubernetesClient.CoreV1().Pods(gp.namespace).Delete(name, nil)
	}()
}

func IsIPv6(podIP string) bool {
	ip := net.ParseIP(podIP)
	return ip != nil && strings.Contains(podIP, ":")
}

func (gp *GenericPool) getFetcherUrl(podIP string) string {
	testUrl := os.Getenv("TEST_FETCHER_URL")
	if len(testUrl) != 0 {
		// it takes a second or so for the test service to
		// become routable once a pod is relabeled. This is
		// super hacky, but only runs in unit tests.
		time.Sleep(5 * time.Second)
		return testUrl
	}
	isv6 := IsIPv6(podIP)
	var baseUrl string
	if isv6 == false {
		baseUrl = fmt.Sprintf("http://%v:8000/", podIP)
	} else if isv6 == true { // We use bracket if the IP is in IPv6.
		baseUrl = fmt.Sprintf("http://[%v]:8000/", podIP)
	}
	return baseUrl

}

func (gp *GenericPool) getSpecializeUrl(podIP string, version int) string {
	u := os.Getenv("TEST_SPECIALIZE_URL")
	isv6 := IsIPv6(podIP)
	var baseUrl string
	if len(u) != 0 {
		return u
	}
	if isv6 == false {
		baseUrl = fmt.Sprintf("http://%v:8888", podIP)
	} else if isv6 == true { // We use bracket if the IP is in IPv6.
		baseUrl = fmt.Sprintf("http://[%v]:8888", podIP)
	}

	if version == 1 {
		return fmt.Sprintf("%v/specialize", baseUrl)
	} else {
		return fmt.Sprintf("%v/v%v/specialize", baseUrl, version)
	}
}

// specializePod chooses a pod, copies the required user-defined function to that pod
// (via fetcher), and calls the function-run container to load it, resulting in a
// specialized pod.
func (gp *GenericPool) specializePod(pod *apiv1.Pod, metadata *metav1.ObjectMeta) error {
	// for fetcher we don't need to create a service, just talk to the pod directly
	podIP := pod.Status.PodIP
	if len(podIP) == 0 {
		return errors.New("Pod has no IP")
	}
	// specialize pod with service
	if gp.useIstio {
		svc := fission.GetFunctionIstioServiceName(metadata.Name, metadata.Namespace)
		podIP = fmt.Sprintf("%v.%v", svc, gp.namespace)
	}

	// tell fetcher to get the function.
	fetcherUrl := gp.getFetcherUrl(podIP)
	log.Printf("[%v] calling fetcher to copy function with fetcher url: %v", metadata.Name, fetcherUrl)

	fn, err := gp.fissionClient.
		Functions(metadata.Namespace).
		Get(metadata.Name)
	if err != nil {
		return err
	}

	// for backward compatibility, since most v1 env
	// still try to load user function from hard coded
	// path /userfunc/user
	targetFilename := "user"
	if gp.env.Spec.Version == 2 {
		targetFilename = string(fn.Metadata.UID)
	}

	err = fetcherClient.MakeClient(fetcherUrl).Fetch(&fetcher.FetchRequest{
		FetchType: fetcher.FETCH_DEPLOYMENT,
		Package: metav1.ObjectMeta{
			Namespace: fn.Spec.Package.PackageRef.Namespace,
			Name:      fn.Spec.Package.PackageRef.Name,
		},
		Filename:       targetFilename,
		Secrets:        fn.Spec.Secrets,
		ConfigMaps:     fn.Spec.ConfigMaps,
		ExtractArchive: gp.env.Spec.ExtractArchive,
	})
	if err != nil {
		return err
	}

	// get function run container to specialize
	log.Printf("[%v] specializing pod", metadata.Name)

	// retry the specialize call a few times in case the env server hasn't come up yet
	maxRetries := 20

	loadReq := fission.FunctionLoadRequest{
		FilePath:         filepath.Join(gp.sharedMountPath, targetFilename),
		FunctionName:     fn.Spec.Package.FunctionName,
		FunctionMetadata: &fn.Metadata,
	}

	body, err := json.Marshal(loadReq)
	if err != nil {
		return err
	}

	for i := 0; i < maxRetries; i++ {
		var resp2 *http.Response
		if gp.env.Spec.Version == 2 {
			specializeUrl := gp.getSpecializeUrl(podIP, 2)
			log.Printf("specialize url: %v", specializeUrl)
			resp2, err = http.Post(specializeUrl, "application/json", bytes.NewReader(body))
		} else {
			specializeUrl := gp.getSpecializeUrl(podIP, 1)
			resp2, err = http.Post(specializeUrl, "text/plain", bytes.NewReader([]byte{}))
		}

		if err == nil && resp2.StatusCode < 300 {
			// Success
			resp2.Body.Close()
			return nil
		}

		retry := false

		// Only retry for the specific case of a connection error.
		if urlErr, ok := err.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if netErr.Op == "dial" {
					if i < maxRetries-1 {
						retry = true
					}
				}
			}
		}

		// Receive response with non-200 http code
		if err == nil {
			err = fission.MakeErrorFromHTTP(resp2)
		}

		// The istio-proxy block all http requests until it's ready
		// to serve traffic. Retry if istio feature is enabled.
		if gp.useIstio {
			retry = true
		}

		if retry && i < maxRetries-1 {
			time.Sleep(500 * time.Duration(2*i) * time.Millisecond)
			log.Printf("Error connecting to pod (%v), retrying", err)
			continue
		}

		log.Printf("Failed to specialize pod: %v", err)
		return err
	}

	return nil
}

// A pool is a deployment of generic containers for an env.  This
// creates the pool but doesn't wait for any pods to be ready.
func (gp *GenericPool) createPool() error {
	poolDeploymentName := fmt.Sprintf("%v-%v-%v",
		gp.env.Metadata.Name, gp.env.Metadata.UID, strings.ToLower(gp.poolInstanceId))

	fetcherResources, err := util.GetFetcherResources()
	if err != nil {
		return err
	}

	// Use long terminationGracePeriodSeconds for connection draining in case that
	// pod still runs user functions.
	gracePeriodSeconds := int64(6 * 60)
	if gp.env.Spec.TerminationGracePeriod > 0 {
		gracePeriodSeconds = gp.env.Spec.TerminationGracePeriod
	}

	podAnnotation := make(map[string]string)

	if gp.useIstio && gp.env.Spec.AllowAccessToExternalNetwork {
		podAnnotation["sidecar.istio.io/inject"] = "false"
	}

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   poolDeploymentName,
			Labels: gp.labelsForPool,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &gp.replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: gp.labelsForPool,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      gp.labelsForPool,
					Annotations: podAnnotation,
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: fission.SharedVolumeUserfunc,
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: fission.SharedVolumeSecrets,
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: fission.SharedVolumeConfigmaps,
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []apiv1.Container{
						fission.MergeContainerSpecs(&apiv1.Container{
							Name:                   gp.env.Metadata.Name,
							Image:                  gp.env.Spec.Runtime.Image,
							ImagePullPolicy:        gp.runtimeImagePullPolicy,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumeUserfunc,
									MountPath: gp.sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: gp.sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: gp.sharedCfgMapPath,
								},
							},
							Resources: gp.env.Spec.Resources,
							// Pod is removed from endpoints list for service when it's
							// state became "Termination". We used preStop hook as the
							// workaround for connection draining since pod maybe shutdown
							// before grace period expires.
							// https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods
							// https://github.com/kubernetes/kubernetes/issues/47576#issuecomment-308900172
							Lifecycle: &apiv1.Lifecycle{
								PreStop: &apiv1.Handler{
									Exec: &apiv1.ExecAction{
										Command: []string{
											"sleep",
											fmt.Sprintf("%v", gracePeriodSeconds),
										},
									},
								},
							},
						}, gp.env.Spec.Runtime.Container),
						{
							Name:                   "fetcher",
							Image:                  gp.fetcherImage,
							ImagePullPolicy:        gp.fetcherImagePullPolicy,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumeUserfunc,
									MountPath: gp.sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: gp.sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: gp.sharedCfgMapPath,
								},
							},
							Resources: fetcherResources,
							Command: []string{"/fetcher",
								"-secret-dir", gp.sharedSecretPath,
								"-cfgmap-dir", gp.sharedCfgMapPath,
								gp.sharedMountPath},
							// Pod is removed from endpoints list for service when it's
							// state became "Termination". We used preStop hook as the
							// workaround for connection draining since pod maybe shutdown
							// before grace period expires.
							// https://kubernetes.io/docs/concepts/workloads/pods/pod/#termination-of-pods
							// https://github.com/kubernetes/kubernetes/issues/47576#issuecomment-308900172
							Lifecycle: &apiv1.Lifecycle{
								PreStop: &apiv1.Handler{
									Exec: &apiv1.ExecAction{
										Command: []string{
											"sleep",
											fmt.Sprintf("%v", gracePeriodSeconds),
										},
									},
								},
							},
							ReadinessProbe: &apiv1.Probe{
								InitialDelaySeconds: 1,
								PeriodSeconds:       1,
								FailureThreshold:    30,
								Handler: apiv1.Handler{
									HTTPGet: &apiv1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8000,
										},
									},
								},
							},
							LivenessProbe: &apiv1.Probe{
								InitialDelaySeconds: 35,
								PeriodSeconds:       5,
								Handler: apiv1.Handler{
									HTTPGet: &apiv1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8000,
										},
									},
								},
							},
						},
					},
					ServiceAccountName: "fission-fetcher",
					// TerminationGracePeriodSeconds should be equal to the
					// sleep time of preStop to make sure that SIGTERM is sent
					// to pod after 6 mins.
					TerminationGracePeriodSeconds: &gracePeriodSeconds,
				},
			},
		},
	}

	depl, err := gp.kubernetesClient.ExtensionsV1beta1().Deployments(gp.namespace).Create(deployment)
	if err != nil {
		log.Printf("Error creating deployment for %s in kubernetes, err: %v", deployment.Name, err)
		return err
	}
	gp.deployment = depl
	return nil
}

func (gp *GenericPool) waitForReadyPod() error {
	startTime := time.Now()
	for {
		// TODO: for now we just poll; use a watch instead
		depl, err := gp.kubernetesClient.ExtensionsV1beta1().Deployments(gp.namespace).Get(
			gp.deployment.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			log.Printf("err: %v", err)
			return err
		}
		gp.deployment = depl
		if gp.deployment.Status.AvailableReplicas > 0 {
			return nil
		}

		if time.Since(startTime) > gp.podReadyTimeout {
			return errors.New("timeout: waited too long for pod to be ready")
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func (gp *GenericPool) createSvc(name string, labels map[string]string) (*apiv1.Service, error) {
	service := apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Ports: []apiv1.ServicePort{
				{
					Protocol:   apiv1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(8888),
				},
			},
			Selector: labels,
		},
	}
	svc, err := gp.kubernetesClient.CoreV1().Services(gp.namespace).Create(&service)
	return svc, err
}

func (gp *GenericPool) GetFuncSvc(m *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	log.Printf("[%v] Choosing pod from pool", m.Name)
	newLabels := gp.labelsForFunction(m)

	if gp.useIstio {
		// Istio only allows accessing pod through k8s service, and requests come to
		// service are not always being routed to the same pod. For example:

		// If there is only one pod (podA) behind the service svcX.

		// svcX -> podA

		// All requests (specialize request & function access requests)
		// will be routed to podA without any problem.

		// If podA and podB are behind svcX.

		// svcX -> podA (specialized)
		//      -> podB (non-specialized)

		// The specialize request may be routed to podA and the function access
		// requests may go to podB. In this case, the function cannot be served
		// properly.

		// To prevent such problem, we need to delete old versions function pods
		// and make sure that there is only one pod behind the service

		sel := map[string]string{
			"functionName": m.Name,
			"functionUid":  string(m.UID),
		}
		podList, err := gp.kubernetesClient.CoreV1().Pods(gp.namespace).List(metav1.ListOptions{
			LabelSelector: labels.Set(sel).AsSelector().String(),
		})
		if err != nil {
			return nil, err
		}

		// Remove old versions function pods
		for _, pod := range podList.Items {
			// Delete pod no matter what status it is
			gp.kubernetesClient.CoreV1().Pods(gp.namespace).Delete(pod.ObjectMeta.Name, nil)
		}
	}

	pod, err := gp.choosePod(newLabels)
	if err != nil {
		return nil, err
	}

	err = gp.specializePod(pod, m)
	if err != nil {
		gp.scheduleDeletePod(pod.ObjectMeta.Name)
		return nil, err
	}
	log.Printf("Specialized pod: %v", pod.ObjectMeta.Name)

	var svcHost string
	if gp.useSvc && !gp.useIstio {
		svcName := fmt.Sprintf("svc-%v", m.Name)
		if len(m.UID) > 0 {
			svcName = fmt.Sprintf("%s-%v", svcName, m.UID)
		}

		labels := gp.labelsForFunction(m)
		svc, err := gp.createSvc(svcName, labels)
		if err != nil {
			gp.scheduleDeletePod(pod.ObjectMeta.Name)
			return nil, err
		}
		if svc.ObjectMeta.Name != svcName {
			gp.scheduleDeletePod(pod.ObjectMeta.Name)
			return nil, errors.New(fmt.Sprintf("sanity check failed for svc %v", svc.ObjectMeta.Name))
		}

		// the fission router isn't in the same namespace, so return a
		// namespace-qualified hostname
		svcHost = fmt.Sprintf("%v.%v", svcName, gp.namespace)
	} else if gp.useIstio {
		svc := fission.GetFunctionIstioServiceName(m.Name, m.Namespace)
		svcHost = fmt.Sprintf("%v.%v:8888", svc, gp.namespace)
	} else {
		log.Printf("Using pod IP for specialized pod")
		svcHost = fmt.Sprintf("%v:8888", pod.Status.PodIP)
	}

	kubeObjRefs := []api.ObjectReference{
		{
			Kind:            "pod",
			Name:            pod.ObjectMeta.Name,
			APIVersion:      pod.TypeMeta.APIVersion,
			Namespace:       pod.ObjectMeta.Namespace,
			ResourceVersion: pod.ObjectMeta.ResourceVersion,
			UID:             pod.ObjectMeta.UID,
		},
	}

	fsvc := &fscache.FuncSvc{
		Name:              pod.ObjectMeta.Name,
		Function:          m,
		Environment:       gp.env,
		Address:           svcHost,
		KubernetesObjects: kubeObjRefs,
		Executor:          fscache.POOLMGR,
		Ctime:             time.Now(),
		Atime:             time.Now(),
	}

	_, err = gp.fsCache.Add(*fsvc)
	if err != nil {
		return nil, err
	}
	return fsvc, nil
}

// destroys the pool -- the deployment, replicaset and pods
func (gp *GenericPool) destroy() error {
	deletePropagation := metav1.DeletePropagationBackground
	delOpt := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}
	err := gp.kubernetesClient.ExtensionsV1beta1().
		Deployments(gp.namespace).Delete(gp.deployment.ObjectMeta.Name, &delOpt)
	if err != nil {
		log.Printf("Error destroying deployment: %v", err)
		return err
	}
	return nil
}
