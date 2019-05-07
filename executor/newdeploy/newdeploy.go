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
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	asv1 "k8s.io/api/autoscaling/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8s_err "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	multierror "github.com/hashicorp/go-multierror"
)

const (
	DeploymentKind    = "Deployment"
	DeploymentVersion = "extensions/v1beta1"
)

func (deploy *NewDeploy) createOrGetDeployment(fn *crd.Function, env *crd.Environment,
	deployName string, deployLabels map[string]string, deployNamespace string, firstcreate bool) (*v1beta1.Deployment, error) {

	minScale := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)

	// If it's not the first time creation and minscale is 0 means that all pods for function were recycled,
	// in such cases we need set minscale to 1 for router to serve requests.
	if !firstcreate && minScale <= 0 {
		minScale = 1
	}

	waitForDeploy := minScale > 0

	existingDepl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deployNamespace).Get(deployName, metav1.GetOptions{})
	if err == nil {
		if waitForDeploy {
			err = deploy.scaleDeployment(existingDepl.Namespace, existingDepl.Name, minScale)
			if err != nil {
				deploy.logger.Error("error scaling up function deployment", zap.Error(err), zap.String("function", fn.Metadata.Name))
				return nil, err
			}

			if existingDepl.Status.AvailableReplicas < minScale {
				existingDepl, err = deploy.waitForDeploy(existingDepl, minScale)
			}
		}
		return existingDepl, err
	}

	if err != nil && k8s_err.IsNotFound(err) {
		err := deploy.setupRBACObjs(deployNamespace, fn)
		if err != nil {
			return nil, err
		}

		deployment, err := deploy.getDeploymentSpec(fn, env, deployName, deployLabels)
		if err != nil {
			return nil, err
		}

		depl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deployNamespace).Create(deployment)
		if err != nil {
			deploy.logger.Error("error while creating function deployment",
				zap.Error(err),
				zap.String("function", fn.Metadata.Name),
				zap.String("deployment_name", deployName),
				zap.String("deployment_namespace", deployNamespace))
			return nil, err
		}

		if waitForDeploy {
			depl, err = deploy.waitForDeploy(depl, minScale)
		}

		return depl, err
	}

	return nil, err

}

func (deploy *NewDeploy) setupRBACObjs(deployNamespace string, fn *crd.Function) error {
	// create fetcher SA in this ns, if not already created
	err := deploy.fetcherConfig.SetupServiceAccount(deploy.kubernetesClient, deployNamespace, fn.Metadata)
	if err != nil {
		deploy.logger.Error("error creating fission fetcher service account for function",
			zap.Error(err),
			zap.String("service_account_name", fission.FissionFetcherSA),
			zap.String("service_account_namespace", deployNamespace),
			zap.String("function_name", fn.Metadata.Name),
			zap.String("function_namespace", fn.Metadata.Namespace))
		return err
	}

	// create a cluster role binding for the fetcher SA, if not already created, granting access to do a get on packages in any ns
	err = fission.SetupRoleBinding(deploy.logger, deploy.kubernetesClient, fission.PackageGetterRB, fn.Spec.Package.PackageRef.Namespace, fission.PackageGetterCR, fission.ClusterRole, fission.FissionFetcherSA, deployNamespace)
	if err != nil {
		deploy.logger.Error("error creating role binding for function",
			zap.Error(err),
			zap.String("role_binding", fission.PackageGetterRB),
			zap.String("function_name", fn.Metadata.Name),
			zap.String("function_namespace", fn.Metadata.Namespace))
		return err
	}

	// create rolebinding in function namespace for fetcherSA.envNamespace to be able to get secrets and configmaps
	err = fission.SetupRoleBinding(deploy.logger, deploy.kubernetesClient, fission.SecretConfigMapGetterRB, fn.Metadata.Namespace, fission.SecretConfigMapGetterCR, fission.ClusterRole, fission.FissionFetcherSA, deployNamespace)
	if err != nil {
		deploy.logger.Error("error creating role binding for function",
			zap.Error(err),
			zap.String("role_binding", fission.SecretConfigMapGetterRB),
			zap.String("function_name", fn.Metadata.Name),
			zap.String("function_namespace", fn.Metadata.Namespace))
		return err
	}

	deploy.logger.Info("set up all RBAC objects for function",
		zap.String("function_name", fn.Metadata.Name),
		zap.String("function_namespace", fn.Metadata.Namespace))
	return nil
}

func (deploy *NewDeploy) getDeployment(ns, name string) (*v1beta1.Deployment, error) {
	return deploy.kubernetesClient.ExtensionsV1beta1().Deployments(ns).Get(name, metav1.GetOptions{})
}

func (deploy *NewDeploy) updateDeployment(deployment *v1beta1.Deployment, ns string) error {
	_, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(ns).Update(deployment)
	return err
}

func (deploy *NewDeploy) deleteDeployment(ns string, name string) error {
	// DeletePropagationBackground deletes the object immediately and dependent are deleted later
	// DeletePropagationForeground not advisable; it markes for deleteion and API can still serve those objects
	deletePropagation := metav1.DeletePropagationBackground
	err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(ns).Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	})
	if err != nil {
		return err
	}
	return nil
}

