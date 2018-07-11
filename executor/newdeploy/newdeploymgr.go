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
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/fscache"
)

type (
	requestType int

	NewDeploy struct {
		kubernetesClient *kubernetes.Clientset
		fissionClient    *crd.FissionClient
		crdClient        *rest.RESTClient
		instanceID       string

		fetcherImg             string
		fetcherImagePullPolicy apiv1.PullPolicy
		namespace              string
		sharedMountPath        string
		sharedSecretPath       string
		sharedCfgMapPath       string
		useIstio               bool

		fsCache        *fscache.FunctionServiceCache // cache funcSvc's by function, address and podname
		requestChannel chan *fnRequest

		functions      []crd.Function
		funcStore      k8sCache.Store
		funcController k8sCache.Controller

		idlePodReapTime time.Duration
	}

	fnRequest struct {
		reqType         requestType
		fn              *crd.Function
		responseChannel chan *fnResponse
	}

	fnResponse struct {
		error
		fSvc *fscache.FuncSvc
	}
)

const (
	FnCreate requestType = iota
	FnUpdate
	FnDelete
)

func MakeNewDeploy(
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	crdClient *rest.RESTClient,
	namespace string,
	fsCache *fscache.FunctionServiceCache,
	instanceID string,
	idlePodReapTime time.Duration,
) *NewDeploy {

	log.Printf("Creating NewDeploy ExecutorType")

	fetcherImg := os.Getenv("FETCHER_IMAGE")
	if len(fetcherImg) == 0 {
		fetcherImg = "fission/fetcher"
	}
	fetcherImagePullPolicy := os.Getenv("FETCHER_IMAGE_PULL_POLICY")
	if len(fetcherImagePullPolicy) == 0 {
		fetcherImagePullPolicy = "IfNotPresent"
	}

	enableIstio := false
	if len(os.Getenv("ENABLE_ISTIO")) > 0 {
		istio, err := strconv.ParseBool(os.Getenv("ENABLE_ISTIO"))
		if err != nil {
			log.Println("Failed to parse ENABLE_ISTIO")
		}
		enableIstio = istio
	}

	nd := &NewDeploy{
		fissionClient:    fissionClient,
		kubernetesClient: kubernetesClient,
		crdClient:        crdClient,
		instanceID:       instanceID,

		namespace: namespace,
		fsCache:   fsCache,

		fetcherImg:             fetcherImg,
		fetcherImagePullPolicy: apiv1.PullIfNotPresent,
		sharedMountPath:        "/userfunc",
		sharedSecretPath:       "/secrets",
		sharedCfgMapPath:       "/configs",
		useIstio:               enableIstio,

		requestChannel:  make(chan *fnRequest),
		idlePodReapTime: idlePodReapTime,
	}

	if nd.crdClient != nil {
		fnStore, fnController := nd.initFuncController()
		nd.funcStore = fnStore
		nd.funcController = fnController
	}

	return nd
}

func (deploy *NewDeploy) Run(ctx context.Context) {
	go deploy.service()
	go deploy.funcController.Run(ctx.Done())
	go deploy.idleObjectReaper()
}

func (deploy *NewDeploy) initFuncController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(deploy.crdClient, "functions", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Function{}, resyncPeriod, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fn := obj.(*crd.Function)
			deploy.createFunction(fn)
		},
		DeleteFunc: func(obj interface{}) {
			fn := obj.(*crd.Function)
			deploy.deleteFunction(fn)
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldFn := oldObj.(*crd.Function)
			newFn := newObj.(*crd.Function)
			deploy.fnUpdate(oldFn, newFn)
		},
	})
	return store, controller
}

func (deploy *NewDeploy) service() {
	for {
		req := <-deploy.requestChannel
		switch req.reqType {
		case FnCreate:
			fsvc, err := deploy.fnCreate(req.fn)
			req.responseChannel <- &fnResponse{
				error: err,
				fSvc:  fsvc,
			}
			continue
		case FnDelete:
			_, err := deploy.fnDelete(req.fn)
			req.responseChannel <- &fnResponse{
				error: err,
				fSvc:  nil,
			}
			continue
			// Update needs two inputs and will be called directly by controller
		}
	}
}

