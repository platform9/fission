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

package newdeploy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	asv1 "k8s.io/api/autoscaling/v1"
	apiv1 "k8s.io/api/core/v1"
	k8s_err "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/executor/util"
	"github.com/fission/fission/pkg/utils"
)

// Deployment Constants
const (
	DeploymentKind    = "Deployment"
	DeploymentVersion = "apps/v1"
)

func (deploy *NewDeploy) createOrGetDeployment(fn *fv1.Function, env *fv1.Environment,
	deployName string, deployLabels map[string]string, deployAnnotations map[string]string, deployNamespace string) (*appsv1.Deployment, error) {

	specializationTimeout := fn.Spec.InvokeStrategy.ExecutionStrategy.SpecializationTimeout
	minScale := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)

	// Always scale to at least one pod when createOrGetDeployment
	// is called. The idleObjectReaper will scale-in the deployment
	// later if no requests to the function.
	if minScale <= 0 {
		minScale = 1
	}

	deployment, err := deploy.getDeploymentSpec(fn, env, &minScale, deployName, deployNamespace, deployLabels, deployAnnotations)
	if err != nil {
		return nil, err
	}

	existingDepl, err := deploy.kubernetesClient.AppsV1().Deployments(deployNamespace).Get(context.TODO(), deployName, metav1.GetOptions{})
	if err == nil {
		// Try to adopt orphan deployment created by the old executor.
		if existingDepl.Annotations[fv1.EXECUTOR_INSTANCEID_LABEL] != deploy.instanceID {
			existingDepl.Annotations = deployment.Annotations
			existingDepl.Labels = deployment.Labels
			existingDepl.Spec.Template.Spec.Containers = deployment.Spec.Template.Spec.Containers
			existingDepl.Spec.Template.Spec.ServiceAccountName = deployment.Spec.Template.Spec.ServiceAccountName
			existingDepl.Spec.Template.Spec.TerminationGracePeriodSeconds = deployment.Spec.Template.Spec.TerminationGracePeriodSeconds

			// Update with the latest deployment spec. Kubernetes will trigger
			// rolling update if spec is different from the one in the cluster.
			existingDepl, err = deploy.kubernetesClient.AppsV1().Deployments(deployNamespace).Update(context.TODO(), existingDepl, metav1.UpdateOptions{})
			if err != nil {
				deploy.logger.Warn("error adopting deploy", zap.Error(err),
					zap.String("deploy", deployName), zap.String("ns", deployNamespace))
				return nil, err
			}
			// In this case, we just return without waiting for it for fast bootstraping.
			return existingDepl, nil
		}

		if *existingDepl.Spec.Replicas < minScale {
			err = deploy.scaleDeployment(existingDepl.Namespace, existingDepl.Name, minScale)
			if err != nil {
				deploy.logger.Error("error scaling up function deployment", zap.Error(err), zap.String("function", fn.ObjectMeta.Name))
				return nil, err
			}
		}
		if existingDepl.Status.AvailableReplicas < minScale {
			existingDepl, err = deploy.waitForDeploy(existingDepl, minScale, specializationTimeout)
		}

		return existingDepl, err
	} else if k8s_err.IsNotFound(err) {
		err := deploy.setupRBACObjs(deployNamespace, fn)
		if err != nil {
			return nil, err
		}

		depl, err := deploy.kubernetesClient.AppsV1().Deployments(deployNamespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			if k8s_err.IsAlreadyExists(err) {
				depl, err = deploy.kubernetesClient.AppsV1().Deployments(deployNamespace).Get(context.TODO(), deployName, metav1.GetOptions{})
			}
			if err != nil {
				deploy.logger.Error("error while creating function deployment",
					zap.Error(err),
					zap.String("function", fn.ObjectMeta.Name),
					zap.String("deployment_name", deployName),
					zap.String("deployment_namespace", deployNamespace))
				return nil, err
			}
		}
		if minScale > 0 {
			depl, err = deploy.waitForDeploy(depl, minScale, specializationTimeout)
		}
		return depl, err
	}
	return nil, err
}

