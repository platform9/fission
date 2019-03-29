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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dchest/uniuri"
	"github.com/fission/fission/throttler"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	NewDeploy struct {
		logger *zap.Logger

		kubernetesClient *kubernetes.Clientset
		fissionClient    *crd.FissionClient
		crdClient        *rest.RESTClient
		instanceID       string

		fetcherImg             string
		fetcherImagePullPolicy apiv1.PullPolicy
		runtimeImagePullPolicy apiv1.PullPolicy
		namespace              string
		sharedMountPath        string
		sharedSecretPath       string
		sharedCfgMapPath       string
		useIstio               bool
		collectorEndpoint      string

		fsCache *fscache.FunctionServiceCache // cache funcSvc's by function, address and pod name

		throttler      *throttler.Throttler
		funcStore      k8sCache.Store
		funcController k8sCache.Controller

		idlePodReapTime time.Duration
	}
)

func MakeNewDeploy(
	logger *zap.Logger,
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	crdClient *rest.RESTClient,
	namespace string,
	instanceID string,
) *NewDeploy {

	logger.Info("creating NewDeploy ExecutorType")

	fetcherImg := os.Getenv("FETCHER_IMAGE")
	if len(fetcherImg) == 0 {
		fetcherImg = "fission/fetcher"
	}

	collectorEndpoint := os.Getenv("TRACE_JAEGER_COLLECTOR_ENDPOINT")

	enableIstio := false
	if len(os.Getenv("ENABLE_ISTIO")) > 0 {
		istio, err := strconv.ParseBool(os.Getenv("ENABLE_ISTIO"))
		if err != nil {
			logger.Info("failed to parse 'ENABLE_ISTIO'")
		}
		enableIstio = istio
	}

	nd := &NewDeploy{
		logger: logger.Named("new_deploy"),

		fissionClient:    fissionClient,
		kubernetesClient: kubernetesClient,
		crdClient:        crdClient,
		instanceID:       instanceID,

		namespace: namespace,
		fsCache:   fscache.MakeFunctionServiceCache(logger),
		throttler: throttler.MakeThrottler(1 * time.Minute),

		fetcherImg:             fetcherImg,
		fetcherImagePullPolicy: fission.GetImagePullPolicy(os.Getenv("FETCHER_IMAGE_PULL_POLICY")),
		runtimeImagePullPolicy: fission.GetImagePullPolicy(os.Getenv("RUNTIME_IMAGE_PULL_POLICY")),
		sharedMountPath:        "/userfunc",
		sharedSecretPath:       "/secrets",
		sharedCfgMapPath:       "/configs",
		collectorEndpoint:      collectorEndpoint,
		useIstio:               enableIstio,

		idlePodReapTime: 2 * time.Minute,
	}

	if nd.crdClient != nil {
		fnStore, fnController := nd.initFuncController()
		nd.funcStore = fnStore
		nd.funcController = fnController
	}

	return nd
}

func (deploy *NewDeploy) Run(ctx context.Context) {
	//go deploy.service()
	go deploy.funcController.Run(ctx.Done())
	go deploy.idleObjectReaper()
}

func (deploy *NewDeploy) initFuncController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(deploy.crdClient, "functions", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Function{}, resyncPeriod, k8sCache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fn := obj.(*crd.Function)
			_, err := deploy.createFunction(fn, true)
			if err != nil {
				deploy.logger.Error("error eager creating function",
					zap.Error(err),
					zap.Any("function", fn))
			}
		},
		DeleteFunc: func(obj interface{}) {
			fn := obj.(*crd.Function)
			err := deploy.deleteFunction(fn)
			if err != nil {
				deploy.logger.Error("error deleting function",
					zap.Error(err),
					zap.Any("function", fn))
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldFn := oldObj.(*crd.Function)
			newFn := newObj.(*crd.Function)
			err := deploy.updateFunction(oldFn, newFn)
			if err != nil {
				deploy.logger.Error("error updating function",
					zap.Error(err),
					zap.Any("old_function", oldFn),
					zap.Any("new_function", newFn))
			}
		},
	})
	return store, controller
}

func (deploy *NewDeploy) GetFuncSvc(ctx context.Context, metadata *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	fn, err := deploy.fissionClient.Functions(metadata.Namespace).Get(metadata.Name)
	if err != nil {
		return nil, err
	}
	return deploy.createFunction(fn, false)
}

