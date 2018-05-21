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

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func getIstioServiceLabels(fnName string) map[string]string {
	return map[string]string{
		"functionName": fnName,
	}
}

func makeFuncIstioServiceRegister(crdClient *rest.RESTClient,
	kubernetesClient *kubernetes.Clientset, fnNamespace string) k8sCache.Controller {

	resyncPeriod := 30 * time.Second
	lw := k8sCache.NewListWatchFromClient(crdClient, "functions", metav1.NamespaceDefault, fields.Everything())
	_, controller := k8sCache.NewInformer(lw, &crd.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				fn := obj.(*crd.Function)

				// Since istio only allows accessing pod through k8s service,
				// for the functions with executor type "poolmgr" we need to
				// create a service for sending requests to pod in pool.
				// Functions with executor type "Newdeploy" is specialized at
				// pod starts. In this case, just ignore such functions.
				fnExecutorType := fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType
				if fnExecutorType != fission.ExecutorTypePoolmgr {
					return
				}

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
						Namespace: fnNamespace,
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
				_, err := kubernetesClient.CoreV1().Services(fnNamespace).Create(&svc)
				if err != nil && !kerrors.IsAlreadyExists(err) {
					log.Printf("Error creating function istio service: %v", err)
				}
			},
			DeleteFunc: func(obj interface{}) {
				fn := obj.(*crd.Function)
				svcName := fission.GetFunctionIstioServiceName(fn.Metadata.Name, fn.Metadata.Namespace)
				// delete function istio service
				err := kubernetesClient.CoreV1().Services(fnNamespace).Delete(svcName, nil)
				if err != nil && !kerrors.IsNotFound(err) {
					log.Printf("Error deleting function istio service: %v", err)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {},
		})

	return controller
}