func (deploy *NewDeploy) setupRBACObjs(deployNamespace string, fn *fv1.Function) error {
	// create fetcher SA in this ns, if not already created
	err := deploy.fetcherConfig.SetupServiceAccount(deploy.kubernetesClient, deployNamespace, fn.ObjectMeta)
	if err != nil {
		deploy.logger.Error("error creating fission fetcher service account for function",
			zap.Error(err),
			zap.String("service_account_name", fv1.FissionFetcherSA),
			zap.String("service_account_namespace", deployNamespace),
			zap.String("function_name", fn.ObjectMeta.Name),
			zap.String("function_namespace", fn.ObjectMeta.Namespace))
		return err
	}

	// create a cluster role binding for the fetcher SA, if not already created, granting access to do a get on packages in any ns
	err = utils.SetupRoleBinding(deploy.logger, deploy.kubernetesClient, fv1.PackageGetterRB, fn.Spec.Package.PackageRef.Namespace, fv1.PackageGetterCR, fv1.ClusterRole, fv1.FissionFetcherSA, deployNamespace)
	if err != nil {
		deploy.logger.Error("error creating role binding for function",
			zap.Error(err),
			zap.String("role_binding", fv1.PackageGetterRB),
			zap.String("function_name", fn.ObjectMeta.Name),
			zap.String("function_namespace", fn.ObjectMeta.Namespace))
		return err
	}

	// create rolebinding in function namespace for fetcherSA.envNamespace to be able to get secrets and configmaps
	err = utils.SetupRoleBinding(deploy.logger, deploy.kubernetesClient, fv1.SecretConfigMapGetterRB, fn.ObjectMeta.Namespace, fv1.SecretConfigMapGetterCR, fv1.ClusterRole, fv1.FissionFetcherSA, deployNamespace)
	if err != nil {
		deploy.logger.Error("error creating role binding for function",
			zap.Error(err),
			zap.String("role_binding", fv1.SecretConfigMapGetterRB),
			zap.String("function_name", fn.ObjectMeta.Name),
			zap.String("function_namespace", fn.ObjectMeta.Namespace))
		return err
	}

	deploy.logger.Info("set up all RBAC objects for function",
		zap.String("function_name", fn.ObjectMeta.Name),
		zap.String("function_namespace", fn.ObjectMeta.Namespace))
	return nil
}

func (deploy *NewDeploy) updateDeployment(deployment *appsv1.Deployment, ns string) error {
	_, err := deploy.kubernetesClient.AppsV1().Deployments(ns).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	return err
}

func (deploy *NewDeploy) deleteDeployment(ns string, name string) error {
	// DeletePropagationBackground deletes the object immediately and dependent are deleted later
	// DeletePropagationForeground not advisable; it marks for deleteion and API can still serve those objects
	deletePropagation := metav1.DeletePropagationBackground
	return deploy.kubernetesClient.AppsV1().Deployments(ns).Delete(context.TODO(), name, metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	})
}