func (deploy *NewDeploy) getDeploymentSpec(fn *crd.Function, env *crd.Environment,
	deployName string, deployLabels map[string]string) (*v1beta1.Deployment, error) {

	replicas := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)

	gracePeriodSeconds := int64(6 * 60)
	if env.Spec.TerminationGracePeriod > 0 {
		gracePeriodSeconds = env.Spec.TerminationGracePeriod
	}

	podAnnotations := env.Metadata.Annotations
	if podAnnotations == nil {
		podAnnotations = make(map[string]string)
	}
	if deploy.useIstio && env.Spec.AllowAccessToExternalNetwork {
		podAnnotations["sidecar.istio.io/inject"] = "false"
	}
	resources := deploy.getResources(env, fn)

	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deployName,
			Labels: deployLabels,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: deployLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      deployLabels,
					Annotations: podAnnotations,
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
						apiv1.Container{
							Name:                   fn.Metadata.Name,
							Image:                  env.Spec.Runtime.Image,
							ImagePullPolicy:        deploy.runtimeImagePullPolicy,
							TerminationMessagePath: "/dev/termination-log",
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
							Resources: resources,
						},
					},
					ServiceAccountName:            "fission-fetcher",
					TerminationGracePeriodSeconds: &gracePeriodSeconds,
				},
			},
		},
	}

	deploy.fetcherConfig.AddSpecializingFetcherToPodSpec(
		&deployment.Spec.Template.Spec,
		fn.Metadata.Name,
		fn,
		env,
	)

	return deployment, nil
}

// getResources overrides only the resources which are overridden at function level otherwise
// default to resources specified at environment level
func (deploy *NewDeploy) getResources(env *crd.Environment, fn *crd.Function) apiv1.ResourceRequirements {
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

func (deploy *NewDeploy) createOrGetHpa(hpaName string, execStrategy *fission.ExecutionStrategy, depl *v1beta1.Deployment) (*asv1.HorizontalPodAutoscaler, error) {

	minRepl := int32(execStrategy.MinScale)
	if minRepl == 0 {
		minRepl = 1
	}
	maxRepl := int32(execStrategy.MaxScale)
	if maxRepl == 0 {
		maxRepl = minRepl
	}
	targetCPU := int32(execStrategy.TargetCPUPercent)

	existingHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Get(hpaName, metav1.GetOptions{})
	if err == nil {
		return existingHpa, err
	}

	if depl == nil {
		return nil, errors.New("failed to create HPA, found empty deployment")
	}

	if err != nil && k8s_err.IsNotFound(err) {
		hpa := asv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:   hpaName,
				Labels: depl.Labels,
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

		cHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(depl.ObjectMeta.Namespace).Create(&hpa)
		if err != nil {
			return nil, err
		}
		return cHpa, nil
	}

	return nil, err

}

func (deploy *NewDeploy) getHpa(ns, name string) (*asv1.HorizontalPodAutoscaler, error) {
	return deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(ns).Get(name, metav1.GetOptions{})
}

func (deploy *NewDeploy) updateHpa(hpa *asv1.HorizontalPodAutoscaler) error {
	_, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(hpa.ObjectMeta.Namespace).Update(hpa)
	return err
}

func (deploy *NewDeploy) deleteHpa(ns string, name string) error {
	err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(ns).Delete(name, &metav1.DeleteOptions{})
	return err
}

func (deploy *NewDeploy) createOrGetSvc(deployLabels map[string]string, svcName string, svcNamespace string) (*apiv1.Service, error) {

	existingSvc, err := deploy.kubernetesClient.CoreV1().Services(svcNamespace).Get(svcName, metav1.GetOptions{})
	if err == nil {
		return existingSvc, err
	}

	if err != nil && k8s_err.IsNotFound(err) {
		service := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   svcName,
				Labels: deployLabels,
			},
			Spec: apiv1.ServiceSpec{
				Ports: []apiv1.ServicePort{
					{
						Name:       "runtime-env-port",
						Port:       int32(80),
						TargetPort: intstr.FromInt(8888),
					},
					{
						Name:       "fetcher-port",
						Port:       int32(8000),
						TargetPort: intstr.FromInt(8000),
					},
				},
				Selector: deployLabels,
				Type:     apiv1.ServiceTypeClusterIP,
			},
		}

		svc, err := deploy.kubernetesClient.CoreV1().Services(svcNamespace).Create(service)
		if err != nil {
			return nil, err
		}

		return svc, nil
	}

	return nil, err
}

func (deploy *NewDeploy) deleteSvc(ns string, name string) error {
	err := deploy.kubernetesClient.CoreV1().Services(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (deploy *NewDeploy) waitForDeploy(depl *v1beta1.Deployment, replicas int32) (*v1beta1.Deployment, error) {
	for i := 0; i < 120; i++ {
		latestDepl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(depl.ObjectMeta.Namespace).Get(depl.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		//TODO check for imagePullerror
		// use AvailableReplicas here is better than ReadyReplicas
		// since the pods may not be able to serve network traffic yet.
		if latestDepl.Status.AvailableReplicas >= replicas {
			return latestDepl, err
		}
		time.Sleep(time.Second)
	}
	return nil, errors.New("failed to create deployment within timeout window")
}

// cleanupNewdeploy cleans all kubernetes objects related to function
func (deploy *NewDeploy) cleanupNewdeploy(ns string, name string) error {
	var multierr *multierror.Error

	err := deploy.deleteSvc(ns, name)
	if err != nil {
		deploy.logger.Error("error deleting service for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		multierror.Append(multierr, err)
	}

	err = deploy.deleteHpa(ns, name)
	if err != nil {
		deploy.logger.Error("error deleting service for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		multierror.Append(multierr, err)
	}

	err = deploy.deleteDeployment(ns, name)
	if err != nil {
		deploy.logger.Error("error deleting deployment for newdeploy function",
			zap.Error(err),
			zap.String("function_name", name),
			zap.String("function_namespace", ns))
		multierror.Append(multierr, err)
	}
	return multierr.ErrorOrNil()
}
