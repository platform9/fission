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

package router

import (
	"context"
	"os"
	"reflect"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/router/util"
)

var podNamespace string

func init() {
	podNamespace = os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "fission"
	}
}

func createIngress(logger *zap.Logger, trigger *fv1.HTTPTrigger, kubeClient *kubernetes.Clientset) {
	if !trigger.Spec.CreateIngress {
		return
	}
	_, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Create(context.Background(), util.GetIngressSpec(podNamespace, trigger), v1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		logger.Error("failed to create ingress", zap.Error(err))
		return
	}
	logger.Debug("created ingress successfully for trigger", zap.String("trigger", trigger.ObjectMeta.Name))
}

func deleteIngress(logger *zap.Logger, trigger *fv1.HTTPTrigger, kubeClient *kubernetes.Clientset) {
	if !trigger.Spec.CreateIngress {
		return
	}

	ingress, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Get(context.Background(), trigger.ObjectMeta.Name, v1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error("failed to get ingress when deleting trigger", zap.Error(err), zap.String("trigger", trigger.ObjectMeta.Name))
		return
	}

	err = kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Delete(context.Background(), ingress.Name, v1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error("failed to delete ingress for trigger",
			zap.Error(err),
			zap.Any("ingress", ingress),
			zap.String("trigger", trigger.ObjectMeta.Name))
	}
}

func updateIngress(logger *zap.Logger, oldT *fv1.HTTPTrigger, newT *fv1.HTTPTrigger, kubeClient *kubernetes.Clientset) {
	if !oldT.Spec.CreateIngress && !newT.Spec.CreateIngress {
		return
	}

	if !oldT.Spec.CreateIngress && newT.Spec.CreateIngress {
		createIngress(logger, newT, kubeClient)
		return
	}

	if !newT.Spec.CreateIngress && oldT.Spec.CreateIngress {
		deleteIngress(logger, oldT, kubeClient)
		return
	}

	oldIngress, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Get(context.Background(), oldT.ObjectMeta.Name, v1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			createIngress(logger, newT, kubeClient)
		}
		logger.Error("failed to get ingress when updating trigger",
			zap.Error(err),
			zap.String("trigger", oldT.ObjectMeta.Name))
		return
	}
	newIngress := util.GetIngressSpec(podNamespace, newT)

	changes := false

	if !reflect.DeepEqual(oldIngress.Annotations, newIngress.Annotations) {
		logger.Debug("ingress annotation",
			zap.Any("old_trigger", oldIngress.Annotations), zap.Any("new_trigger", newIngress.Annotations))

		if oldIngress.Annotations == nil || newIngress.Annotations == nil {
			oldIngress.Annotations = newIngress.Annotations
		} else {
			for k, v := range newIngress.Annotations {
				oldIngress.Annotations[k] = v
			}
		}
		changes = true
	}

	if !reflect.DeepEqual(oldIngress.Spec, newIngress.Spec) {
		logger.Debug("ingress spec",
			zap.Any("old_trigger", oldIngress.Spec), zap.Any("new_trigger", newIngress.Spec))

		oldIngress.Spec = newIngress.Spec
		changes = true
	}

	if changes {
		_, err = kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Update(context.Background(), oldIngress, v1.UpdateOptions{})
		if err != nil {
			logger.Error("failed to update ingress for trigger", zap.Error(err), zap.String("trigger", oldT.ObjectMeta.Name))
			return
		}

		logger.Debug("updated ingress successfully for trigger",
			zap.String("old_trigger", oldT.ObjectMeta.Name), zap.String("new_trigger", newT.ObjectMeta.Name))
	}
}