func (deploy *NewDeploy) getDeploymentSpec(fn *fv1.Function, env *fv1.Environment, targetReplicas *int32,
	deployName string, deployNamespace string, deployLabels map[string]string, deployAnnotations map[string]string) (*appsv1.Deployment, error) {

	replicas := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)
	if targetReplicas != nil {
		replicas = *targetReplicas
	}

	gracePeriodSeconds := int64(6 * 60)
	if env.Spec.TerminationGracePeriod > 0 {
		gracePeriodSeconds = env.Spec.TerminationGracePeriod
	}

	podAnnotations := env.ObjectMeta.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}

	// Here, we don't append deployAnnotations to podAnnotations
	// since newdeploy doesn't manager pod lifecycle directly.

	if deploy.useIstio && env.Spec.AllowAccessToExternalNetwork {
		podAnnotations["sidecar.istio.io/inject"] = "false"
	}

	podLabels := env.ObjectMeta.Labels
	if podLabels == nil {
		podLabels = make(map[string]string)
	}

	for k, v := range deployLabels {
		podLabels[k] = v
	}

	resources := deploy.getResources(env, fn)

	var maxUnavailable, maxSurge intstr.IntOrString

	if env.Spec.RollingUpdate == nil {
		// Defaults are same as upstream defaults so they match the OpenAPI description
		maxUnavailable = intstr.FromString("25%")
		maxSurge = intstr.FromString("25%")
	} else {
		maxSurge = intstr.FromString(env.Spec.RollingUpdate.MaxSurge.String())
		maxUnavailable = intstr.FromString(env.Spec.RollingUpdate.MaxUnavailable.String())
	}
	// Newdeploy updates the environment variable "LastUpdateTimestamp" of deployment
	// whenever a configmap/secret gets an update, but it also leaves multiple ReplicaSets for
	// rollback purpose. Since fission always update a deployment instead of performing a
	// rollback, set RevisionHistoryLimit to 0 to disable this feature.
	revisionHistoryLimit := int32(0)

	rvCount, err := referencedResourcesRVSum(deploy.kubernetesClient, fn.ObjectMeta.Namespace, fn.Spec.Secrets, fn.Spec.ConfigMaps)
	if err != nil {
		return nil, err
	}

	container, err := util.MergeContainer(&apiv1.Container{
		Name:                   env.ObjectMeta.Name,
		Image:                  env.Spec.Runtime.Image,
		ImagePullPolicy:        deploy.runtimeImagePullPolicy,
		TerminationMessagePath: "/dev/termination-log",
		Lifecycle: &apiv1.Lifecycle{
			PreStop: &apiv1.Handler{
				Exec: &apiv1.ExecAction{
					Command: []string{
						"/bin/sleep",
						fmt.Sprintf("%v", gracePeriodSeconds),
					},
				},
			},
		},
		Env: []apiv1.EnvVar{
			{
				Name:  fv1.ResourceVersionCount,
				Value: fmt.Sprintf("%v", rvCount),
			},
		},
		// https://istio.io/docs/setup/kubernetes/additional-setup/requirements/
		Ports: []apiv1.ContainerPort{
			{
				Name: "http-env",
				// Now that we have added Port field in spec, should we make this configurable too?
				ContainerPort: int32(8888),
			},
		},
		Resources: resources,
	}, env.Spec.Runtime.Container)
	if err != nil {
		return nil, err
	}

	pod := apiv1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      podLabels,
			Annotations: podAnnotations,
		},
		Spec: apiv1.PodSpec{
			Containers:                    []apiv1.Container{*container},
			ServiceAccountName:            "fission-fetcher",
			TerminationGracePeriodSeconds: &gracePeriodSeconds,
		},
	}

	pod.Spec = *(util.ApplyImagePullSecret(env.Spec.ImagePullSecret, pod.Spec))

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deployName,
			Labels:      deployLabels,
			Annotations: deployAnnotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: deployLabels,
			},
			Template: pod,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	// Order of merging is important here - first fetcher, then containers and lastly pod spec
	err = deploy.fetcherConfig.AddSpecializingFetcherToPodSpec(
		&deployment.Spec.Template.Spec,
		env.ObjectMeta.Name,
		fn,
		env,
	)
	if err != nil {
		return nil, err
	}

	if env.Spec.Runtime.PodSpec != nil {
		newPodSpec, err := util.MergePodSpec(&deployment.Spec.Template.Spec, env.Spec.Runtime.PodSpec)
		if err != nil {
			return nil, err
		}
		deployment.Spec.Template.Spec = *newPodSpec
	}

	return deployment, nil
}

// getResources overrides only the resources which are overridden at function level otherwise
// default to resources specified at environment level
func (deploy *NewDeploy) getResources(env *fv1.Environment, fn *fv1.Function) apiv1.ResourceRequirements {
	resources := env.Spec.Resources
	if resources.Requests == nil {
		resources.Requests = make(map[apiv1.ResourceName]resource.Quantity)
	}
	if resources.Limits == nil {
		resources.Limits = make(map[apiv1.ResourceName]resource.Quantity)
	}
	// Only override the once specified at function, rest default to values from env.
	val, ok := fn.Spec.Resources.Requests[apiv1.ResourceCPU]
	if ok && !val.IsZero() {
		resources.Requests[apiv1.ResourceCPU] = fn.Spec.Resources.Requests[apiv1.ResourceCPU]
	}

	val, ok = fn.Spec.Resources.Requests[apiv1.ResourceMemory]
	if ok && !val.IsZero() {
		resources.Requests[apiv1.ResourceMemory] = fn.Spec.Resources.Requests[apiv1.ResourceMemory]
	}

	val, ok = fn.Spec.Resources.Limits[apiv1.ResourceCPU]
	if ok && !val.IsZero() {
		resources.Limits[apiv1.ResourceCPU] = fn.Spec.Resources.Limits[apiv1.ResourceCPU]
	}

	val, ok = fn.Spec.Resources.Limits[apiv1.ResourceMemory]
	if ok && !val.IsZero() {
		resources.Limits[apiv1.ResourceMemory] = fn.Spec.Resources.Limits[apiv1.ResourceMemory]
	}

	return resources
}