func (deploy *NewDeploy) createFunction(fn *crd.Function, firstcreate bool) (*fscache.FuncSvc, error) {
	if fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy {
		return nil, nil
	}

	fsvcObj, err := deploy.throttler.RunOnce(string(fn.Metadata.UID), func(ableToCreate bool) (interface{}, error) {
		if ableToCreate {
			return deploy.fnCreate(fn, firstcreate)
		}
		return deploy.fsCache.GetByFunctionUID(fn.Metadata.UID)
	})

	fsvc, ok := fsvcObj.(*fscache.FuncSvc)
	if !ok {
		deploy.logger.Panic("receive unknown object while creating function - expected pointer of function service object")
	}

	return fsvc, err
}

func (deploy *NewDeploy) deleteFunction(fn *crd.Function) error {
	if fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy {
		return nil
	}
	err := deploy.fnDelete(fn)
	if err != nil {
		err = errors.Wrapf(err, "error deleting kubernetes objects of function %v", fn.Metadata)
	}
	return err
}

func (deploy *NewDeploy) fnCreate(fn *crd.Function, firstcreate bool) (*fscache.FuncSvc, error) {
	env, err := deploy.fissionClient.
		Environments(fn.Spec.Environment.Namespace).
		Get(fn.Spec.Environment.Name)
	if err != nil {
		return nil, err
	}

	objName := deploy.getObjName(fn)
	if !firstcreate {
		// retrieve back the previous obj name for later use.
		fsvc, err := deploy.fsCache.GetByFunctionUID(fn.Metadata.UID)
		if err == nil {
			objName = fsvc.Name
		}
	}
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
		deploy.logger.Error("error creating service", zap.Error(err), zap.String("service", objName))
		go deploy.cleanupNewdeploy(ns, objName)
		return nil, errors.Wrapf(err, "error creating service %v", objName)
	}
	svcAddress := fmt.Sprintf("%v.%v", svc.Name, svc.Namespace)
	depl, err := deploy.createOrGetDeployment(fn, env, objName, deployLabels, ns, firstcreate)
	if err != nil {
		deploy.logger.Error("error creating deployment", zap.Error(err), zap.String("deployment", objName))
		go deploy.cleanupNewdeploy(ns, objName)
		return nil, errors.Wrapf(err, "error creating deployment %v", objName)
	}

	hpa, err := deploy.createOrGetHpa(objName, &fn.Spec.InvokeStrategy.ExecutionStrategy, depl)
	if err != nil {
		deploy.logger.Error("error creating HPA", zap.Error(err), zap.String("hpa", objName))
		go deploy.cleanupNewdeploy(ns, objName)
		return nil, errors.Wrapf(err, "error creating the HPA %v", objName)
	}

	kubeObjRefs := []apiv1.ObjectReference{
		{
			//obj.TypeMeta.Kind does not work hence this, needs investigation and a fix
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

	fsvc := &fscache.FuncSvc{
		Name:              objName,
		Function:          &fn.Metadata,
		Environment:       env,
		Address:           svcAddress,
		KubernetesObjects: kubeObjRefs,
		Executor:          fscache.NEWDEPLOY,
	}

	_, err = deploy.fsCache.Add(*fsvc)
	if err != nil {
		deploy.logger.Error("error adding function to cache", zap.Error(err), zap.Any("function", fsvc.Function))
		return fsvc, err
	}
	return fsvc, nil
}

func (deploy *NewDeploy) updateFunction(oldFn *crd.Function, newFn *crd.Function) error {

	if oldFn.Metadata.ResourceVersion == newFn.Metadata.ResourceVersion {
		return nil
	}

	// Ignoring updates to functions which are not of NewDeployment type
	if newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
		oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy {
		return nil
	}

	// Executor type is no longer New Deployment
	if newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
		oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
		deploy.logger.Info("function does not use new deployment executor anymore, deleting resources",
			zap.Any("function", newFn))
		// IMP - pass the oldFn, as the new/modified function is not in cache
		return deploy.deleteFunction(oldFn)
	}

	// Executor type changed to New Deployment from something else
	if oldFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType != fission.ExecutorTypeNewdeploy &&
		newFn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType == fission.ExecutorTypeNewdeploy {
		deploy.logger.Info("function type changed to new deployment, creating resources",
			zap.Any("old_function", oldFn.Metadata),
			zap.Any("new_function", newFn.Metadata))
		_, err := deploy.createFunction(newFn, true)
		if err != nil {
			deploy.updateStatus(oldFn, err, "error changing the function's type to newdeploy")
		}
		return err
	}

	fsvc, err := deploy.fsCache.GetByFunctionUID(newFn.Metadata.UID)
	if err != nil {
		err = errors.Wrapf(err, "error updating function due to unable to find function service cache: %v", oldFn)
		return err
	}

	fnObjName := fsvc.Name
	deployChanged := false

	if oldFn.Spec.InvokeStrategy != newFn.Spec.InvokeStrategy {

		// to support backward compatibility, if the function was created in default ns, we fall back to creating the
		// deployment of the function in fission-function ns, so cleaning up resources there
		ns := deploy.namespace
		if newFn.Metadata.Namespace != metav1.NamespaceDefault {
			ns = newFn.Metadata.Namespace
		}

		hpa, err := deploy.getHpa(ns, fnObjName)
		if err != nil {
			deploy.updateStatus(oldFn, err, "error getting HPA while updating function")
			return err
		}

		hpaChanged := false

		if newFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale != oldFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale {
			replicas := int32(newFn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)
			hpa.Spec.MinReplicas = &replicas
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
				deploy.updateStatus(oldFn, err, "error updating HPA while updating function")
				return err
			}
		}
	}

	if oldFn.Spec.Environment != newFn.Spec.Environment ||
		oldFn.Spec.Package.PackageRef != newFn.Spec.Package.PackageRef ||
		oldFn.Spec.Package.FunctionName != newFn.Spec.Package.FunctionName {
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
			deploy.updateStatus(oldFn, err, "failed to get environment while updating function")
			return err
		}

		deployLabels := deploy.getDeployLabels(oldFn, env)
		deploy.logger.Info("updating deployment due to function update", zap.String("deployment", fnObjName), zap.String("function", newFn.Metadata.Name))
		newDeployment, err := deploy.getDeploymentSpec(newFn, env, fnObjName, deployLabels)
		if err != nil {
			deploy.updateStatus(oldFn, err, "failed to get new deployment spec while updating function")
			return err
		}

		// to support backward compatibility, if the function was created in default ns, we fall back to creating the
		// deployment of the function in fission-function ns
		ns := deploy.namespace
		if newFn.Metadata.Namespace != metav1.NamespaceDefault {
			ns = newFn.Metadata.Namespace
		}

		err = deploy.updateDeployment(newDeployment, ns)
		if err != nil {
			deploy.updateStatus(oldFn, err, "failed to update deployment while updating function")
			return err
		}
	}

	return nil
}

