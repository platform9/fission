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
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	fetcherClient "github.com/fission/fission/environments/fetcher/client"
	fetcherConfig "github.com/fission/fission/environments/fetcher/config"
	"github.com/fission/fission/executor/fscache"
)

type (
	GenericPool struct {
		logger                 *zap.Logger
		env                    *crd.Environment
		replicas               int32                         // num idle pods
		deployment             *v1beta1.Deployment           // kubernetes deployment
		namespace              string                        // namespace to keep our resources
		functionNamespace      string                        // fallback namespace for fission functions
		podReadyTimeout        time.Duration                 // timeout for generic pods to become ready
		idlePodReapTime        time.Duration                 // pods unused for idlePodReapTime are deleted
		fsCache                *fscache.FunctionServiceCache // cache funcSvc's by function, address and podname
		useSvc                 bool                          // create k8s service for specialized pods
		useIstio               bool
		poolInstanceId         string           // small random string to uniquify pod names
		runtimeImagePullPolicy apiv1.PullPolicy // pull policy for generic pool to created env deployment
		kubernetesClient       *kubernetes.Clientset
		fissionClient          *crd.FissionClient
		instanceId             string // poolmgr instance id
		labelsForPool          map[string]string
		requestChannel         chan *choosePodRequest
		fetcherConfig          *fetcherConfig.Config
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

func MakeGenericPool(
	logger *zap.Logger,
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	env *crd.Environment,
	initialReplicas int32,
	namespace string,
	functionNamespace string,
	fsCache *fscache.FunctionServiceCache,
	fetcherConfig *fetcherConfig.Config,
	instanceId string,
	enableIstio bool) (*GenericPool, error) {

	gpLogger := logger.Named("generic_pool")

	gpLogger.Info("creating pool", zap.Any("environment", env.Metadata))

	// TODO: in general we need to provide the user a way to configure pools.  Initial
	// replicas, autoscaling params, various timeouts, etc.
	gp := &GenericPool{
		logger:            gpLogger,
		env:               env,
		replicas:          initialReplicas, // TODO make this an env param instead?
		requestChannel:    make(chan *choosePodRequest),
		fissionClient:     fissionClient,
		kubernetesClient:  kubernetesClient,
		namespace:         namespace,
		functionNamespace: functionNamespace,
		podReadyTimeout:   5 * time.Minute, // TODO make this an env param?
		idlePodReapTime:   3 * time.Minute, // TODO make this configurable
		fsCache:           fsCache,
		poolInstanceId:    uniuri.NewLen(8),
		fetcherConfig:     fetcherConfig,
		instanceId:        instanceId,
		useSvc:            false,       // defaults off -- svc takes a second or more to become routable, slowing cold start
		useIstio:          enableIstio, // defaults off -- istio integration requires pod relabeling and it takes a second or more to become routable, slowing cold start
	}

	gp.runtimeImagePullPolicy = fission.GetImagePullPolicy(os.Getenv("RUNTIME_IMAGE_PULL_POLICY"))

	// create fetcher SA in this ns, if not already created
	err := fetcherConfig.SetupServiceAccount(gp.kubernetesClient, gp.namespace, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating fetcher service account in namespace %q", gp.namespace)
	}

	// Labels for generic deployment/RS/pods.
	gp.labelsForPool = gp.getDeployLabels()

	// create the pool
	err = gp.createPool()
	if err != nil {
		return nil, err
	}
	gpLogger.Info("deployment created", zap.Any("environment", env.Metadata))

	go gp.choosePodService()

	return gp, nil
}

func (gp *GenericPool) getDeployLabels() map[string]string {
	return map[string]string{
		fission.EXECUTOR_INSTANCEID_LABEL: gp.instanceId,
		fission.EXECUTOR_TYPE:             fission.ExecutorTypePoolmgr,
		fission.ENVIRONMENT_NAME:          gp.env.Metadata.Name,
		fission.ENVIRONMENT_NAMESPACE:     gp.env.Metadata.Namespace,
		fission.ENVIRONMENT_UID:           string(gp.env.Metadata.UID),
		"managed":                         "true", // this allows us to easily find pods managed by the deployment
	}
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
			gp.logger.Error("timed out waiting for pod", zap.Any("labels", newLabels), zap.Duration("timeout", gp.podReadyTimeout))
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

			// Ignore not ready pod here
			if !fission.IsReadyPod(&pod) {
				continue
			}

			// add it to the list of ready pods
			readyPods = append(readyPods, &pod)
		}
		gp.logger.Info("found ready pods",
			zap.Any("labels", newLabels),
			zap.Int("ready_count", len(readyPods)),
			zap.Int("total", len(podList.Items)))

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
			gp.logger.Info("relabeling pod", zap.String("pod", chosenPod.ObjectMeta.Name))
			_, err = gp.kubernetesClient.CoreV1().Pods(gp.namespace).Update(chosenPod)
			if err != nil {
				gp.logger.Error("failed to relabel pod", zap.Error(err), zap.String("pod", chosenPod.ObjectMeta.Name))
				continue
			}
		}
		gp.logger.Info("chose pod", zap.String("pod", chosenPod.ObjectMeta.Name), zap.Duration("elapsed_time", time.Since(startTime)))
		return chosenPod, nil
	}
}