func (deploy *NewDeploy) GetFuncSvc(metadata *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	c := make(chan *fnResponse)
	fn, err := deploy.fissionClient.Functions(metadata.Namespace).Get(metadata.Name)
	if err != nil {
		return nil, err
	}

	deploy.requestChannel <- &fnRequest{
		fn:              fn,
		reqType:         FnCreate,
		responseChannel: c,
	}

	resp := <-c
	if resp.error != nil {
		return nil, resp.error
	}
	return resp.fSvc, nil
}

func (deploy *NewDeploy) createFunction(fn *crd.Function) {
	if fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy {
		return
	}
	if fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale <= 0 {
		return
	}
	// Eager creation of function if minScale is greater than 0
	log.Printf("Eagerly creating newDeploy objects for function")
	c := make(chan *fnResponse)
	deploy.requestChannel <- &fnRequest{
		fn:              fn,
		reqType:         FnCreate,
		responseChannel: c,
	}
	resp := <-c
	if resp.error != nil {
		log.Printf("Error eager creating function: %v", resp.error)
	}
}

func (deploy *NewDeploy) updateFunction(fn *crd.Function) {
	c := make(chan *fnResponse)
	deploy.requestChannel <- &fnRequest{
		fn:              fn,
		reqType:         FnUpdate,
		responseChannel: c,
	}
	resp := <-c
	if resp.error != nil {
		log.Printf("Error eager updating function: %v", resp.error)
	}
}

func (deploy *NewDeploy) deleteFunction(fn *crd.Function) {
	if fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
		c := make(chan *fnResponse)
		deploy.requestChannel <- &fnRequest{
			fn:              fn,
			reqType:         FnDelete,
			responseChannel: c,
		}
		resp := <-c
		if resp.error != nil {
			log.Printf("Error deleing the function: %v", resp.error)
		}
	}
}

func (deploy *NewDeploy) fnCreate(fn *crd.Function) (*fscache.FuncSvc, error) {
	fsvc, err := deploy.fsCache.GetByFunction(&fn.Metadata)
	if err == nil {
		return fsvc, err
	}

	env, err := deploy.fissionClient.
		Environments(fn.Spec.Environment.Namespace).
		Get(fn.Spec.Environment.Name)
	if err != nil {
		return fsvc, err
	}

	objName := deploy.getObjName(fn)

	deployLabels := deploy.getDeployLabels(fn, env)

	// to support backward compatibility, if the function was created in default ns, we fall back to creating the
	// deployment of the function in fission-function ns
	ns := deploy.namespace
	if fn.Metadata.Namespace != metav1.NamespaceDefault {
		ns = fn.Metadata.Namespace
	}

	// Envoy(istio-proxy) returns 404 directly before istio pilot
	// propagates latest Envoy-specific configuration.
	// Since newdeploy waits for pods of deployment to be ready,
	// change the order of kubeObject creation (create service first,
	// then deployment) to take advantage of waiting time.
	svc, err := deploy.createOrGetSvc(deployLabels, objName, ns)
	if err != nil {
		log.Printf("Error creating the service %v: %v", objName, err)
		return fsvc, err
	}
	svcAddress := fmt.Sprintf("%v.%v", svc.Name, svc.Namespace)

	depl, err := deploy.createOrGetDeployment(fn, env, objName, deployLabels, ns)
	if err != nil {
		log.Printf("Error creating the deployment %v: %v", objName, err)
		return fsvc, err
	}

	hpa, err := deploy.createOrGetHpa(objName, &fn.Spec.InvokeStrategy.ExecutionStrategy, depl)
	if err != nil {
		return fsvc, errors.Wrap(err, fmt.Sprintf("error creating the HPA %v:", objName))
	}

	kubeObjRefs := []apiv1.ObjectReference{
		{
			//obj.TypeMeta.Kind does not work hence this, needs investigationa and a fix
			Kind:            "deployment",
			Name:            depl.ObjectMeta.Name,
			APIVersion:      depl.TypeMeta.APIVersion,
			Namespace:       depl.ObjectMeta.Namespace,
			ResourceVersion: depl.ObjectMeta.ResourceVersion,
			UID:             depl.ObjectMeta.UID,
		},
		{
			Kind:            "service",
			Name:            svc.ObjectMeta.Name,
			APIVersion:      svc.TypeMeta.APIVersion,
			Namespace:       svc.ObjectMeta.Namespace,
			ResourceVersion: svc.ObjectMeta.ResourceVersion,
			UID:             svc.ObjectMeta.UID,
		},
		{
			Kind:            "horizontalpodautoscaler",
			Name:            hpa.ObjectMeta.Name,
			APIVersion:      hpa.TypeMeta.APIVersion,
			Namespace:       hpa.ObjectMeta.Namespace,
			ResourceVersion: hpa.ObjectMeta.ResourceVersion,
			UID:             hpa.ObjectMeta.UID,
		},
	}

	fsvc = &fscache.FuncSvc{
		Name:              objName,
		Function:          &fn.Metadata,
		Environment:       env,
		Address:           svcAddress,
		KubernetesObjects: kubeObjRefs,
		Executor:          fscache.NEWDEPLOY,
	}

	_, err = deploy.fsCache.Add(*fsvc)
	if err != nil {
		log.Printf("Error adding the function to cache: %v", err)
		return fsvc, err
	}
	return fsvc, nil
}

