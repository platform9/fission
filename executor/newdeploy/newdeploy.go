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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"time"

	asv1 "k8s.io/api/autoscaling/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8s_err "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/environments/fetcher"
	"github.com/fission/fission/executor/util"
)

const (
	DeploymentKind    = "Deployment"
	DeploymentVersion = "extensions/v1beta1"
)

const (
	envVersion = "ENV_VERSION"
)

func (deploy *NewDeploy) createOrGetDeployment(fn *crd.Function, env *crd.Environment,
	deployName string, deployLabels map[string]string, deployNamespace string) (*v1beta1.Deployment, error) {

	replicas := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)
	if replicas == 0 {
		replicas = 1
	}

	existingDepl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deployNamespace).Get(deployName, metav1.GetOptions{})
	if err == nil {
		// If minScale is 0 then scale to 1, otherwise scale up to minScale
		if *existingDepl.Spec.Replicas == 0 {
			err = scaleDeployment(deploy.kubernetesClient,
				existingDepl.Namespace, existingDepl.Name, replicas)
			if err != nil {
				log.Printf("Error scaling up deployment for function %v: %v", fn.Metadata.Name, err)
				return nil, err
			}
		}
		if existingDepl.Status.ReadyReplicas < replicas {
			existingDepl, err = deploy.waitForDeploy(existingDepl, replicas)
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
			log.Printf("Error while creating deployment: %v", err)
			return nil, err
		}

		return deploy.waitForDeploy(depl, replicas)
	}

	return nil, err

}

func (deploy *NewDeploy) setupRBACObjs(deployNamespace string, fn *crd.Function) error {
	// create fetcher SA in this ns, if not already created
	_, err := fission.SetupSA(deploy.kubernetesClient, fission.FissionFetcherSA, deployNamespace)
	if err != nil {
		log.Printf("Error : %v creating %s in ns : %s for function: %s.%s", err, fission.FissionFetcherSA, deployNamespace, fn.Metadata.Name, fn.Metadata.Namespace)
		return err
	}

	// create a cluster role binding for the fetcher SA, if not already created, granting access to do a get on packages in any ns
	err = fission.SetupRoleBinding(deploy.kubernetesClient, fission.PackageGetterRB, fn.Spec.Package.PackageRef.Namespace, fission.PackageGetterCR, fission.ClusterRole, fission.FissionFetcherSA, deployNamespace)
	if err != nil {
		log.Printf("Error : %v creating %s RoleBinding for function: %s.%s", err, fission.PackageGetterRB, fn.Metadata.Name, fn.Metadata.Namespace)
		return err
	}

	// create rolebinding in function namespace for fetcherSA.envNamespace to be able to get secrets and configmaps
	err = fission.SetupRoleBinding(deploy.kubernetesClient, fission.SecretConfigMapGetterRB, fn.Metadata.Namespace, fission.SecretConfigMapGetterCR, fission.ClusterRole, fission.FissionFetcherSA, deployNamespace)
	if err != nil {
		log.Printf("Error : %v creating %s RoleBinding for function %s.%s", err, fission.SecretConfigMapGetterRB, fn.Metadata.Name, fn.Metadata.Namespace)
		return err
	}

	log.Printf("Set up all RBAC objects for function : %s.%s", fn.Metadata.Name, fn.Metadata.Namespace)
	return nil
}

