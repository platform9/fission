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
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/fission/fission/crd"
)

var podNamespace string

func init() {
	podNamespace = os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "fission"
	}
}

func createIngress(trigger *crd.HTTPTrigger, kubeClient *kubernetes.Clientset) {

	if !trigger.Spec.CreateIngress {
		log.Printf("Skipping creation of ingress for trigger: %v", trigger.Metadata.Name)
	}

	ing := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       getDeployLabels(trigger),
			GenerateName: "ingress-" + trigger.Spec.FunctionReference.Name + "-",
			// The Ingress NS MUST be same as Router NS, check long discussion:
			// https://github.com/kubernetes/kubernetes/issues/17088
			// We need to revisit this in future, once Kubernetes supports cross namespace ingress
			Namespace: podNamespace,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				v1beta1.IngressRule{
					Host: trigger.Spec.Host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								v1beta1.HTTPIngressPath{
									Backend: v1beta1.IngressBackend{
										ServiceName: "router",
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 80,
										},
									},
									Path: trigger.Spec.RelativeURL,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Create(ing)
	if err != nil {
		log.Printf("Failed to create ingress: %v", err)
	}
}

func getDeployLabels(trigger *crd.HTTPTrigger) map[string]string {
	return map[string]string{
		"triggerName":  trigger.Metadata.Name,
		"functionName": trigger.Spec.FunctionReference.Name,
	}
}

func deleteIngress(trigger *crd.HTTPTrigger, kubeClient *kubernetes.Clientset) {
	if !trigger.Spec.CreateIngress {
		return
	}

	ingressList, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).List(v1.ListOptions{
		LabelSelector: labels.Set(getDeployLabels(trigger)).AsSelector().String(),
	})
	if err != nil {
		log.Printf("Failed to get ingress for trigger: %v, %v", err, trigger)
	}

	ingress := filterIngress(ingressList, trigger)

	err = kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Delete(ingress.Name, &v1.DeleteOptions{})

	if err != nil {
		log.Printf("Failed to delete ingress %v error: %v", ingress, err)
	}

}

func updateIngress(oldT *crd.HTTPTrigger, newT *crd.HTTPTrigger, kubeClient *kubernetes.Clientset) {

	if oldT.Spec.CreateIngress == false && newT.Spec.CreateIngress == true {
		createIngress(newT, kubeClient)
		return
	}

	if newT.Spec.CreateIngress == false && oldT.Spec.CreateIngress == true {
		deleteIngress(oldT, kubeClient)
		return
	}

	if newT.Spec.Host != oldT.Spec.Host || newT.Spec.RelativeURL != oldT.Spec.RelativeURL {
		ingressList, err := kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).List(v1.ListOptions{
			LabelSelector: labels.Set(getDeployLabels(oldT)).AsSelector().String(),
		})
		if err != nil {
			log.Printf("Failed to get ingress for trigger: %v", err)
		}

		ingress := filterIngress(ingressList, oldT)

		if newT.Spec.Host != oldT.Spec.Host {
			ingress.Spec.Rules[0].Host = newT.Spec.Host
		}

		if newT.Spec.RelativeURL != oldT.Spec.RelativeURL {
			ingress.Spec.Rules[0].HTTP.Paths[0].Path = newT.Spec.RelativeURL
		}

		_, err = kubeClient.ExtensionsV1beta1().Ingresses(podNamespace).Update(ingress)
		if err != nil {
			log.Printf("Failed to update ingress for trigger: %v", err)
		}
	}

}

func filterIngress(iList *v1beta1.IngressList, trigger *crd.HTTPTrigger) *v1beta1.Ingress {
	if len(iList.Items) > 1 {
		log.Printf("Found more than one ingress, using the first one: %v", iList)
		//TODO: May be check if ingress name uses convention - ingress-funcname-* instead of randomly selecting first??
	}
	return &iList.Items[0]
}
