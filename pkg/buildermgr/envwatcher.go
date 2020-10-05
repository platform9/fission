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
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/util"
	fetcherConfig "github.com/fission/fission/pkg/fetcher/config"
	"github.com/fission/fission/pkg/utils"
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
		deployment  *appsv1.Deployment
		service     *apiv1.Service
	}

	envwRequest struct {
		requestType
		env      *fv1.Environment
		envList  []fv1.Environment
		respChan chan envwResponse
	}

	envwResponse struct {
		builderInfo *builderInfo
		err         error
	}

	environmentWatcher struct {
		logger                 *zap.Logger
		cache                  map[string]*builderInfo
		requestChan            chan envwRequest
		builderNamespace       string
		fissionClient          *crd.FissionClient
		kubernetesClient       *kubernetes.Clientset
		fetcherConfig          *fetcherConfig.Config
		builderImagePullPolicy apiv1.PullPolicy
		useIstio               bool
	}
)

func makeEnvironmentWatcher(
	logger *zap.Logger,
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	fetcherConfig *fetcherConfig.Config,
	builderNamespace string) *environmentWatcher {

	useIstio := false
	enableIstio := os.Getenv("ENABLE_ISTIO")
	if len(enableIstio) > 0 {
		istio, err := strconv.ParseBool(enableIstio)
		if err != nil {
			logger.Error("Failed to parse ENABLE_ISTIO, defaults to false")
		}
		useIstio = istio
	}

	builderImagePullPolicy := utils.GetImagePullPolicy(os.Getenv("BUILDER_IMAGE_PULL_POLICY"))

	envWatcher := &environmentWatcher{
		logger:                 logger.Named("environment_watcher"),
		cache:                  make(map[string]*builderInfo),
		requestChan:            make(chan envwRequest),
		builderNamespace:       builderNamespace,
		fissionClient:          fissionClient,
		kubernetesClient:       kubernetesClient,
		builderImagePullPolicy: builderImagePullPolicy,
		useIstio:               useIstio,
		fetcherConfig:          fetcherConfig,
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
		wi, err := envw.fissionClient.CoreV1().Environments(metav1.NamespaceAll).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			if utils.IsNetworkError(err) {
				envw.logger.Error("encountered network error, retrying later", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			envw.logger.Fatal("error watching environment list", zap.Error(err))
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
			env := ev.Object.(*fv1.Environment)
			rv = env.ObjectMeta.ResourceVersion
			envw.sync()
		}
	}
}

func (envw *environmentWatcher) sync() {
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		envList, err := envw.fissionClient.CoreV1().Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if utils.IsNetworkError(err) {
				envw.logger.Error("error syncing environment CRD resources due to network error, retrying later", zap.Error(err))
				time.Sleep(50 * time.Duration(2*i) * time.Millisecond)
				continue
			}
			envw.logger.Fatal("error syncing environment CRD resources", zap.Error(err))
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
				envw.logger.Error("error creating builder", zap.Error(err), zap.String("builder_target", env.ObjectMeta.Name))
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
			if req.env.ObjectMeta.Namespace != metav1.NamespaceDefault {
				ns = req.env.ObjectMeta.Namespace
			}

			key := envw.getCacheKey(req.env.ObjectMeta.Name, ns, req.env.ObjectMeta.ResourceVersion)
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
			latestEnvList := make(map[string]*fv1.Environment)
			for i := range req.envList {
				env := req.envList[i]
				// In order to support backward compatibility, for all builder images created in default
				// env, the pods are created in fission-builder namespace
				ns := envw.builderNamespace
				if env.ObjectMeta.Namespace != metav1.NamespaceDefault {
					ns = env.ObjectMeta.Namespace
				}
				key := envw.getCacheKey(env.ObjectMeta.Name, ns, env.ObjectMeta.ResourceVersion)
				latestEnvList[key] = &env
			}

			// If an environment is deleted when builder manager down,
			// the builder belongs to the environment will be out-of-
			// control (an orphan builder) since there is no record in
			// cache and CRD. We need to iterate over the services &
			// deployments to remove both normal and orphan builders.

			svcList, err := envw.getBuilderServiceList(envw.getLabelForDeploymentOwner(), metav1.NamespaceAll)
			if err != nil {
				envw.logger.Error("error getting the builder service list", zap.Error(err))
			}
			for _, svc := range svcList {
				envName := svc.ObjectMeta.Labels[LABEL_ENV_NAME]
				envNamespace := svc.ObjectMeta.Labels[LABEL_ENV_NAMESPACE]
				envResourceVersion := svc.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION]
				key := envw.getCacheKey(envName, envNamespace, envResourceVersion)
				if _, ok := latestEnvList[key]; !ok {
					err := envw.deleteBuilderServiceByName(svc.ObjectMeta.Name, svc.ObjectMeta.Namespace)
					if err != nil {
						envw.logger.Error("error removing builder service", zap.Error(err),
							zap.String("service_name", svc.ObjectMeta.Name),
							zap.String("service_namespace", svc.ObjectMeta.Namespace))
					}
				}
				delete(envw.cache, key)
			}

			deployList, err := envw.getBuilderDeploymentList(envw.getLabelForDeploymentOwner(), metav1.NamespaceAll)
			if err != nil {
				envw.logger.Error("error getting the builder deployment list", zap.Error(err))
			}
			for _, deploy := range deployList {
				envName := deploy.ObjectMeta.Labels[LABEL_ENV_NAME]
				envNamespace := deploy.ObjectMeta.Labels[LABEL_ENV_NAMESPACE]
				envResourceVersion := deploy.ObjectMeta.Labels[LABEL_ENV_RESOURCEVERSION]
				key := envw.getCacheKey(envName, envNamespace, envResourceVersion)
				if _, ok := latestEnvList[key]; !ok {
					err := envw.deleteBuilderDeploymentByName(deploy.ObjectMeta.Name, deploy.ObjectMeta.Namespace)
					if err != nil {
						envw.logger.Error("error removing builder deployment", zap.Error(err),
							zap.String("deployment_name", deploy.ObjectMeta.Name),
							zap.String("deployment_namespace", deploy.ObjectMeta.Namespace))
					}
				}
				delete(envw.cache, key)
			}
		}
	}
}