func (deploy *NewDeploy) fnDelete(fn *crd.Function) error {
	var multierr *multierror.Error

	// GetByFunction uses resource version as part of cache key, however,
	// the resource version in function metadata will be changed when a function
	// is deleted and cause newdeploy backend fails to delete the entry.
	// Use GetByFunctionUID instead of GetByFunction here to find correct
	// fsvc entry.
	fsvc, err := deploy.fsCache.GetByFunctionUID(fn.Metadata.UID)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("fsvc not found in cache: %v", fn.Metadata))
		return err
	}

	objName := fsvc.Name

	_, err = deploy.fsCache.DeleteOld(fsvc, time.Second*0)
	if err != nil {
		multierr = multierror.Append(multierr,
			errors.Wrap(err, fmt.Sprintf("error deleting the function from cache")))
	}

	// to support backward compatibility, if the function was created in default ns, we fall back to creating the
	// deployment of the function in fission-function ns, so cleaning up resources there
	ns := deploy.namespace
	if fn.Metadata.Namespace != metav1.NamespaceDefault {
		ns = fn.Metadata.Namespace
	}

	err = deploy.cleanupNewdeploy(ns, objName)
	multierr = multierror.Append(multierr, err)

	return multierr.ErrorOrNil()
}

// getObjName returns a unique name for kubernetes objects of function
func (deploy *NewDeploy) getObjName(fn *crd.Function) string {
	return strings.ToLower(fmt.Sprintf("newdeploy-%v-%v-%v", fn.Metadata.Name, fn.Metadata.Namespace, uniuri.NewLen(8)))
}