func (deploy *NewDeploy) createOrGetHpa(hpaName string, execStrategy *fv1.ExecutionStrategy,
	depl *appsv1.Deployment, deployLabels map[string]string, deployAnnotations map[string]string) (*asv1.HorizontalPodAutoscaler, error) {

	if depl == nil {
		return nil, errors.New("failed to create HPA, found empty deployment")
	}

	minRepl := int32(execStrategy.MinScale)
	if minRepl == 0 {
		minRepl = 1
	}
	maxRepl := int32(execStrategy.MaxScale)
	if maxRepl == 0 {
		maxRepl = minRepl
	}
	targetCPU := int32(execStrategy.TargetCPUPercent)

	hpa := &asv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hpaName,
			Labels:      deployLabels,
			Annotations: deployAnnotations,
		},
		Spec: asv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: asv1.CrossVersionObjectReference{
				Kind:       DeploymentKind,
				Name:       depl.ObjectMeta.Name,
				APIVersion: DeploymentVersion,
			},
			MinReplicas:                    &minRepl,
			MaxReplicas:                    maxRepl,
			TargetCPUUtilizationPercentage: &targetCPU,
		},
	}

	existingHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Get(context.TODO(), hpaName, metav1.GetOptions{})
	if err == nil {
		// to adopt orphan service
		if existingHpa.Annotations[fv1.EXECUTOR_INSTANCEID_LABEL] != deploy.instanceID {
			existingHpa.Annotations = hpa.Annotations
			existingHpa.Labels = hpa.Labels
			existingHpa.Spec = hpa.Spec
			existingHpa, err = deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Update(context.TODO(), existingHpa, metav1.UpdateOptions{})
			if err != nil {
				deploy.logger.Warn("error adopting HPA", zap.Error(err),
					zap.String("HPA", hpaName), zap.String("ns", depl.ObjectMeta.Namespace))
				return nil, err
			}
		}
		return existingHpa, err
	} else if k8s_err.IsNotFound(err) {
		cHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Create(context.TODO(), hpa, metav1.CreateOptions{})
		if err != nil {
			if k8s_err.IsAlreadyExists(err) {
				cHpa, err = deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Get(context.TODO(), hpaName, metav1.GetOptions{})
			}
			if err != nil {
				return nil, err
			}
		}
		return cHpa, nil
	}
	return nil, err
}

func (deploy *NewDeploy) getHpa(ns, name string) (*asv1.HorizontalPodAutoscaler, error) {
	return deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(ns).Get(context.TODO(), name, metav1.GetOptions{})
}

func (deploy *NewDeploy) updateHpa(hpa *asv1.HorizontalPodAutoscaler) error {
	_, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(hpa.ObjectMeta.Namespace).Update(context.TODO(), hpa, metav1.UpdateOptions{})
	return err
}

func (deploy *NewDeploy) deleteHpa(ns string, name string) error {
	return deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (deploy *NewDeploy) createOrGetSvc(deployLabels map[string]string, deployAnnotations map[string]string, svcName string, svcNamespace string) (*apiv1.Service, error) {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        svcName,
			Labels:      deployLabels,
			Annotations: deployAnnotations,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name: "http-env",
					Port: int32(80),
					// Since Function spec now supports Port , should we make this configurable too?
					TargetPort: intstr.FromInt(8888),
				},
			},
			Selector: deployLabels,
			Type:     apiv1.ServiceTypeClusterIP,
		},
	}

	existingSvc, err := deploy.kubernetesClient.CoreV1().Services(svcNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
	if err == nil {
		// to adopt orphan service
		if existingSvc.Annotations[fv1.EXECUTOR_INSTANCEID_LABEL] != deploy.instanceID {
			existingSvc.Annotations = service.Annotations
			existingSvc.Labels = service.Labels
			existingSvc.Spec.Ports = service.Spec.Ports
			existingSvc.Spec.Selector = service.Spec.Selector
			existingSvc.Spec.Type = service.Spec.Type
			existingSvc, err = deploy.kubernetesClient.CoreV1().Services(svcNamespace).Update(context.TODO(), existingSvc, metav1.UpdateOptions{})
			if err != nil {
				deploy.logger.Warn("error adopting service", zap.Error(err),
					zap.String("service", svcName), zap.String("ns", svcNamespace))
				return nil, err
			}
		}
		return existingSvc, err
	} else if k8s_err.IsNotFound(err) {
		svc, err := deploy.kubernetesClient.CoreV1().Services(svcNamespace).Create(context.TODO(), service, metav1.CreateOptions{})
		if err != nil {
			if k8s_err.IsAlreadyExists(err) {
				svc, err = deploy.kubernetesClient.CoreV1().Services(svcNamespace).Get(context.TODO(), svcName, metav1.GetOptions{})
			}
			if err != nil {
				return nil, err
			}
		}
		return svc, nil
	}
	return nil, err
}