func (envw *environmentWatcher) getEnvBuilder(env *fv1.Environment) (*builderInfo, error) {
	respChan := make(chan envwResponse)
	envw.requestChan <- envwRequest{
		requestType: GET_BUILDER,
		env:         env,
		respChan:    respChan,
	}
	resp := <-respChan
	return resp.builderInfo, resp.err
}

func (envw *environmentWatcher) cleanupEnvBuilders(envs []fv1.Environment) {
	envw.requestChan <- envwRequest{
		requestType: CLEANUP_BUILDERS,
		envList:     envs,
	}
}

func (envw *environmentWatcher) createBuilder(env *fv1.Environment, ns string) (*builderInfo, error) {
	var svc *apiv1.Service
	var deploy *appsv1.Deployment

	sel := envw.getLabels(env.ObjectMeta.Name, ns, env.ObjectMeta.ResourceVersion)

	svcList, err := envw.getBuilderServiceList(sel, ns)
	if err != nil {
		return nil, err
	}
	// there should be only one service in svcList
	if len(svcList) == 0 {
		svc, err = envw.createBuilderService(env, ns)
		if err != nil {
			return nil, errors.Wrap(err, "error creating builder service")
		}
	} else if len(svcList) == 1 {
		svc = &svcList[0]
	} else {
		return nil, fmt.Errorf("found more than one builder service for environment %q", env.ObjectMeta.Name)
	}

	deployList, err := envw.getBuilderDeploymentList(sel, ns)
	if err != nil {
		return nil, err
	}
	// there should be only one deploy in deployList
	if len(deployList) == 0 {
		// create builder SA in this ns, if not already created
		_, err := utils.SetupSA(envw.kubernetesClient, fv1.FissionBuilderSA, ns)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating %q in ns: %s", fv1.FissionBuilderSA, ns)
		}

		deploy, err = envw.createBuilderDeployment(env, ns)
		if err != nil {
			return nil, errors.Wrap(err, "error creating builder deployment")
		}
	} else if len(deployList) == 1 {
		deploy = &deployList[0]
	} else {
		return nil, fmt.Errorf("found more than one builder deployment for environment %q", env.ObjectMeta.Name)
	}

	return &builderInfo{
		envMetadata: &env.ObjectMeta,
		service:     svc,
		deployment:  deploy,
	}, nil
}

