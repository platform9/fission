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

package cms

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/executortype"
)

type (
	// ConfigSecretController represents a controller for configmaps and secrets
	ConfigSecretController struct {
		logger *zap.Logger

		configmapController cache.Controller
		secretController    cache.Controller

		fissionClient *crd.FissionClient
	}
)

//MakeConfigSecretController makes a controller for configmaps and secrets which changes related functions
func MakeConfigSecretController(logger *zap.Logger, fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, types map[fv1.ExecutorType]executortype.ExecutorType) *ConfigSecretController {
	logger.Debug("Creating ConfigMap & Secret Controller")
	_, cmcontroller := initConfigmapController(logger, fissionClient, kubernetesClient, types)
	_, scontroller := initSecretController(logger, fissionClient, kubernetesClient, types)
	cmsController := &ConfigSecretController{
		logger:              logger,
		configmapController: cmcontroller,
		secretController:    scontroller,
		fissionClient:       fissionClient,
	}
	return cmsController
}

//Run runs the controllers for configmaps and secrets
func (csController *ConfigSecretController) Run(ctx context.Context) {
	go csController.configmapController.Run(ctx.Done())
	go csController.secretController.Run(ctx.Done())
}

func initConfigmapController(logger *zap.Logger, fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, types map[fv1.ExecutorType]executortype.ExecutorType) (cache.Store, cache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := cache.NewListWatchFromClient(kubernetesClient.CoreV1().RESTClient(), "configmaps", metav1.NamespaceAll, fields.Everything())
	store, controller := cache.NewInformer(listWatch, &apiv1.ConfigMap{}, resyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {},
		DeleteFunc: func(obj interface{}) {},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldCm := oldObj.(*apiv1.ConfigMap)
			newCm := newObj.(*apiv1.ConfigMap)
			if oldCm.ObjectMeta.ResourceVersion != newCm.ObjectMeta.ResourceVersion {
				if newCm.ObjectMeta.Namespace != "kube-system" {
					logger.Debug("Configmap changed",
						zap.String("configmap_name", newCm.ObjectMeta.Name),
						zap.String("configmap_namespace", newCm.ObjectMeta.Namespace))
				}
				funcs, err := getConfigmapRelatedFuncs(logger, &newCm.ObjectMeta, fissionClient)
				if err != nil {
					logger.Error("Failed to get functions related to configmap", zap.String("configmap_name", newCm.ObjectMeta.Name), zap.String("configmap_namespace", newCm.ObjectMeta.Namespace))
				}
				refreshPods(logger, funcs, types)
			}
		},
	})
	return store, controller
}

func getConfigmapRelatedFuncs(logger *zap.Logger, m *metav1.ObjectMeta, fissionClient *crd.FissionClient) ([]fv1.Function, error) {
	funcList, err := fissionClient.CoreV1().Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// In future a cache that populates at start and is updated on changes might be better solution
	relatedFunctions := make([]fv1.Function, 0)
	for _, f := range funcList.Items {
		for _, cm := range f.Spec.ConfigMaps {
			if (cm.Name == m.Name) && (cm.Namespace == m.Namespace) {
				relatedFunctions = append(relatedFunctions, f)
				break
			}
		}
	}
	return relatedFunctions, nil
}

func initSecretController(logger *zap.Logger, fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset, types map[fv1.ExecutorType]executortype.ExecutorType) (cache.Store, cache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := cache.NewListWatchFromClient(kubernetesClient.CoreV1().RESTClient(), "secrets", metav1.NamespaceAll, fields.Everything())
	store, controller := cache.NewInformer(listWatch, &apiv1.Secret{}, resyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {},
		DeleteFunc: func(obj interface{}) {},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldS := oldObj.(*apiv1.Secret)
			newS := newObj.(*apiv1.Secret)
			if oldS.ObjectMeta.ResourceVersion != newS.ObjectMeta.ResourceVersion {
				if newS.ObjectMeta.Namespace != "kube-system" {
					logger.Debug("Secret changed",
						zap.String("configmap_name", newS.ObjectMeta.Name),
						zap.String("configmap_namespace", newS.ObjectMeta.Namespace))

				}
				funcs, err := getSecretRelatedFuncs(logger, &newS.ObjectMeta, fissionClient)
				if err != nil {
					logger.Error("Failed to get functions related to secret", zap.String("secret_name", newS.ObjectMeta.Name), zap.String("secret_namespace", newS.ObjectMeta.Namespace))
				}
				refreshPods(logger, funcs, types)
			}
		},
	})
	return store, controller

}

func getSecretRelatedFuncs(logger *zap.Logger, m *metav1.ObjectMeta, fissionClient *crd.FissionClient) ([]fv1.Function, error) {
	funcList, err := fissionClient.CoreV1().Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// In future a cache that populates at start and is updated on changes might be better solution
	relatedFunctions := make([]fv1.Function, 0)
	for _, f := range funcList.Items {
		for _, secret := range f.Spec.Secrets {
			if (secret.Name == m.Name) && (secret.Namespace == m.Namespace) {
				relatedFunctions = append(relatedFunctions, f)
				break
			}
		}
	}
	return relatedFunctions, nil
}

func refreshPods(logger *zap.Logger, funcs []fv1.Function, types map[fv1.ExecutorType]executortype.ExecutorType) {
	for _, f := range funcs {
		var err error

		et, exists := types[f.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType]
		if exists {
			err = et.RefreshFuncPods(logger, f)
		} else {
			err = errors.Errorf("Unknown executor type '%v'", f.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType)
		}

		if err != nil {
			logger.Error("Failed to recycle pods for function after configmap/secret changed",
				zap.Error(err),
				zap.Any("function", f))
		}
	}
}
