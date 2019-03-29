/*
Copyright 2018 The Fission Authors.

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
	"time"

	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func getIstioServiceLabels(fnName string) map[string]string {
	return map[string]string{
		"functionName": fnName,
	}
}

func (gpm *GenericPoolManager) makeFuncController(fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, fissionfnNamespace string, istioEnabled bool) (k8sCache.Store, k8sCache.Controller) {

	resyncPeriod := 30 * time.Second
	lw := k8sCache.NewListWatchFromClient(fissionClient.GetCrdClient(), "functions", metav1.NamespaceAll, fields.Everything())

	funcStore, controller := k8sCache.NewInformer(lw, &crd.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fn := obj.(*crd.Function)

				// Since istio only allows accessing pod through k8s service,
				// for the functions with executor type "poolmgr" we need to
				// create a service for sending requests to pod in pool.
				// Functions with executor type "Newdeploy" is specialized at
				// pod starts. In this case, just ignore such functions.
				fnExecutorType := fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType

				// In some cases, user may not enter the executorType explicitly, for example in his spec.yaml.
				// we assume it to be of type poolmgr
				if fnExecutorType != "" && fnExecutorType != fission.ExecutorTypePoolmgr {
					return
				}

				// create or update role-binding
				envNs := fissionfnNamespace
				if fn.Spec.Environment.Namespace != metav1.NamespaceDefault {
					envNs = fn.Spec.Environment.Namespace
				}

				// TODO : Just bring to your attention during review :
				// setup rolebinding is tried, if it fails, we dont return. we just log an error and move on, because :
				// 1. not all functions have secrets and/or configmaps, so things will work without this rolebinding in that case.
				// 2. on the contrary, when the route is tried, the env fetcher logs will show a 403 forbidden message and same will be relayed to executor.
				err := fission.SetupRoleBinding(gpm.logger, kubernetesClient, fission.SecretConfigMapGetterRB, fn.Metadata.Namespace, fission.SecretConfigMapGetterCR, fission.ClusterRole, fission.FissionFetcherSA, envNs)
				if err != nil {
					gpm.logger.Error("error creating rolebinding", zap.Error(err), zap.String("role_binding", fission.SecretConfigMapGetterRB))
				} else {
					gpm.logger.Info("successfully set up rolebinding for fetcher service account for function",
						zap.String("service_account", fission.FissionFetcherSA),
						zap.String("service_account_namepsace", envNs),
						zap.String("function_name", fn.Metadata.Name),
						zap.String("function_namespace", fn.Metadata.Namespace))
				}

				if istioEnabled {
					// create a same name service for function
					// since istio only allows the traffic to service
					sel := map[string]string{
						"functionName": fn.Metadata.Name,
						"functionUid":  string(fn.Metadata.UID),
					}

					svcName := fission.GetFunctionIstioServiceName(fn.Metadata.Name, fn.Metadata.Namespace)

					// service for accepting user traffic
					svc := apiv1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: envNs,
							Name:      svcName,
							Labels:    getIstioServiceLabels(fn.Metadata.Name),
						},
						Spec: apiv1.ServiceSpec{
							Type: apiv1.ServiceTypeClusterIP,
							Ports: []apiv1.ServicePort{
								// Service port name should begin with a recognized prefix, or the traffic will be
								// treated as TCP traffic. (https://istio.io/docs/setup/kubernetes/sidecar-injection.html)
								// Originally the ports' name are similar to "http-fetch" and "http-specialize".
								// But for istio 0.5.1, istio-proxy return unexpected 431 error with such naming.
								// https://github.com/istio/istio/issues/928
								// Workaround: remove prefix
								// TODO: prepend prefix once the bug fixed
								{
									Name:       "fetch",
									Protocol:   apiv1.ProtocolTCP,
									Port:       8000,
									TargetPort: intstr.FromInt(8000),
								},
								{
									Name:       "specialize",
									Protocol:   apiv1.ProtocolTCP,
									Port:       8888,
									TargetPort: intstr.FromInt(8888),
								},
							},
							Selector: sel,
						},
					}

					// create function istio service if it does not exist
					_, err = kubernetesClient.CoreV1().Services(envNs).Create(&svc)
					if err != nil && !kerrors.IsAlreadyExists(err) {
						gpm.logger.Error("error creating istio service for function",
							zap.Error(err),
							zap.String("service_name", svcName),
							zap.String("function_name", fn.Metadata.Name),
							zap.Any("selectors", sel))
					}
				}
			},

			DeleteFunc: func(obj interface{}) {
				fn := obj.(*crd.Function)

				fnExecutorType := fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType
				if fnExecutorType != "" && fnExecutorType != fission.ExecutorTypePoolmgr {
					return
				}

				envNs := fissionfnNamespace
				if fn.Spec.Environment.Namespace != metav1.NamespaceDefault {
					envNs = fn.Spec.Environment.Namespace
				}

				if istioEnabled {
					svcName := fission.GetFunctionIstioServiceName(fn.Metadata.Name, fn.Metadata.Namespace)
					// delete function istio service
					err := kubernetesClient.CoreV1().Services(envNs).Delete(svcName, nil)
					if err != nil && !kerrors.IsNotFound(err) {
						gpm.logger.Error("error deleting istio service for function",
							zap.Error(err),
							zap.String("service_name", svcName),
							zap.String("function_name", fn.Metadata.Name))

					}
				}
			},

			UpdateFunc: func(oldObj, newObj interface{}) {
				oldFunc := oldObj.(*crd.Function)
				newFunc := newObj.(*crd.Function)

				if oldFunc.Metadata.ResourceVersion == newFunc.Metadata.ResourceVersion {
					return
				}

				envChanged := (oldFunc.Spec.Environment.Namespace != newFunc.Spec.Environment.Namespace)

				executorTypeChangedToPM := (oldFunc.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypePoolmgr &&
					newFunc.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypePoolmgr)

				// if a func's env reference gets updated and the newly referenced env is in a different ns,
				// we need to create a rolebinding in func's ns so that the fetcher-sa in env ns has access
				// to fetch secrets and config maps from the func's ns.
				// similarly if executorType changed to Pool Manager, we now need a rolebinding in the func ns for fetcher sa
				// present in env ns because for newdeploy, the fetcher sa is in function namespace
				if envChanged || executorTypeChangedToPM {
					envNs := fissionfnNamespace
					if newFunc.Spec.Environment.Namespace != metav1.NamespaceDefault {
						envNs = newFunc.Spec.Environment.Namespace
					}

					err := fission.SetupRoleBinding(gpm.logger, kubernetesClient, fission.SecretConfigMapGetterRB,
						newFunc.Metadata.Namespace, fission.SecretConfigMapGetterCR, fission.ClusterRole,
						fission.FissionFetcherSA, envNs)

					if err != nil {
						gpm.logger.Error("error creating rolebinding", zap.Error(err), zap.String("role_binding", fission.SecretConfigMapGetterRB))
					} else {
						gpm.logger.Info("successfully set up rolebinding for fetcher service account for function",
							zap.String("service_account", fission.FissionFetcherSA),
							zap.String("service_account_namepsace", envNs),
							zap.String("function_name", newFunc.Metadata.Name),
							zap.String("function_namespace", newFunc.Metadata.Namespace))
					}
				}
			},
		})

	return funcStore, controller
}
