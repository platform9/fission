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

	k8s_err "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	asv1 "k8s.io/client-go/pkg/apis/autoscaling/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

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
	deployName string, deployLabels map[string]string) (*v1beta1.Deployment, error) {

	replicas := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)
	if replicas == 0 {
		replicas = 1
	}
	targetFilename := "user"
	userfunc := "userfunc"
	secrets := "secrets"
	config := "config"
	var gracePeriodSeconds int64 = 6 * 60

	existingDepl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deploy.namespace).Get(deployName, metav1.GetOptions{})
	if err == nil && existingDepl.Status.ReadyReplicas >= replicas {
		return existingDepl, err
	}

	if err != nil && k8s_err.IsNotFound(err) {
		fetchReq := &fetcher.FetchRequest{
			FetchType: fetcher.FETCH_DEPLOYMENT,
			Package: metav1.ObjectMeta{
				Namespace: fn.Spec.Package.PackageRef.Namespace,
				Name:      fn.Spec.Package.PackageRef.Name,
			},
			Filename:   targetFilename,
			Secrets:    fn.Spec.Secrets,
			ConfigMaps: fn.Spec.ConfigMaps,
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

		podAnnotation := map[string]string{
			"sidecar.istio.io/inject": strconv.FormatBool(env.Spec.AllowedAccessExternalNetwork),
		}

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
						Annotations: podAnnotation,
					},
					Spec: apiv1.PodSpec{
						Volumes: []apiv1.Volume{
							{
								Name: userfunc,
								VolumeSource: apiv1.VolumeSource{
									EmptyDir: &apiv1.EmptyDirVolumeSource{},
								},
							},
						},
						Containers: []apiv1.Container{
							{
								Name:                   fn.Metadata.Name,
								Image:                  env.Spec.Runtime.Image,
								ImagePullPolicy:        apiv1.PullIfNotPresent,
								TerminationMessagePath: "/dev/termination-log",
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      userfunc,
										MountPath: deploy.sharedMountPath,
									},
									{
										Name:      secrets,
										MountPath: deploy.sharedSecretPath,
									},

									{
										Name:      config,
										MountPath: deploy.sharedCfgMapPath,
									},
								},
								Resources: env.Spec.Resources,
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
							},
							{
								Name:                   "fetcher",
								Image:                  deploy.fetcherImg,
								ImagePullPolicy:        deploy.fetcherImagePullPolicy,
								TerminationMessagePath: "/dev/termination-log",
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      userfunc,
										MountPath: deploy.sharedMountPath,
									},
									{
										Name:      secrets,
										MountPath: deploy.sharedSecretPath,
									},
									{
										Name:      config,
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
								// TBD Use smaller default resources, for now needed to make HPA work
								Resources: fetcherResources,
								ReadinessProbe: &apiv1.Probe{
									Handler: apiv1.Handler{
										Exec: &apiv1.ExecAction{
											Command: []string{"cat", "/tmp/ready"},
										},
									},
									InitialDelaySeconds: 1,
									PeriodSeconds:       1,
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
		depl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deploy.namespace).Create(deployment)
		if err != nil {
			log.Printf("Error while creating deployment: %v", err)
			return nil, err
		}

		for i := 0; i < 120; i++ {
			latestDepl, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deploy.namespace).Get(depl.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			//TODO check for imagePullerror
			if latestDepl.Status.ReadyReplicas == replicas {
				return latestDepl, err
			}
			time.Sleep(time.Second)
		}
		return nil, errors.New("failed to create deployment within timeout window")
	}

	return nil, err

}

func (deploy *NewDeploy) deleteDeployment(ns string, name string) error {
	deletePropagation := metav1.DeletePropagationForeground
	err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(ns).Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	})
	if err != nil {
		return err
	}
	return nil
}

func (deploy *NewDeploy) createOrGetHpa(hpaName string, execStrategy *fission.ExecutionStrategy, depl *v1beta1.Deployment) (*asv1.HorizontalPodAutoscaler, error) {

	minRepl := int32(execStrategy.MinScale)
	if minRepl == 0 {
		minRepl = 1
	}
	maxRepl := int32(execStrategy.MaxScale)
	targetCPU := int32(execStrategy.TargetCPUPercent)

	existingHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(deploy.namespace).Get(hpaName, metav1.GetOptions{})
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
				Namespace: deploy.namespace,
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

		cHpa, err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(deploy.namespace).Create(&hpa)
		if err != nil {
			return nil, err
		}
		return cHpa, nil
	}

	return nil, err

}

func (deploy NewDeploy) deleteHpa(ns string, name string) error {
	err := deploy.kubernetesClient.AutoscalingV1().HorizontalPodAutoscalers(ns).Delete(name, &metav1.DeleteOptions{})
	return err
}

func (deploy *NewDeploy) createOrGetSvc(deployLabels map[string]string, svcName string) (*apiv1.Service, error) {

	existingSvc, err := deploy.kubernetesClient.CoreV1().Services(deploy.namespace).Get(svcName, metav1.GetOptions{})
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

		svc, err := deploy.kubernetesClient.CoreV1().Services(deploy.namespace).Create(service)
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