func (deploy *NewDeploy) fnUpdate(oldFn *crd.Function, newFn *crd.Function) {

	if oldFn.Metadata.ResourceVersion == newFn.Metadata.ResourceVersion {
		return
	}

	// Ignoring updates to functions which are not of NewDeployment type
	if newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
		oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy {
		return
	}

	deployChanged := false

	if oldFn.Spec.InvokeStrategy != newFn.Spec.InvokeStrategy {

		// Executor type is no longer New Deployment
		if newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
			oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
			log.Printf("function does not use new deployment executor anymore, deleting resources: %v", newFn)
			// IMP - pass the oldFn, as the new/modified function is not in cache
			deploy.fnDelete(oldFn)
			return
		}

		// Executor type changed to New Deployment from something else
		if oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
			newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
			log.Printf("function type changed to new deployment, creating resources: %v", newFn)
			_, err := deploy.fnCreate(newFn)
			if err != nil {
				updateStatus(oldFn, err, "error changing the function's type to newdeploy")
			}
			return
		}

		hpa, err := deploy.getHpa(newFn)
		if err != nil {
			updateStatus(oldFn, err, "error getting HPA while updating function")
			return
		}

		hpaChanged := false

		if newFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale != oldFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale {
			replicas := int32(newFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)
			hpa.Spec.MinReplicas = &replicas
			deployChanged = true
			hpaChanged = true
		}

		if newFn.Spec.InvokeStrategy.ExecutionStrategy.MaxScale != oldFn.Spec.InvokeStrategy.ExecutionStrategy.MaxScale {
			hpa.Spec.MaxReplicas = int32(newFn.Spec.InvokeStrategy.ExecutionStrategy.MaxScale)
			hpaChanged = true
		}

		if newFn.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent != oldFn.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent {
			targetCpupercent := int32(newFn.Spec.InvokeStrategy.ExecutionStrategy.TargetCPUPercent)
			hpa.Spec.TargetCPUUtilizationPercentage = &targetCpupercent
			hpaChanged = true
		}

		if hpaChanged {
			err := deploy.updateHpa(hpa)
			if err != nil {
				updateStatus(oldFn, err, "error updating HPA while updating function")
				return
			}
		}
	}

	if oldFn.Spec.Environment != newFn.Spec.Environment ||
		oldFn.Spec.Package.PackageRef != newFn.Spec.Package.PackageRef {
		deployChanged = true
	}

	// If length of slice has changed then no need to check individual elements
	if len(oldFn.Spec.Secrets) != len(newFn.Spec.Secrets) {
		deployChanged = true
	} else {
		for i, newSecret := range newFn.Spec.Secrets {
			if newSecret != oldFn.Spec.Secrets[i] {
				deployChanged = true
				break
			}
		}
	}
	if len(oldFn.Spec.ConfigMaps) != len(newFn.Spec.ConfigMaps) {
		deployChanged = true
	} else {
		for i, newConfig := range newFn.Spec.ConfigMaps {
			if newConfig != oldFn.Spec.ConfigMaps[i] {
				deployChanged = true
				break
			}
		}
	}

	if deployChanged == true {
		env, err := deploy.fissionClient.Environments(newFn.Spec.Environment.Namespace).
			Get(newFn.Spec.Environment.Name)
		if err != nil {
			updateStatus(oldFn, err, "failed to get environment while updating function")
			return
		}
		deployName := deploy.getObjName(oldFn)
		deployLabels := deploy.getDeployLabels(oldFn, env)
		log.Printf("updating deployment due to function update")
		newDeployment, err := deploy.getDeploymentSpec(newFn, env, deployName, deployLabels)
		if err != nil {
			updateStatus(oldFn, err, "failed to get new deployment spec while updating function")
			return
		}

		// to support backward compatibility, if the function was created in default ns, we fall back to creating the
		// deployment of the function in fission-function ns
		ns := deploy.namespace
		if newFn.Metadata.Namespace != metav1.NamespaceDefault {
			ns = newFn.Metadata.Namespace
		}

		err = deploy.updateDeployment(newDeployment, ns)
		if err != nil {
			updateStatus(oldFn, err, "failed to update deployment while updating function")
			return
		}
	}
}

