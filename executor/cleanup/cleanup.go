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

package cleanup

import (
	"fmt"
	"log"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

var (
	deletePropagation = meta_v1.DeletePropagationBackground
	delOpt            = meta_v1.DeleteOptions{PropagationPolicy: &deletePropagation}
)

// CleanupObjects cleans up resources created by old executortype instances
func CleanupObjects(kubernetesClient *kubernetes.Clientset,
	namespace string,
	instanceId string) {
	go func() {
		err := cleanup(kubernetesClient, namespace, instanceId)
		if err != nil {
			// TODO retry cleanup; logged and ignored for now
			log.Printf("Failed to cleanup: %v", err)
		}
	}()
}

func cleanup(client *kubernetes.Clientset, namespace string, instanceId string) error {

	err := cleanupServices(client, namespace, instanceId)
	if err != nil {
		return err
	}

	err = cleanupHpa(client, namespace, instanceId)
	if err != nil {
		return err
	}

	// Deployments are used for idle pools and can be cleaned up
	// immediately.  (We should "adopt" these instead of creating
	// a new pool.)
	err = cleanupDeployments(client, namespace, instanceId)
	if err != nil {
		return err
	}

	// Pods might still be running user functions, so we give them
	// a few minutes before terminating them.  This time is the
	// maximum function runtime, plus the time a router might
	// still route to an old instance, i.e. router cache expiry
	// time.
	time.Sleep(6 * time.Minute)

	err = cleanupPods(client, namespace, instanceId)
	if err != nil {
		return err
	}

	return nil
}

// DeleteKubeObject deletes given kubernetes object
func DeleteKubeObject(kubeClient *kubernetes.Clientset, kubeobj *apiv1.ObjectReference) {
	switch strings.ToLower(kubeobj.Kind) {
	case "pod":
		err := kubeClient.CoreV1().Pods(kubeobj.Namespace).Delete(kubeobj.Name, nil)
		logErr(fmt.Sprintf("cleaning up pod %v ", kubeobj.Name), err)

	case "service":
		err := kubeClient.CoreV1().Services(kubeobj.Namespace).Delete(kubeobj.Name, nil)
		logErr(fmt.Sprintf("cleaning up service %v ", kubeobj.Name), err)

	case "deployment":
		err := kubeClient.ExtensionsV1beta1().Deployments(kubeobj.Namespace).Delete(kubeobj.Name, &delOpt)
		logErr(fmt.Sprintf("cleaning up deployment %v ", kubeobj.Name), err)

	case "horizontalpodautoscaler":
		err := kubeClient.AutoscalingV1().HorizontalPodAutoscalers(kubeobj.Namespace).Delete(kubeobj.Name, nil)
		logErr(fmt.Sprintf("cleaning up horizontalpodautoscaler %v ", kubeobj.Name), err)

	default:
		log.Printf("There was an error identifying the object type: %v for obj: %v", kubeobj.Kind, kubeobj)

	}
}

func cleanupDeployments(client *kubernetes.Clientset, namespace string, instanceId string) error {
	deploymentList, err := client.ExtensionsV1beta1().Deployments(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dep := range deploymentList.Items {
		id, ok := dep.ObjectMeta.Labels[fission.EXECUTOR_INSTANCEID_LABEL]
		if ok && id != instanceId {
			log.Printf("Cleaning up deployment %v", dep.ObjectMeta.Name)
			err := client.ExtensionsV1beta1().Deployments(namespace).Delete(dep.ObjectMeta.Name, &delOpt)
			logErr("cleaning up deployment", err)
			// ignore err
		}
		// Backward compatibility with older label name
		pid, pok := dep.ObjectMeta.Labels[fission.POOLMGR_INSTANCEID_LABEL]
		if pok && pid != instanceId {
			log.Printf("Cleaning up deployment %v", dep.ObjectMeta.Name)
			err := client.ExtensionsV1beta1().Deployments(namespace).Delete(dep.ObjectMeta.Name, &delOpt)
			logErr("cleaning up deployment", err)
			// ignore err
		}
	}
	return nil
}

func cleanupPods(client *kubernetes.Clientset, namespace string, instanceId string) error {
	podList, err := client.CoreV1().Pods(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range podList.Items {
		log.Printf("Clean pod: %v", pod.ObjectMeta.Name)
		id, ok := pod.ObjectMeta.Labels[fission.EXECUTOR_INSTANCEID_LABEL]
		if ok && id != instanceId {
			log.Printf("Cleaning up pod %v", pod.ObjectMeta.Name)
			err := client.CoreV1().Pods(namespace).Delete(pod.ObjectMeta.Name, nil)
			logErr("cleaning up pod", err)
			// ignore err
		}
		// Backward compatibility with older label name
		pid, pok := pod.ObjectMeta.Labels[fission.POOLMGR_INSTANCEID_LABEL]
		if pok && pid != instanceId {
			log.Printf("Cleaning up pod %v", pod.ObjectMeta.Name)
			err := client.CoreV1().Pods(namespace).Delete(pod.ObjectMeta.Name, nil)
			logErr("cleaning up pod", err)
			// ignore err
		}

	}
	return nil
}

func cleanupServices(client *kubernetes.Clientset, namespace string, instanceId string) error {
	svcList, err := client.CoreV1().Services(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}
	for _, svc := range svcList.Items {
		id, ok := svc.ObjectMeta.Labels[fission.EXECUTOR_INSTANCEID_LABEL]
		if ok && id != instanceId {
			log.Printf("Cleaning up svc %v", svc.ObjectMeta.Name)
			err := client.CoreV1().Services(namespace).Delete(svc.ObjectMeta.Name, nil)
			logErr("cleaning up service", err)
			// ignore err
		}
	}
	return nil
}

func cleanupHpa(client *kubernetes.Clientset, namespace string, instanceId string) error {
	hpaList, err := client.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(meta_v1.ListOptions{})
	if err != nil {
		return err
	}

	for _, hpa := range hpaList.Items {
		id, ok := hpa.ObjectMeta.Labels[fission.EXECUTOR_INSTANCEID_LABEL]
		if ok && id != instanceId {
			log.Printf("Cleaning up HPA %v", hpa.ObjectMeta.Name)
			err := client.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(hpa.ObjectMeta.Name, nil)
			logErr("cleaning up HPA", err)
		}

	}
	return nil

}

func logErr(msg string, err error) {
	if err != nil {
		log.Printf("Error %v: %v", msg, err)
	}
}

// CleanupRoleBindings periodically lists rolebindings across all namespaces and removes Service Accounts from them or
// deletes the rolebindings completely if there are no Service Accounts in a rolebinding object.
func CleanupRoleBindings(client *kubernetes.Clientset, fissionClient *crd.FissionClient, functionNs, envBuilderNs string, cleanupRoleBindingInterval time.Duration) {
	for {
		log.Println("Starting cleanupRoleBindings cycle")
		// get all rolebindings ( just to be efficient, one call to kubernetes )
		rbList, err := client.RbacV1beta1().RoleBindings(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
		if err != nil {
			// something wrong, but next iteration hopefully succeeds
			log.Printf("Error listing rolebindings in all ns, err: %v", err)
			continue
		}

		// go through each role-binding object and do the cleanup necessary
		for _, roleBinding := range rbList.Items {
			// ignore role-bindings in kube-system namespace
			if roleBinding.Namespace == "kube-system" {
				continue
			}

			// ignore role-bindings not created by fission
			if roleBinding.Name != fission.PackageGetterRB && roleBinding.Name != fission.SecretConfigMapGetterRB {
				continue
			}

			// in order to find out if there are any functions that need this role-binding in role-binding namespace,
			// we can list the functions once per role-binding.
			funcList, err := fissionClient.Functions(roleBinding.Namespace).List(meta_v1.ListOptions{})
			if err != nil {
				log.Printf("Error fetching environment list in : %s", roleBinding.Namespace)
				continue
			}

			// final map of service accounts that can be removed from this roleBinding object
			// using a map here instead of a list so the code in RemoveSAFromRoleBindingWithRetries is efficient.
			saToRemove := make(map[string]bool)

			// the following flags are needed to decide if any of the service accounts can be removed from role-bindings depending on the functions that need them.
			// ndmFunc denotes if there's at least one function that has executor type New deploy Manager
			// funcEnvReference denotes if there's at least one function that has reference to an environment in the SA Namespace for the SA in question
			var ndmFunc, funcEnvReference bool

			// iterate through each subject in the role-binding and check if there are any references to them
			for _, subj := range roleBinding.Subjects {
				ndmFunc = false
				funcEnvReference = false

				// this is the reverse of what we're doing in setting up of role-bindings. if objects are created in default ns,
				// the SA namespace will have the value of "fission-function"/"fission-builder" depending on the SA.
				// so now we need to look for the objects in default namespace.
				saNs := subj.Namespace
				if subj.Namespace == functionNs ||
					subj.Namespace == envBuilderNs {
					saNs = meta_v1.NamespaceDefault
				}

				// go through each function and find out if there's either at least one function with env reference in the same namespace as the Service Account in this iteration
				// or at least one function using ndm executor in the role-binding namespace and set the corresponding flags
				for _, fn := range funcList.Items {
					if fn.Spec.Environment.Namespace == saNs {
						funcEnvReference = true
						break
					}

					if fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
						ndmFunc = true
						break
					}
				}

				// if its a package-getterr-rb, we have 2 kinds of SAs and each of them is handled differently
				// else if its a secret-configmap-rb, we have only one SA which is fission-fetcher
				if roleBinding.Name == fission.PackageGetterRB {
					// check if there is an env obj in saNs
					envList, err := fissionClient.Environments(saNs).List(meta_v1.ListOptions{})
					if err != nil {
						log.Printf("Error fetching environment list in : %s", saNs)
						continue
					}

					// if the SA in this iteration is fission-builder, then we need to only check
					// if either there's at least one env object in the SA's namespace, or,
					// if there's at least one function in the role-binding namespace with env reference
					// to the SA's namespace.
					// if neither, then we can remove this SA from this role-binding
					if subj.Name == fission.FissionBuilderSA {
						if len(envList.Items) == 0 && !funcEnvReference {
							saToRemove[fission.MakeSAMapKey(subj.Name, subj.Namespace)] = true
						}
					}

					// if the SA in this iteration is fission-fetcher, then in addition to above checks,
					// we also need to check if there's at least one function with executor type New deploy
					// in the rolebinding's namespace.
					// if none of them are true, then remove this SA from this role-binding
					if subj.Name == fission.FissionFetcherSA {
						if len(envList.Items) == 0 && !ndmFunc && !funcEnvReference {
							// remove SA from rolebinding
							saToRemove[fission.MakeSAMapKey(subj.Name, subj.Namespace)] = true
						}
					}
				} else if roleBinding.Name == fission.SecretConfigMapGetterRB {
					// if there's not even one function in the role-binding's namespace and there's not even
					// one function with env reference to the SA's namespace, then remove that SA
					// from this role-binding
					if !ndmFunc && !funcEnvReference {
						saToRemove[fission.MakeSAMapKey(subj.Name, subj.Namespace)] = true
					}
				}
			}

			// finally, make a call to RemoveSAFromRoleBindingWithRetries for all the service accounts that need to be removed
			// for the role-binding in this iteration
			if len(saToRemove) != 0 {
				log.Printf("saToRemove : %v for rolebinding : %s.%s", saToRemove, roleBinding.Name, roleBinding.Namespace)

				// call this once in the end for each role-binding
				err = fission.RemoveSAFromRoleBindingWithRetries(client, roleBinding.Name, roleBinding.Namespace, saToRemove)
				if err != nil {
					// if there's an error, we just log it and proceed with the next role-binding, hoping that this role-binding
					// will be processed in next iteration.
					log.Printf("Error removing SA : %v from rolebinding : %s.%s, err: %v", saToRemove, roleBinding.Name,
						roleBinding.Namespace, err)
				}
			}
		}

		// some sleep before the next cleanup iteration
		time.Sleep(cleanupRoleBindingInterval)
	}
}