func (deploy *NewDeploy) getDeployLabels(fn *crd.Function, env *crd.Environment) map[string]string {
	return map[string]string{
		fission.EXECUTOR_INSTANCEID_LABEL: deploy.instanceID,
		fission.EXECUTOR_TYPE:             fission.ExecutorTypeNewdeploy,
		fission.ENVIRONMENT_NAME:          env.Metadata.Name,
		fission.ENVIRONMENT_NAMESPACE:     env.Metadata.Namespace,
		fission.ENVIRONMENT_UID:           string(env.Metadata.UID),
		fission.FUNCTION_NAME:             fn.Metadata.Name,
		fission.FUNCTION_NAMESPACE:        fn.Metadata.Namespace,
		fission.FUNCTION_UID:              string(fn.Metadata.UID),
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
	return fmt.Errorf("error finding kubernetes object reference with kind: %v", objKind)
}

// updateStatus is a function which updates status of update.
// Current implementation only logs messages, in future it will update function status
func (deploy *NewDeploy) updateStatus(fn *crd.Function, err error, message string) {
	deploy.logger.Info("function status update", zap.Error(err), zap.Any("function", fn), zap.String("message", message))
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
		deploy.logger.Error("error validating function service address", zap.String("function", fsvc.Function.Name), zap.Error(err))
		return false
	}

	deployObj := getDeploymentObj(fsvc.KubernetesObjects)
	if deployObj == nil {
		deploy.logger.Error("deployment obj for function does not exist", zap.String("function", fsvc.Function.Name))
		return false
	}

	currentDeploy, err := deploy.kubernetesClient.ExtensionsV1beta1().
		Deployments(deployObj.Namespace).Get(deployObj.Name, metav1.GetOptions{})
	if err != nil {
		deploy.logger.Error("error validating function deployment", zap.Error(err), zap.String("function", fsvc.Function.Name))
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

	pollSleep := time.Duration(deploy.idlePodReapTime)
	for {
		time.Sleep(pollSleep)

		envs, err := deploy.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			deploy.logger.Fatal("failed to get environment list", zap.Error(err))
		}

		envList := make(map[types.UID]struct{})
		for _, env := range envs.Items {
			envList[env.Metadata.UID] = struct{}{}
		}

		funcSvcs, err := deploy.fsCache.ListOld(deploy.idlePodReapTime)
		if err != nil {
			deploy.logger.Error("error reaping idle pods", zap.Error(err))
			continue
		}

		for _, fsvc := range funcSvcs {
			if fsvc.Executor != fscache.NEWDEPLOY {
				continue
			}

			// For function with the environment that no longer exists, executor
			// scales down the deployment as usual and prints log to notify user.
			if _, ok := envList[fsvc.Environment.Metadata.UID]; !ok {
				deploy.logger.Error("function environment no longer exists",
					zap.String("environment", fsvc.Environment.Metadata.Name),
					zap.String("function", fsvc.Name))
			}

			fn, err := deploy.fissionClient.Functions(fsvc.Function.Namespace).Get(fsvc.Function.Name)
			if err != nil {
				// Newdeploy manager handles the function delete event and clean cache/kubeobjs itself,
				// so we ignore the not found error for functions with newdeploy executor type here.
				if k8sErrs.IsNotFound(err) && fsvc.Executor == fscache.NEWDEPLOY {
					continue
				}
				deploy.logger.Error("error getting function", zap.Error(err), zap.String("function", fsvc.Function.Name))
				continue
			}

			deployObj := getDeploymentObj(fsvc.KubernetesObjects)
			if deployObj == nil {
				deploy.logger.Error("error finding function deployment", zap.Error(err), zap.String("function", fsvc.Function.Name))
				continue
			}

			currentDeploy, err := deploy.kubernetesClient.ExtensionsV1beta1().
				Deployments(deployObj.Namespace).Get(deployObj.Name, metav1.GetOptions{})
			if err != nil {
				deploy.logger.Error("error validating function deployment", zap.Error(err), zap.String("function", fsvc.Function.Name))
				continue
			}

			minScale := int32(fn.Spec.InvokeStrategy.ExecutionStrategy.MinScale)

			// do nothing if the current replicas is already lower than minScale
			if *currentDeploy.Spec.Replicas <= minScale {
				continue
			}

			err = deploy.scaleDeployment(deployObj.Namespace, deployObj.Name, minScale)
			if err != nil {
				deploy.logger.Error("error scaling down function deployment", zap.Error(err), zap.String("function", fsvc.Function.Name))
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

func (deploy *NewDeploy) scaleDeployment(deplNS string, deplName string, replicas int32) error {
	deploy.logger.Info("scaling deployment",
		zap.String("deployment", deplName),
		zap.String("namespace", deplNS),
		zap.Int32("replicas", replicas))
	_, err := deploy.kubernetesClient.ExtensionsV1beta1().Deployments(deplNS).UpdateScale(deplName, &v1beta1.Scale{
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