func (deploy *NewDeploy) fnDelete(fn *crd.Function) (*fscache.FuncSvc, error) {

	var delError error

	// GetByFunction uses resource version as part of cache key, however,
	// the resource version in function metadata will be changed when a function
	// is deleted and cause newdeploy backend fails to delete the entry.
	// Use GetByFunctionUID instead of GetByFunction here to find correct
	// fsvc entry.
	fsvc, err := deploy.fsCache.GetByFunctionUID(fn.Metadata.UID)
	if err != nil {
		log.Printf("fsvc not found in cache: %v", fn.Metadata)
		delError = err
		return nil, err
	}

	_, err = deploy.fsCache.DeleteOld(fsvc, time.Second*0)
	if err != nil {
		log.Printf("Error deleting the function from cache: %v", fsvc)
		delError = err
	}
	objName := fsvc.Name

	// to support backward compatibility, if the function was created in default ns, we fall back to creating the
	// deployment of the function in fission-function ns, so cleaning up resources there
	ns := deploy.namespace
	if fn.Metadata.Namespace != metav1.NamespaceDefault {
		ns = fn.Metadata.Namespace
	}

	err = deploy.deleteDeployment(ns, objName)
	if err != nil {
		log.Printf("Error deleting the deployment: %v", objName)
		delError = err
	}

	err = deploy.deleteSvc(ns, objName)
	if err != nil {
		log.Printf("Error deleting the service: %v", objName)
		delError = err
	}

	err = deploy.deleteHpa(ns, objName)
	if err != nil {
		log.Printf("Error deleting the HPA: %v", objName)
		delError = err
	}

	if delError != nil {
		return nil, delError
	}

	return nil, nil
}

func (deploy *NewDeploy) getObjName(fn *crd.Function) string {
	return fmt.Sprintf("%v-%v",
		fn.Metadata.Name,
		deploy.instanceID)
}

func (deploy *NewDeploy) getDeployLabels(fn *crd.Function, env *crd.Environment) map[string]string {
	return map[string]string{
		"environmentName":                 env.Metadata.Name,
		"environmentUid":                  string(env.Metadata.UID),
		"functionName":                    fn.Metadata.Name,
		"functionUid":                     string(fn.Metadata.UID),
		fission.EXECUTOR_INSTANCEID_LABEL: deploy.instanceID,
		"executorType":                    fission.ExecutorTypeNewdeploy,
	}
}

// updateKubeObjRefRV update the resource version of kubeObjectRef with
// given kind and return error if failed to find the reference.
func (deploy *NewDeploy) updateKubeObjRefRV(fsvc *fscache.FuncSvc, objKind string, rv string) error {
	kubeObjs := fsvc.KubernetesObjects
	for i, obj := range kubeObjs {
		if obj.Kind == objKind {
			kubeObjs[i].ResourceVersion = rv
			return nil
		}
	}
	fsvc.KubernetesObjects = kubeObjs
	return errors.New(fmt.Sprintf("error finding kubernetes object reference with kind: %v", objKind))
}

// updateStatus is a function which updates status of update.
// Current implementation only logs messages, in future it will update function status
func updateStatus(fn *crd.Function, err error, message string) {
	log.Printf(message, err)
}