func (envw *environmentWatcher) deleteBuilderServiceByName(name, namespace string) error {
	err := envw.kubernetesClient.CoreV1().
		Services(namespace).
		Delete(context.Background(), name, delOpt)
	if err != nil {
		return errors.Wrapf(err, "error deleting builder service %s.%s", name, namespace)
	}
	return nil
}

func (envw *environmentWatcher) deleteBuilderDeploymentByName(name, namespace string) error {
	err := envw.kubernetesClient.AppsV1().
		Deployments(namespace).
		Delete(context.Background(), name, delOpt)
	if err != nil {
		return errors.Wrapf(err, "error deleting builder deployment %s.%s", name, namespace)
	}
	return nil
}

func (envw *environmentWatcher) getBuilderServiceList(sel map[string]string, ns string) ([]apiv1.Service, error) {
	svcList, err := envw.kubernetesClient.CoreV1().Services(ns).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labels.Set(sel).AsSelector().String(),
		})
	if err != nil {
		return nil, errors.Wrap(err, "error getting builder service list")
	}
	return svcList.Items, nil
}

func (envw *environmentWatcher) createBuilderService(env *fv1.Environment, ns string) (*apiv1.Service, error) {
	name := fmt.Sprintf("%v-%v", env.ObjectMeta.Name, env.ObjectMeta.ResourceVersion)
	sel := envw.getLabels(env.ObjectMeta.Name, ns, env.ObjectMeta.ResourceVersion)
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
	envw.logger.Info("creating builder service", zap.String("service_name", name))
	_, err := envw.kubernetesClient.CoreV1().Services(ns).Create(context.Background(), &service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func (envw *environmentWatcher) getBuilderDeploymentList(sel map[string]string, ns string) ([]appsv1.Deployment, error) {
	deployList, err := envw.kubernetesClient.AppsV1().Deployments(ns).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labels.Set(sel).AsSelector().String(),
		})
	if err != nil {
		return nil, errors.Wrap(err, "error getting builder deployment list")
	}
	return deployList.Items, nil
}

func (envw *environmentWatcher) createBuilderDeployment(env *fv1.Environment, ns string) (*appsv1.Deployment, error) {
	name := fmt.Sprintf("%v-%v", env.ObjectMeta.Name, env.ObjectMeta.ResourceVersion)
	sel := envw.getLabels(env.ObjectMeta.Name, ns, env.ObjectMeta.ResourceVersion)
	var replicas int32 = 1

	podAnnotations := env.ObjectMeta.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}
	if envw.useIstio && env.Spec.AllowAccessToExternalNetwork {
		podAnnotations["sidecar.istio.io/inject"] = "false"
	}

	container, err := util.MergeContainer(&apiv1.Container{
		Name:                   "builder",
		Image:                  env.Spec.Builder.Image,
		ImagePullPolicy:        envw.builderImagePullPolicy,
		TerminationMessagePath: "/dev/termination-log",
		Command:                []string{"/builder", envw.fetcherConfig.SharedMountPath()},
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
	}, env.Spec.Builder.Container)
	if err != nil {
		return nil, err
	}

	pod := apiv1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      sel,
			Annotations: podAnnotations,
		},
		Spec: apiv1.PodSpec{
			Containers:         []apiv1.Container{*container},
			ServiceAccountName: "fission-builder",
		},
	}

	pod.Spec = *(util.ApplyImagePullSecret(env.Spec.ImagePullSecret, pod.Spec))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    sel,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: sel,
			},
			Template: pod,
		},
	}

	err = envw.fetcherConfig.AddFetcherToPodSpec(&deployment.Spec.Template.Spec, "builder")
	if err != nil {
		return nil, err
	}

	if env.Spec.Builder.PodSpec != nil {
		newPodSpec, err := util.MergePodSpec(&deployment.Spec.Template.Spec, env.Spec.Builder.PodSpec)
		if err != nil {
			return nil, err
		}
		deployment.Spec.Template.Spec = *newPodSpec
	}

	_, err = envw.kubernetesClient.AppsV1().Deployments(ns).Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	envw.logger.Info("creating builder deployment", zap.String("deployment", name))

	return deployment, nil
}