func (deploy *NewDeploy) getDeployment(fn *crd.Function) (*v1beta1.Deployment, error) {
	deployName := deploy.getObjName(fn)
	return deploy.kubernetesClient.ExtensionsV1beta1().Deployments(fn.Metadata.Namespace).Get(deployName, metav1.GetOptions{})
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
	if replicas == 0 {
		replicas = 1
	}

	targetFilename := "user"

	gracePeriodSeconds := int64(6 * 60)
	if env.Spec.TerminationGracePeriod > 0 {
		gracePeriodSeconds = env.Spec.TerminationGracePeriod
	}

	fetchReq := &fetcher.FetchRequest{
		FetchType: fetcher.FETCH_DEPLOYMENT,
		Package: metav1.ObjectMeta{
			Namespace: fn.Spec.Package.PackageRef.Namespace,
			Name:      fn.Spec.Package.PackageRef.Name,
		},
		Filename:    targetFilename,
		Secrets:     fn.Spec.Secrets,
		ConfigMaps:  fn.Spec.ConfigMaps,
		KeepArchive: env.Spec.KeepArchive,
	}

	loadReq := fission.FunctionLoadRequest{
		FilePath:         filepath.Join(deploy.sharedMountPath, targetFilename),
		FunctionName:     fn.Spec.Package.FunctionName,
		FunctionMetadata: &fn.Metadata,
	}

	fetchPayload, err := json.Marshal(fetchReq)
	if err != nil {
		return nil, err
	}
	loadPayload, err := json.Marshal(loadReq)
	if err != nil {
		return nil, err
	}

	fetcherResources, err := util.GetFetcherResources()
	if err != nil {
		log.Printf("Error while parsing fetcher resources: %v", err)
		return nil, err
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
			Labels: deployLabels,
			Name:   deployName,
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
						fission.MergeContainerSpecs(&apiv1.Container{
							Name:                   fn.Metadata.Name,
							Image:                  env.Spec.Runtime.Image,
							ImagePullPolicy:        apiv1.PullIfNotPresent,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumeUserfunc,
									MountPath: deploy.sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: deploy.sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: deploy.sharedCfgMapPath,
								},
							},
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
						}, env.Spec.Runtime.Container),
						{
							Name:                   "fetcher",
							Image:                  deploy.fetcherImg,
							ImagePullPolicy:        deploy.fetcherImagePullPolicy,
							TerminationMessagePath: "/dev/termination-log",
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      fission.SharedVolumeUserfunc,
									MountPath: deploy.sharedMountPath,
								},
								{
									Name:      fission.SharedVolumeSecrets,
									MountPath: deploy.sharedSecretPath,
								},
								{
									Name:      fission.SharedVolumeConfigmaps,
									MountPath: deploy.sharedCfgMapPath,
								},
							},
							Command: []string{"/fetcher", "-specialize-on-startup",
								"-fetch-request", string(fetchPayload),
								"-load-request", string(loadPayload),
								"-secret-dir", deploy.sharedSecretPath,
								"-cfgmap-dir", deploy.sharedCfgMapPath,
								deploy.sharedMountPath},
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
							Env: []apiv1.EnvVar{
								{
									Name:  envVersion,
									Value: strconv.Itoa(env.Spec.Version),
								},
							},
							Resources: fetcherResources,
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
					ServiceAccountName:            "fission-fetcher",
					TerminationGracePeriodSeconds: &gracePeriodSeconds,
				},
			},
		},
	}

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
	_, ok := fn.Spec.Resources.Requests[apiv1.ResourceCPU]
	if ok {
		resources.Requests[apiv1.ResourceCPU] = fn.Spec.Resources.Requests[apiv1.ResourceCPU]
	}

	_, ok = fn.Spec.Resources.Requests[apiv1.ResourceMemory]
	if ok {
		resources.Requests[apiv1.ResourceMemory] = fn.Spec.Resources.Requests[apiv1.ResourceMemory]
	}

	_, ok = fn.Spec.Resources.Limits[apiv1.ResourceCPU]
	if ok {
		resources.Limits[apiv1.ResourceCPU] = fn.Spec.Resources.Limits[apiv1.ResourceCPU]
	}

	_, ok = fn.Spec.Resources.Limits[apiv1.ResourceMemory]
	if ok {
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
				Name:      hpaName,
				Namespace: depl.ObjectMeta.Namespace,
				Labels:    depl.Labels,
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

func (deploy *NewDeploy) getHpa(fn *crd.Function) (*asv1.HorizontalPodAutoscaler, error) {
	hpaName := deploy.getObjName(fn)
	return deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(fn.Metadata.Namespace).Get(hpaName, metav1.GetOptions{})
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
		if latestDepl.Status.ReadyReplicas >= replicas {
			return latestDepl, err
		}
		time.Sleep(time.Second)
	}
	return nil, errors.New("failed to create deployment within timeout window")
}
