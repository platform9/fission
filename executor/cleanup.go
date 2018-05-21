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

package executor

import (
	"fmt"
	"log"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/fscache"
)

var (
	deletePropagation = meta_v1.DeletePropagationBackground
	delOpt            = meta_v1.DeleteOptions{PropagationPolicy: &deletePropagation}
)

// cleanupObjects cleans up resources created by old executortype instances
func cleanupObjects(kubernetesClient *kubernetes.Clientset,
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

// idleObjectReaper reaps objects after certain idle time
func idleObjectReaper(kubeClient *kubernetes.Clientset,
	fissionClient *crd.FissionClient,
	fsCache *fscache.FunctionServiceCache,
	idlePodReapTime time.Duration) {

	pollSleep := time.Duration(2 * time.Minute)
	for {
		time.Sleep(pollSleep)

		envs, err := fissionClient.Environments(meta_v1.NamespaceAll).List(meta_v1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to get environment list: %v", err)
		}

		envList := make(map[types.UID]struct{})
		for i := range envs.Items {
			env := envs.Items[i]
			envList[env.Metadata.UID] = struct{}{}
		}

		funcSvcs, err := fsCache.ListOld(idlePodReapTime)
		if err != nil {
			log.Printf("Error reaping idle pods: %v", err)
			continue
		}

		for _, fsvc := range funcSvcs {
			if _, ok := envList[fsvc.Environment.Metadata.UID]; !ok {
				log.Printf("Environment %v for function %v no longer exists",
					fsvc.Environment.Metadata.Name, fsvc.Name)
			}

			if fsvc.Environment.Spec.AllowedFunctionsPerContainer == fission.AllowedFunctionsPerContainerInfinite {
				continue
			}

			// Newdeploy manager handles the function delete event and clean cache/kubeobjs itself,
			// so we ignore the function service cache with newdepoy executor type here.
			if fsvc.Executor != fscache.NEWDEPLOY {
				deleted, err := fsCache.DeleteOld(fsvc, idlePodReapTime)
				if err != nil {
					log.Printf("Error deleting Kubernetes objects for fsvc '%v': %v", fsvc, err)
					log.Printf("Object Name| Object Kind | Object Namespace")
					for _, kubeobj := range fsvc.KubernetesObjects {
						log.Printf("%v | %v | %v", kubeobj.Name, kubeobj.Kind, kubeobj.Namespace)
					}
				}

				if !deleted {
					continue
				}

				for _, kubeobj := range fsvc.KubernetesObjects {
					deleteKubeobject(kubeClient, &kubeobj)
				}
			}
		}
	}
}

func deleteKubeobject(kubeClient *kubernetes.Clientset, kubeobj *apiv1.ObjectReference) {
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