func (deploy *NewDeploy) deleteSvc(ns string, name string) error {
	return deploy.kubernetesClient.CoreV1().Services(ns).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (deploy *NewDeploy) waitForDeploy(depl *appsv1.Deployment, replicas int32, specializationTimeout int) (latestDepl *appsv1.Deployment, err error) {
	oldStatus := depl.Status

	// if no specializationTimeout is set, use default value
	if specializationTimeout < fv1.DefaultSpecializationTimeOut {
		specializationTimeout = fv1.DefaultSpecializationTimeOut
	}

	for i := 0; i < specializationTimeout; i++ {
		latestDepl, err = deploy.kubernetesClient.AppsV1().Deployments(depl.ObjectMeta.Namespace).Get(context.TODO(), depl.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		// TODO check for imagePullerror
		// use AvailableReplicas here is better than ReadyReplicas
		// since the pods may not be able to serve network traffic yet.
		if latestDepl.Status.AvailableReplicas >= replicas {
			return latestDepl, err
		}
		time.Sleep(time.Second)
	}

	deploy.logger.Error("Deployment provision failed within timeout window",
		zap.String("name", latestDepl.ObjectMeta.Name), zap.Any("old_status", oldStatus),
		zap.Any("current_status", latestDepl.Status), zap.Int("timeout", specializationTimeout))

	// this error appears in the executor pod logs
	timeoutError := fmt.Errorf("failed to create deployment within the timeout window of %d seconds", specializationTimeout)
	return nil, timeoutError
}

// cleanupNewdeploy cleans all kubernetes objects related to function
func (deploy *NewDeploy) cleanupNewdeploy(ns string, name string) error {
	result := &multierror.Error{}

	err := deploy.deleteSvc(ns, name)
	if err != nil && !k8s_err.IsNotFound(err) {
		deploy.logger.Error("error deleting service for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		result = multierror.Append(result, err)
	}

	err = deploy.deleteHpa(ns, name)
	if err != nil && !k8s_err.IsNotFound(err) {
		deploy.logger.Error("error deleting HPA for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		result = multierror.Append(result, err)
	}

	err = deploy.deleteDeployment(ns, name)
	if err != nil && !k8s_err.IsNotFound(err) {
		deploy.logger.Error("error deleting deployment for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

// referencedResourcesRVSum returns the sum of resource version of all resources the function references to.
// We used to update timestamp in the deployment environment field in order to trigger a rolling update when
// the function referenced resources get updated. However, use timestamp means we are not able to avoid
// triggering a rolling update when executor tries to adopt orphaned deployment due to timestamp changed which
// is unwanted. In order to let executor adopt deployment without triggering a rolling update, we need an
// identical way to get a value that can reflect resources changed without affecting by the time.
// To achieve this goal, the sum of the resource version of all referenced resources is a good fit for our
// scenario since the sum of the resource version is always the same as long as no resources changed.
func referencedResourcesRVSum(client *kubernetes.Clientset, namespace string, secrets []fv1.SecretReference, cfgmaps []fv1.ConfigMapReference) (int, error) {
	rvCount := 0

	if len(secrets) > 0 {
		list, err := client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return 0, err
		}

		objmap := make(map[string]apiv1.Secret)
		for _, secret := range list.Items {
			objmap[secret.Namespace+"/"+secret.Name] = secret
		}

		for _, ref := range secrets {
			s, ok := objmap[ref.Namespace+"/"+ref.Name]
			if ok {
				rv, _ := strconv.ParseInt(s.ResourceVersion, 10, 32)
				rvCount += int(rv)
			}
		}
	}

	if len(cfgmaps) > 0 {
		list, err := client.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return 0, err
		}

		objmap := make(map[string]apiv1.ConfigMap)
		for _, cfg := range list.Items {
			objmap[cfg.Namespace+"/"+cfg.Name] = cfg
		}

		for _, ref := range cfgmaps {
			s, ok := objmap[ref.Namespace+"/"+ref.Name]
			if ok {
				rv, _ := strconv.ParseInt(s.ResourceVersion, 10, 32)
				rvCount += int(rv)
			}
		}
	}

	return rvCount, nil
}