// IsValid does a get on the service address to ensure it's a valid service, then
// scale deployment to 1 replica if there are no available replicas for function.
// Return true if no error occurs, return false otherwise.
func (deploy *NewDeploy) IsValid(fsvc *fscache.FuncSvc) bool {
	service := strings.Split(fsvc.Address, ".")
	if len(service) == 0 {
		return false
	}

	_, err := deploy.kubernetesClient.CoreV1().Services(service[1]).Get(service[0], metav1.GetOptions{})
	if err != nil {
		log.Printf("Error validating service address for function %v: %v", fsvc.Function.Name, err)
		return false
	}

	deployObj := getDeploymentObj(fsvc.KubernetesObjects)
	if deployObj == nil {
		log.Printf("Deployment obj for function %v does not exist", fsvc.Function.Name)
		return false
	}

	currentDeploy, err := deploy.kubernetesClient.ExtensionsV1beta1().
		Deployments(deployObj.Namespace).Get(deployObj.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error validating deployment for function %v: %v", fsvc.Function.Name, err)
		return false
	}

	// return directly when available replicas > 0
	if currentDeploy.Status.AvailableReplicas > 0 {
		return true
	}

	return false
}

// idleObjectReaper reaps objects after certain idle time
func (deploy *NewDeploy) idleObjectReaper() {

	pollSleep := time.Duration(2 * time.Minute)
	for {
		time.Sleep(pollSleep)

		envs, err := deploy.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Failed to get environment list: %v", err)
		}

		envList := make(map[types.UID]struct{})
		for _, env := range envs.Items {
			envList[env.Metadata.UID] = struct{}{}
		}

		funcSvcs, err := deploy.fsCache.ListOld(deploy.idlePodReapTime)
		if err != nil {
			log.Printf("Error reaping idle pods: %v", err)
			continue
		}

		for _, fsvc := range funcSvcs {
			if fsvc.Executor != fscache.NEWDEPLOY {
				continue
			}

			if _, ok := envList[fsvc.Environment.Metadata.UID]; !ok {
				log.Printf("Environment %v for function %v no longer exists",
					fsvc.Environment.Metadata.Name, fsvc.Name)
			}

			fn, err := deploy.fissionClient.Functions(fsvc.Function.Namespace).Get(fsvc.Function.Name)
			if err != nil {
				// Newdeploy manager handles the function delete event and clean cache/kubeobjs itself,
				// so we ignore the not found error for functions with newdeploy executor type here.
				if k8sErrs.IsNotFound(err) && fsvc.Executor == fscache.NEWDEPLOY {
					continue
				}
				log.Printf("Error getting function: %v", fsvc.Function.Name)
				continue
			}

			deployObj := getDeploymentObj(fsvc.KubernetesObjects)
			if deployObj == nil {
				log.Printf("Error finding deployment for function %v: %v", fsvc.Function.Name, err)
				continue
			}

			currentDeploy, err := deploy.kubernetesClient.ExtensionsV1beta1().
				Deployments(deployObj.Namespace).Get(deployObj.Name, metav1.GetOptions{})
			if err != nil {
				log.Printf("Error validating deployment for function %v: %v", fsvc.Function.Name, err)
				continue
			}

			replicas := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)

			// do nothing if the current replicas is already lower than minScale
			if *currentDeploy.Spec.Replicas <= replicas {
				continue
			}

			err = scaleDeployment(deploy.kubernetesClient, deployObj.Namespace, deployObj.Name, replicas)
			if err != nil {
				log.Printf("Error scaling down deployment for function %v: %v", fsvc.Function.Name, err)
			}
		}
	}
}

func getDeploymentObj(kubeobjs []apiv1.ObjectReference) *apiv1.ObjectReference {
	for _, kubeobj := range kubeobjs {
		switch strings.ToLower(kubeobj.Kind) {
		case "deployment":
			return &kubeobj
		}
	}
	return nil
}

func scaleDeployment(client *kubernetes.Clientset, deplNS string, deplName string, replicas int32) error {
	log.Printf("Scale deployment %v in namespace %v to replicas %v", deplName, deplNS, replicas)
	_, err := client.ExtensionsV1beta1().Deployments(deplNS).UpdateScale(deplName, &v1beta1.Scale{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deplName,
			Namespace: deplNS,
		},
		Spec: v1beta1.ScaleSpec{
			Replicas: replicas,
		},
	})
	return err
}
