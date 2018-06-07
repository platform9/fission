/*
Copyright 2017 The Fission Authors.

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

package buildermgr

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

type requestType int

const (
	GET_BUILDER requestType = iota
	CLEANUP_BUILDERS

	LABEL_ENV_NAME            = "envName"
	LABEL_ENV_NAMESPACE       = "envNamespace"
	LABEL_ENV_RESOURCEVERSION = "envResourceVersion"
	LABEL_DEPLOYMENT_OWNER    = "owner"
	BUILDER_MGR               = "buildermgr"
)

var (
	deletePropagation = metav1.DeletePropagationBackground
	delOpt            = metav1.DeleteOptions{PropagationPolicy: &deletePropagation}
)

type (
	builderInfo struct {
		envMetadata *metav1.ObjectMeta
		deployment  *v1beta1.Deployment
		service     *apiv1.Service
	}

	envwRequest struct {
		requestType
		env      *crd.Environment
		envList  []crd.Environment
		respChan chan envwResponse
	}

	envwResponse struct {
		builderInfo *builderInfo
		err         error
	}

	environmentWatcher struct {
		cache                  map[string]*builderInfo
		requestChan            chan envwRequest
		builderNamespace       string
		fissionClient          *crd.FissionClient
		kubernetesClient       *kubernetes.Clientset
		fetcherImage           string
		fetcherImagePullPolicy apiv1.PullPolicy
		useIstio               bool
	}
)

func makeEnvironmentWatcher(fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, builderNamespace string) *environmentWatcher {

	useIstio := false
	enableIstio := os.Getenv("ENABLE_ISTIO")
	if len(enableIstio) > 0 {
		istio, err := strconv.ParseBool(enableIstio)
		if err != nil {
			log.Println("Failed to parse ENABLE_ISTIO, defaults to false")
		}
		useIstio = istio
	}

	fetcherImage := os.Getenv("FETCHER_IMAGE")
	if len(fetcherImage) == 0 {
		fetcherImage = "fission/fetcher"
	}

	fetcherImagePullPolicy := os.Getenv("FETCHER_IMAGE_PULL_POLICY")
	if len(fetcherImagePullPolicy) == 0 {
		fetcherImagePullPolicy = "IfNotPresent"
	}

	var pullPolicy apiv1.PullPolicy
	switch fetcherImagePullPolicy {
	case "Always":
		pullPolicy = apiv1.PullAlways
	case "Never":
		pullPolicy = apiv1.PullNever
	default:
		pullPolicy = apiv1.PullIfNotPresent
	}

	envWatcher := &environmentWatcher{
		cache:                  make(map[string]*builderInfo),
		requestChan:            make(chan envwRequest),
		builderNamespace:       builderNamespace,
		fissionClient:          fissionClient,
		kubernetesClient:       kubernetesClient,
		fetcherImage:           fetcherImage,
		fetcherImagePullPolicy: pullPolicy,
		useIstio:               useIstio,
	}

	go envWatcher.service()

	return envWatcher
}

func (envw *environmentWatcher) getCacheKey(envName string, envNamespace string, envResourceVersion string) string {
	return fmt.Sprintf("%v-%v-%v", envName, envNamespace, envResourceVersion)
}

func (env *environmentWatcher) getLabelForDeploymentOwner() map[string]string {
	return map[string]string{
		LABEL_DEPLOYMENT_OWNER: BUILDER_MGR,
	}
}

func (envw *environmentWatcher) getLabels(envName string, envNamespace string, envResourceVersion string) map[string]string {
	return map[string]string{
		LABEL_ENV_NAME:            envName,
		LABEL_ENV_NAMESPACE:       envNamespace,
		LABEL_ENV_RESOURCEVERSION: envResourceVersion,
		LABEL_DEPLOYMENT_OWNER:    BUILDER_MGR,
	}
}

func (envw *environmentWatcher) watchEnvironments() {
	rv := ""
	for {
		wi, err := envw.fissionClient.Environments(metav1.NamespaceAll).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			if fission.IsNetworkError(err) {
				log.Printf("Encounter network error, retrying later: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Fatalf("Error watching environment list: %v", err)
		}

		for {
			ev, more := <-wi.ResultChan()
			if !more {
				// restart watch from last rv
				break
			}
			if ev.Type == watch.Error {
				// restart watch from the start
				rv = ""
				time.Sleep(time.Second)
				break
			}
			env := ev.Object.(*crd.Environment)
			rv = env.Metadata.ResourceVersion
			envw.sync()
		}
	}
}

func (envw *environmentWatcher) sync() {
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		envList, err := envw.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if fission.IsNetworkError(err) {
				log.Printf("Error syncing environment CRD resources due to network error, retrying later: %v", err)
				time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
				continue
			}
			log.Fatalf("Error syncing environment CRD resources: %v", err)
		}

		// Create environment builders for all environments
		for i := range envList.Items {
			env := envList.Items[i]

			if env.Spec.Version == 1 || // builder is not supported with v1 interface
				len(env.Spec.Builder.Image) == 0 { // ignore env without builder image
				continue
			}
			_, err := envw.getEnvBuilder(&env)
			if err != nil {
				log.Printf("Error creating builder for %v: %v", env.Metadata.Name, err)
			}
		}

		// Remove environment builders no longer needed
		envw.cleanupEnvBuilders(envList.Items)
		break
	}
}

func (envw *environmentWatcher) service() {
	for {
		req := <-envw.requestChan
		switch req.requestType {
		case GET_BUILDER:
			// In order to support backward compatibility, for all environments with builder image created in default env,
			// the pods will be created in fission-builder namespace
			ns := envw.builderNamespace
			if req.env.Metadata.Namespace != metav1.NamespaceDefault {
				ns = req.env.Metadata.Namespace
			}

			key := envw.getCacheKey(req.env.Metadata.Name, ns, req.env.Metadata.ResourceVersion)
			builderInfo, ok := envw.cache[key]
			if !ok {
				builderInfo, err := envw.createBuilder(req.env, ns)
				if err != nil {
					req.respChan <- envwResponse{err: err}
					continue
				}
				envw.cache[key] = builderInfo
			}
			req.respChan <- envwResponse{builderInfo: builderInfo}

		case CLEANUP_BUILDERS:
			latestEnvList := make(map[string]*crd.Environment)
			for i := range req.envList {
				env := req.envList[i]
				// In order to support backward compatibility, for all builder images created in default
				// env, the pods are created in fission-builder namespace
				ns := envw.builderNamespace
				if env.Metadata.Namespace != metav1.NamespaceDefault {
					ns = env.Metadata.Namespace
				}
				key := envw.getCacheKey(env.Metadata.Name, ns, env.Metadata.ResourceVersion)
				latestEnvList[key] = &env
			}

			// If an environment is deleted when builder manager down,
			// the builder belongs to the environment will be out-of-
			// control (an orphan builder) since there is no record in
			// cache and CRD. We need to iterate over the services &
			// deployments to remove both normal and orphan builders.

			svcList, err := envw.getBuilderServiceList(envw.getLabelForDeploymentOwner(), metav1.NamespaceAll)
			if err != nil {
				log.Println(err.Error())
			}
			for _, svc := range svcList {
				envName := svc.ObjectMeta.Labels[LABEL_ENV_NAME]
				envNamespace := svc.ObjectMeta.Labels[LABEL_ENV_NAMESPACE]
				envResourceVersion := svc.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION]
				key := envw.getCacheKey(envName, envNamespace, envResourceVersion)
				if _, ok := latestEnvList[key]; !ok {
					err := envw.deleteBuilderServiceByName(svc.ObjectMeta.Name, svc.ObjectMeta.Namespace)
					if err != nil {
						log.Printf("Error removing builder service: %v", err)
					}
				}
				delete(envw.cache, key)
			}

			deployList, err := envw.getBuilderDeploymentList(envw.getLabelForDeploymentOwner(), metav1.NamespaceAll)
			if err != nil {
				log.Printf(err.Error())
			}
			for _, deploy := range deployList {
				envName := deploy.ObjectMeta.Labels[LABEL_ENV_NAME]
				envNamespace := deploy.ObjectMeta.Labels[LABEL_ENV_NAMESPACE]
				envResourceVersion := deploy.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION]
				key := envw.getCacheKey(envName, envNamespace, envResourceVersion)
				if _, ok := latestEnvList[key]; !ok {
					err := envw.deleteBuilderDeploymentByName(deploy.ObjectMeta.Name, deploy.ObjectMeta.Namespace)
					if err != nil {
						log.Printf("Error removing builder deployment: %v", err)
					}
				}
				delete(envw.cache, key)
			}
		}
	}
}

func (envw *environmentWatcher) getEnvBuilder(env *crd.Environment) (*builderInfo, error) {
	respChan := make(chan envwResponse)
	envw.requestChan <- envwRequest{
		requestType: GET_BUILDER,
		env:         env,
		respChan:    respChan,
	}
	resp := <-respChan
	return resp.builderInfo, resp.err
}

func (envw *environmentWatcher) cleanupEnvBuilders(envs []crd.Environment) {
	envw.requestChan <- envwRequest{
		requestType: CLEANUP_BUILDERS,
		envList:     envs,
	}
}

func (envw *environmentWatcher) createBuilder(env *crd.Environment, ns string) (*builderInfo, error) {
	var svc *apiv1.Service
	var deploy *v1beta1.Deployment

	sel := envw.getLabels(env.Metadata.Name, ns, env.Metadata.ResourceVersion)

	svcList, err := envw.getBuilderServiceList(sel, ns)
	if err != nil {
		return nil, err
	}
	// there should be only one service in svcList
	if len(svcList) == 0 {
		svc, err = envw.createBuilderService(env, ns)
		if err != nil {
			return nil, fmt.Errorf("Error creating builder service: %v", err)
		}
	} else if len(svcList) == 1 {
		svc = &svcList[0]
	} else {
		return nil, fmt.Errorf("Found more than one builder service for environment %v", env.Metadata.Name)
	}

	deployList, err := envw.getBuilderDeploymentList(sel, ns)
	if err != nil {
		return nil, err
	}
	// there should be only one deploy in deployList
	if len(deployList) == 0 {
		// create builder SA in this ns, if not already created
		_, err := fission.SetupSA(envw.kubernetesClient, fission.FissionBuilderSA, ns)
		if err != nil {
			log.Printf("Error : %v creating %s in ns : %s", err, fission.FissionBuilderSA, ns)
			return nil, err
		}

		deploy, err = envw.createBuilderDeployment(env, ns)
		if err != nil {
			return nil, fmt.Errorf("Error creating builder deployment: %v", err)
		}
	} else if len(deployList) == 1 {
		deploy = &deployList[0]
	} else {
		return nil, fmt.Errorf("Found more than one builder deployment for environment %v", env.Metadata.Name)
	}

	return &builderInfo{
		envMetadata: &env.Metadata,
		service:     svc,
		deployment:  deploy,
	}, nil
}

func (envw *environmentWatcher) deleteBuilderServiceByName(name, namespace string) error {
	err := envw.kubernetesClient.CoreV1().
		Services(namespace).
		Delete(name, &delOpt)
	if err != nil {
		return fmt.Errorf("Error deleting builder service %s.%s: %v", name, namespace, err)
	}
	return nil
}

func (envw *environmentWatcher) deleteBuilderDeploymentByName(name, namespace string) error {
	err := envw.kubernetesClient.ExtensionsV1beta1().
		Deployments(namespace).
		Delete(name, &delOpt)
	if err != nil {
		return fmt.Errorf("Error deleting builder deployment %s.%s: %v", name, namespace, err)
	}
	return nil
}

func (envw *environmentWatcher) deleteBuilderService(sel map[string]string, ns string) error {
	svcList, err := envw.getBuilderServiceList(sel, ns)
	if err != nil {
		return err
	}
	for _, svc := range svcList {
		log.Printf("Removing builder service: %v", svc.ObjectMeta.Name)
		err = envw.kubernetesClient.CoreV1().
			Services(ns).
			Delete(svc.ObjectMeta.Name, &delOpt)
		if err != nil {
			return fmt.Errorf("Error deleting builder service: %v", err)
		}
	}
	return nil
}

func (envw *environmentWatcher) deleteBuilderDeployment(sel map[string]string, ns string) error {
	deployList, err := envw.getBuilderDeploymentList(sel, ns)
	if err != nil {
		return err
	}
	for _, deploy := range deployList {
		log.Printf("Removing builder deployment: %v", deploy.ObjectMeta.Name)
		err = envw.kubernetesClient.ExtensionsV1beta1().
			Deployments(ns).
			Delete(deploy.ObjectMeta.Name, &delOpt)
		if err != nil {
			return fmt.Errorf("Error deleteing builder deployment: %v", err)
		}
	}
	return nil
}

func (envw *environmentWatcher) getBuilderServiceList(sel map[string]string, ns string) ([]apiv1.Service, error) {
	svcList, err := envw.kubernetesClient.CoreV1().Services(ns).List(
		metav1.ListOptions{
			LabelSelector: labels.Set(sel).AsSelector().String(),
		})
	if err != nil {
		return nil, fmt.Errorf("Error getting builder service list: %v", err)
	}
	return svcList.Items, nil
}

func (envw *environmentWatcher) createBuilderService(env *crd.Environment, ns string) (*apiv1.Service, error) {
	name := fmt.Sprintf("%v-%v", env.Metadata.Name, env.Metadata.ResourceVersion)
	sel := envw.getLabels(env.Metadata.Name, ns, env.Metadata.ResourceVersion)
	service := apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    sel,
		},
		Spec: apiv1.ServiceSpec{
			Selector: sel,
			Type:     apiv1.ServiceTypeClusterIP,
			Ports: []apiv1.ServicePort{
				{
					Name:     "fetcher-port",
					Protocol: apiv1.ProtocolTCP,
					Port:     8000,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8000,
					},
				},
				{
					Name:     "builder-port",
					Protocol: apiv1.ProtocolTCP,
					Port:     8001,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8001,
					},
				},
			},
		},
	}
	log.Printf("Creating builder service: %v", name)
	_, err := envw.kubernetesClient.CoreV1().Services(ns).Create(&service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func (envw *environmentWatcher) getBuilderDeploymentList(sel map[string]string, ns string) ([]v1beta1.Deployment, error) {
	deployList, err := envw.kubernetesClient.ExtensionsV1beta1().Deployments(ns).List(
		metav1.ListOptions{
			LabelSelector: labels.Set(sel).AsSelector().String(),
		})
	if err != nil {
		return nil, fmt.Errorf("Error getting builder deployment list: %v", err)
	}
	return deployList.Items, nil
}

func (envw *environmentWatcher) createBuilderDeployment(env *crd.Environment, ns string) (*v1beta1.Deployment, error) {
	sharedMountPath := "/packages"
	sharedCfgMapPath := "/configs"
	sharedSecretPath := "/secrets"
	name := fmt.Sprintf("%v-%v", env.Metadata.Name, env.Metadata.ResourceVersion)
	sel := envw.getLabels(env.Metadata.Name, ns, env.Metadata.ResourceVersion)
	var replicas int32 = 1

	podAnnotation := env.Spec.Runtime.Annotations
	if podAnnotation == nil {
		podAnnotation = make(map[string]string)
	}
	if envw.useIstio && env.Spec.AllowAccessToExternalNetwork {
		podAnnotation["sidecar.istio.io/inject"] = "false"
	}

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    sel,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: sel,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      sel,
					Annotations: podAnnotation,
				},
				Spec: apiv1.PodSpec{
					Volumes: []apiv1.Volume{
						{
							Name: fission.SharedVolumePackages,
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
							Name:                   "builder",
							Image:                  env.Spec.Builder.Image,
							ImagePullPolicy:        apiv1.PullAlways,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumePackages,
									MountPath: sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: sharedCfgMapPath,
								},
							},
							Command: []string{"/builder", sharedMountPath},
							ReadinessProbe: &apiv1.Probe{
								InitialDelaySeconds: 5,
								PeriodSeconds:       2,
								Handler: apiv1.Handler{
									HTTPGet: &apiv1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8001,
										},
									},
								},
							},
						}, env.Spec.Builder.Container),
						{
							Name:                   "fetcher",
							Image:                  envw.fetcherImage,
							ImagePullPolicy:        envw.fetcherImagePullPolicy,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumePackages,
									MountPath: sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: sharedCfgMapPath,
								},
							},
							Command: []string{"/fetcher",
								"-secret-dir", sharedSecretPath,
								"-cfgmap-dir", sharedCfgMapPath,
								sharedMountPath},
							ReadinessProbe: &apiv1.Probe{
								InitialDelaySeconds: 5,
								PeriodSeconds:       2,
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
					ServiceAccountName: "fission-builder",
				},
			},
		},
	}
	log.Printf("Creating builder deployment: %v", fmt.Sprintf("%v-%v", env.Metadata.Name, env.Metadata.ResourceVersion))
	_, err := envw.kubernetesClient.ExtensionsV1beta1().Deployments(ns).Create(deployment)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}