func (gp *GenericPool) labelsForFunction(metadata *metav1.ObjectMeta) map[string]string {
	label := gp.getDeployLabels()
	label[fission.FUNCTION_NAME] = metadata.Name
	label[fission.FUNCTION_UID] = string(metadata.UID)
	label[fission.FUNCTION_NAMESPACE] = metadata.Namespace // function CRD must stay within same namespace of environment CRD
	label["managed"] = "false"                             // this allows us to easily find pods not managed by the deployment
	return label

}

func (gp *GenericPool) scheduleDeletePod(name string) {
	go func() {
		// The sleep allows debugging or collecting logs from the pod before it's
		// cleaned up.  (We need a better solutions for both those things; log
		// aggregation and storage will help.)
		gp.logger.Error("error in pod - scheduling cleanup", zap.String("pod", name))
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

func (gp *GenericPool) getSpecializeUrl(podIP string) string {
	testUrl := os.Getenv("TEST_SPECIALIZE_URL")
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

// specializePod chooses a pod, copies the required user-defined function to that pod
// (via fetcher), and calls the function-run container to load it, resulting in a
// specialized pod.
func (gp *GenericPool) specializePod(ctx context.Context, pod *apiv1.Pod, metadata *metav1.ObjectMeta) error {
	// for fetcher we don't need to create a service, just talk to the pod directly
	podIP := pod.Status.PodIP
	if len(podIP) == 0 {
		return errors.Errorf("Pod %s in namespace %s has no IP", pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
	}
	// specialize pod with service
	if gp.useIstio {
		svc := fission.GetFunctionIstioServiceName(metadata.Name, metadata.Namespace)
		podIP = fmt.Sprintf("%v.%v", svc, gp.namespace)
	}

	// tell fetcher to get the function.
	fetcherUrl := gp.getSpecializeUrl(podIP)
	gp.logger.Info("calling fetcher to copy function", zap.String("function", metadata.Name), zap.String("url", fetcherUrl))

	fn, err := gp.fissionClient.
		Functions(metadata.Namespace).
		Get(metadata.Name)
	if err != nil {
		return err
	}

	specializeReq := gp.fetcherConfig.NewSpecializeRequest(fn, gp.env)

	gp.logger.Info("specializing pod", zap.String("function", metadata.Name))

	err = fetcherClient.MakeClient(gp.logger, fetcherUrl).Specialize(ctx, &specializeReq)
	if err != nil {
		return err
	}

	return nil
}

// getPoolName returns a unique name of an environment
func (gp *GenericPool) getPoolName() string {
	return strings.ToLower(fmt.Sprintf("poolmgr-%v-%v-%v", gp.env.Metadata.Name, gp.env.Metadata.Namespace, uniuri.NewLen(8)))
}

// A pool is a deployment of generic containers for an env.  This
// creates the pool but doesn't wait for any pods to be ready.
func (gp *GenericPool) createPool() error {
	// Use long terminationGracePeriodSeconds for connection draining in case that
	// pod still runs user functions.
	gracePeriodSeconds := int64(6 * 60)
	if gp.env.Spec.TerminationGracePeriod > 0 {
		gracePeriodSeconds = gp.env.Spec.TerminationGracePeriod
	}

	podAnnotations := gp.env.Metadata.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}
	if gp.useIstio && gp.env.Spec.AllowAccessToExternalNetwork {
		podAnnotations["sidecar.istio.io/inject"] = "false"
	}

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   gp.getPoolName(),
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
					Annotations: podAnnotations,
				},
				Spec: fission.MergePodSpecs(&apiv1.PodSpec{
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

					Containers: []apiv1.Container{fission.MergeContainerSpecs(&apiv1.Container{
						Name:                   gp.env.Metadata.Name,
						Image:                  gp.env.Spec.Runtime.Image,
						ImagePullPolicy:        gp.runtimeImagePullPolicy,
						TerminationMessagePath: "/dev/termination-log",
						Resources:              gp.env.Spec.Resources,
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
					},
					ServiceAccountName: "fission-fetcher",
					// TerminationGracePeriodSeconds should be equal to the
					// sleep time of preStop to make sure that SIGTERM is sent
					// to pod after 6 mins.
					TerminationGracePeriodSeconds: &gracePeriodSeconds,
				}, gp.env.Spec.Runtime.PodSpec),
			},
		},
	}

	err := gp.fetcherConfig.AddFetcherToPodSpec(&deployment.Spec.Template.Spec, gp.env.Metadata.Name)
	if err != nil {
		return err
	}

	depl, err := gp.kubernetesClient.ExtensionsV1beta1().Deployments(gp.namespace).Create(deployment)
	if err != nil {
		gp.logger.Error("error creating deployment in kubernetes", zap.Error(err), zap.String("deployment", deployment.Name))
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
			e := "error waiting for ready pod for deployment"
			gp.logger.Error(e, zap.String("deployment", gp.deployment.ObjectMeta.Name), zap.String("namespace", gp.namespace))
			return fmt.Errorf("%s %q in namespace %q", e, gp.deployment.ObjectMeta.Name, gp.namespace)
		}

		gp.deployment = depl
		if gp.deployment.Status.AvailableReplicas > 0 {
			return nil
		}

		if time.Since(startTime) > gp.podReadyTimeout {
			podList, err := gp.kubernetesClient.CoreV1().Pods(gp.namespace).List(metav1.ListOptions{
				LabelSelector: labels.Set(
					gp.deployment.Spec.Selector.MatchLabels).AsSelector().String(),
			})
			if err != nil {
				gp.logger.Error("error getting pod list after timeout waiting for ready pod", zap.Error(err))
			}

			// Since even single pod is not ready, choosing the first pod to inspect is a good approximation. In future this can be done better
			pod := podList.Items[0]
			var multierr *multierror.Error
			for _, cStatus := range pod.Status.ContainerStatuses {
				if cStatus.Ready != true {
					multierr = multierror.Append(multierr, errors.New(fmt.Sprintf("%v: %v", cStatus.State.Waiting.Reason, cStatus.State.Waiting.Message)))
				}
			}
			return errors.Wrapf(multierr, "Timeout: waited too long for pod of deployment %v in namespace %v to be ready",
				gp.deployment.ObjectMeta.Name, gp.namespace)
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

func (gp *GenericPool) GetFuncSvc(ctx context.Context, m *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	gp.logger.Info("choosing pod from pool", zap.String("function", m.Name))
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

	err = gp.specializePod(ctx, pod, m)
	if err != nil {
		gp.scheduleDeletePod(pod.ObjectMeta.Name)
		return nil, err
	}
	gp.logger.Info("specialized pod", zap.String("pod", pod.ObjectMeta.Name), zap.String("function", m.Name))

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
			return nil, errors.Errorf("sanity check failed for svc %v", svc.ObjectMeta.Name)
		}

		// the fission router isn't in the same namespace, so return a
		// namespace-qualified hostname
		svcHost = fmt.Sprintf("%v.%v", svcName, gp.namespace)
	} else if gp.useIstio {
		svc := fission.GetFunctionIstioServiceName(m.Name, m.Namespace)
		svcHost = fmt.Sprintf("%v.%v:8888", svc, gp.namespace)
	} else {
		gp.logger.Info("using pod IP for specialized pod", zap.String("pod", pod.ObjectMeta.Name), zap.String("function", m.Name))
		svcHost = fmt.Sprintf("%v:8888", pod.Status.PodIP)
	}

	kubeObjRefs := []apiv1.ObjectReference{
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
		gp.logger.Error("error destroying deployment",
			zap.Error(err),
			zap.String("deployment_name", gp.deployment.ObjectMeta.Name),
			zap.String("deployment_namespace", gp.namespace))
		return err
	}
	return nil
}
