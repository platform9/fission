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

package poolmgr

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sTypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/cache"
	"github.com/fission/fission/pkg/crd"
	"github.com/fission/fission/pkg/executor/executortype"
	"github.com/fission/fission/pkg/executor/fscache"
	"github.com/fission/fission/pkg/executor/reaper"
	fetcherConfig "github.com/fission/fission/pkg/fetcher/config"
	"github.com/fission/fission/pkg/utils"
)

var _ executortype.ExecutorType = &GenericPoolManager{}

type requestType int

const (
	GET_POOL requestType = iota
	CLEANUP_POOLS
)

type (
	GenericPoolManager struct {
		logger *zap.Logger

		pools            map[string]*GenericPool
		kubernetesClient *kubernetes.Clientset
		namespace        string

		fissionClient  *crd.FissionClient
		functionEnv    *cache.Cache
		fsCache        *fscache.FunctionServiceCache
		instanceId     string
		requestChannel chan *request

		enableIstio   bool
		fetcherConfig *fetcherConfig.Config

		funcStore      k8sCache.Store
		funcController k8sCache.Controller
		pkgStore       k8sCache.Store
		pkgController  k8sCache.Controller

		defaultIdlePodReapTime time.Duration
	}
	request struct {
		requestType
		env             *fv1.Environment
		envList         []fv1.Environment
		responseChannel chan *response
	}
	response struct {
		error
		pool *GenericPool
	}
)

func MakeGenericPoolManager(
	logger *zap.Logger,
	fissionClient *crd.FissionClient,
	kubernetesClient *kubernetes.Clientset,
	functionNamespace string,
	fetcherConfig *fetcherConfig.Config,
	instanceId string) executortype.ExecutorType {

	gpmLogger := logger.Named("generic_pool_manager")

	gpm := &GenericPoolManager{
		logger:                 gpmLogger,
		pools:                  make(map[string]*GenericPool),
		kubernetesClient:       kubernetesClient,
		namespace:              functionNamespace,
		fissionClient:          fissionClient,
		functionEnv:            cache.MakeCache(10*time.Second, 0),
		fsCache:                fscache.MakeFunctionServiceCache(gpmLogger),
		instanceId:             instanceId,
		requestChannel:         make(chan *request),
		defaultIdlePodReapTime: 2 * time.Minute,
		fetcherConfig:          fetcherConfig,
	}

	go gpm.service()

	if len(os.Getenv("ENABLE_ISTIO")) > 0 {
		istio, err := strconv.ParseBool(os.Getenv("ENABLE_ISTIO"))
		if err != nil {
			gpmLogger.Error("failed to parse 'ENABLE_ISTIO', set to false", zap.Error(err))
		}
		gpm.enableIstio = istio
	}

	gpm.funcStore, gpm.funcController = gpm.makeFuncController(
		gpm.fissionClient, gpm.kubernetesClient, gpm.namespace, gpm.enableIstio)

	gpm.pkgStore, gpm.pkgController = gpm.makePkgController(gpm.fissionClient, gpm.kubernetesClient, gpm.namespace)

	return gpm
}

func (gpm *GenericPoolManager) Run(ctx context.Context) {
	// eagerPoolCreator must run after CleanupOldExecutorObjects.
	// Otherwise, the poolmanager may wrongly delete the deployment.
	go gpm.eagerPoolCreator()
	go gpm.funcController.Run(ctx.Done())
	go gpm.pkgController.Run(ctx.Done())
	go gpm.idleObjectReaper()
}

func (gpm *GenericPoolManager) GetTypeName() fv1.ExecutorType {
	return fv1.ExecutorTypePoolmgr
}

func (gpm *GenericPoolManager) GetFuncSvc(ctx context.Context, fn *fv1.Function) (*fscache.FuncSvc, error) {
	// from Func -> get Env
	gpm.logger.Debug("getting environment for function", zap.String("function", fn.ObjectMeta.Name))
	env, err := gpm.getFunctionEnv(fn)
	if err != nil {
		return nil, err
	}

	pool, err := gpm.getPool(env)
	if err != nil {
		return nil, err
	}

	// from GenericPool -> get one function container
	// (this also adds to the cache)
	gpm.logger.Debug("getting function service from pool", zap.String("function", fn.ObjectMeta.Name))
	return pool.getFuncSvc(ctx, fn)
}

func (gpm *GenericPoolManager) GetFuncSvcFromCache(fn *fv1.Function) (*fscache.FuncSvc, error) {
	return gpm.fsCache.GetFuncSvc(&fn.ObjectMeta)
}

func (gpm *GenericPoolManager) DeleteFuncSvcFromCache(fsvc *fscache.FuncSvc) {
	gpm.fsCache.DeleteFunctionSvc(fsvc)
}

func (gpm *GenericPoolManager) UnTapService(fn *fv1.Function, svcHost string) {
	gpm.fsCache.MarkAvailable(fn, svcHost)
}

func (gpm *GenericPoolManager) GetTotalAvailable(fn *fv1.Function) int {
	return gpm.fsCache.GetTotalAvailable(&fn.ObjectMeta)
}

func (gpm *GenericPoolManager) TapService(svcHost string) error {
	err := gpm.fsCache.TouchByAddress(svcHost)
	if err != nil {
		return err
	}
	return nil
}

// IsValid checks if pod is not deleted and that it has the address passed as the argument. Also checks that all the
// containers in it are reporting a ready status for the healthCheck.
func (gpm *GenericPoolManager) IsValid(fsvc *fscache.FuncSvc) bool {
	for _, obj := range fsvc.KubernetesObjects {
		if strings.ToLower(obj.Kind) == "pod" {
			pod, err := gpm.kubernetesClient.CoreV1().Pods(obj.Namespace).Get(context.Background(), obj.Name, metav1.GetOptions{})
			if err == nil && utils.IsReadyPod(pod) {
				// Normally, the address format is http://[pod-ip]:[port], however, if the
				// Istio is enabled the address format changes to http://[svc-name]:[port].
				// So if the Istio is enabled and pod is in ready state, we return true directly;
				// Otherwise, we need to ensure that the address contains pod ip.
				if gpm.enableIstio ||
					(!gpm.enableIstio && strings.Contains(fsvc.Address, pod.Status.PodIP)) {
					gpm.logger.Debug("valid address",
						zap.String("address", fsvc.Address),
						zap.Any("function", fsvc.Function),
						zap.String("executor", string(fsvc.Executor)),
					)
					return true
				}
			}
		}
	}
	return false
}

func (gpm *GenericPoolManager) RefreshFuncPods(logger *zap.Logger, f fv1.Function) error {

	env, err := gpm.fissionClient.CoreV1().Environments(f.Spec.Environment.Namespace).Get(f.Spec.Environment.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	gp, err := gpm.getPool(env)
	if err != nil {
		return err
	}

	funcSvc, err := gp.fsCache.GetByFunction(&f.ObjectMeta)
	if err != nil {
		return err
	}

	gp.fsCache.DeleteEntry(funcSvc)

	funcLabels := gp.labelsForFunction(&f.ObjectMeta)

	podList, err := gpm.kubernetesClient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set(funcLabels).AsSelector().String(),
	})

	if err != nil {
		return err
	}

	for _, po := range podList.Items {
		err := gpm.kubernetesClient.CoreV1().Pods(po.ObjectMeta.Namespace).Delete(context.Background(), po.ObjectMeta.Name, metav1.DeleteOptions{})
		if k8serrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (gpm *GenericPoolManager) AdoptExistingResources() {
	envs, err := gpm.fissionClient.CoreV1().Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		gpm.logger.Error("error getting environment list", zap.Error(err))
		return
	}

	envMap := make(map[string]fv1.Environment, len(envs.Items))
	wg := &sync.WaitGroup{}

	for i := range envs.Items {
		env := envs.Items[i]

		if gpm.getEnvPoolsize(&env) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := gpm.getPool(&env)
				if err != nil {
					gpm.logger.Error("adopt pool failed", zap.Error(err))
				}
			}()
		}

		// create environment map for later use
		key := fmt.Sprintf("%v/%v", env.ObjectMeta.Namespace, env.ObjectMeta.Name)
		envMap[key] = env
	}

	l := map[string]string{
		fv1.EXECUTOR_TYPE: string(fv1.ExecutorTypePoolmgr),
	}

	podList, err := gpm.kubernetesClient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set(l).AsSelector().String(),
	})

	if err != nil {
		gpm.logger.Error("error getting pod list", zap.Error(err))
		return
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		if !utils.IsReadyPod(pod) {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			// avoid too many requests arrive Kubernetes API server at the same time.
			time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)

			patch := fmt.Sprintf(`{"metadata":{"annotations":{"%v":"%v"}}}`, fv1.EXECUTOR_INSTANCEID_LABEL, gpm.instanceId)
			pod, err = gpm.kubernetesClient.CoreV1().Pods(pod.Namespace).Patch(context.Background(), pod.Name, k8sTypes.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
			if err != nil {
				// just log the error since it won't affect the function serving
				gpm.logger.Warn("error patching executor instance ID of pod", zap.Error(err),
					zap.String("pod", pod.Name), zap.String("ns", pod.Namespace))
				return
			}

			// for unspecialized pod, we only update its annotations
			if pod.Labels["managed"] == "true" {
				return
			}

			fnName, ok1 := pod.Labels[fv1.FUNCTION_NAME]
			fnNS, ok2 := pod.Labels[fv1.FUNCTION_NAMESPACE]
			fnUID, ok3 := pod.Labels[fv1.FUNCTION_UID]
			fnRV, ok4 := pod.Annotations[fv1.FUNCTION_RESOURCE_VERSION]
			envName, ok5 := pod.Labels[fv1.ENVIRONMENT_NAME]
			envNS, ok6 := pod.Labels[fv1.ENVIRONMENT_NAMESPACE]
			svcHost, ok7 := pod.Annotations[fv1.ANNOTATION_SVC_HOST]
			env, ok8 := envMap[fmt.Sprintf("%v/%v", envNS, envName)]

			if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7 && ok8) {
				gpm.logger.Warn("failed to adopt pod for function due to lack of necessary information",
					zap.String("pod", pod.Name), zap.Any("labels", pod.Labels), zap.Any("annotations", pod.Annotations),
					zap.String("env", env.ObjectMeta.Name))
				return
			}

			fsvc := fscache.FuncSvc{
				Name: pod.Name,
				Function: &metav1.ObjectMeta{
					Name:            fnName,
					Namespace:       fnNS,
					UID:             k8sTypes.UID(fnUID),
					ResourceVersion: fnRV,
				},
				Environment: &env,
				Address:     svcHost,
				KubernetesObjects: []apiv1.ObjectReference{
					{
						Kind:            "pod",
						Name:            pod.Name,
						APIVersion:      pod.TypeMeta.APIVersion,
						Namespace:       pod.ObjectMeta.Namespace,
						ResourceVersion: pod.ObjectMeta.ResourceVersion,
						UID:             pod.ObjectMeta.UID,
					},
				},
				Executor: fv1.ExecutorTypePoolmgr,
				Ctime:    time.Now(),
				Atime:    time.Now(),
			}

			_, err = gpm.fsCache.Add(fsvc)
			if err != nil {
				// If fsvc already exists we just skip the duplicate one. And let reaper to recycle the duplicate pods.
				// This is for the case that there are multiple function pods for the same function due to unknown reason.
				if !fscache.IsNameExistError(err) {
					gpm.logger.Warn("failed to adopt pod for function", zap.Error(err), zap.String("pod", pod.Name))
				}

				return
			}

			gpm.logger.Info("adopt function pod",
				zap.String("pod", pod.Name), zap.Any("labels", pod.Labels), zap.Any("annotations", pod.Annotations))
		}()
	}

	wg.Wait()
}

func (gpm *GenericPoolManager) CleanupOldExecutorObjects() {
	gpm.logger.Info("Poolmanager starts to clean orphaned resources", zap.String("instanceID", gpm.instanceId))

	errs := &multierror.Error{}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(map[string]string{fv1.EXECUTOR_TYPE: string(fv1.ExecutorTypePoolmgr)}).AsSelector().String(),
	}

	err := reaper.CleanupDeployments(gpm.logger, gpm.kubernetesClient, gpm.instanceId, listOpts)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	err = reaper.CleanupPods(gpm.logger, gpm.kubernetesClient, gpm.instanceId, listOpts)
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	if errs.ErrorOrNil() != nil {
		// TODO retry reaper; logged and ignored for now
		gpm.logger.Error("Failed to cleanup old executor objects", zap.Error(err))
	}
}

func (gpm *GenericPoolManager) service() {
	for {
		req := <-gpm.requestChannel
		switch req.requestType {
		case GET_POOL:
			// just because they are missing in the cache, we end up creating another duplicate pool.
			var err error
			pool, ok := gpm.pools[crd.CacheKey(&req.env.ObjectMeta)]
			if !ok {
				poolsize := gpm.getEnvPoolsize(req.env)
				switch req.env.Spec.AllowedFunctionsPerContainer {
				case fv1.AllowedFunctionsPerContainerInfinite:
					poolsize = 1
				}

				// To support backward compatibility, if envs are created in default ns, we go ahead
				// and create pools in fission-function ns as earlier.
				ns := gpm.namespace
				if req.env.ObjectMeta.Namespace != metav1.NamespaceDefault {
					ns = req.env.ObjectMeta.Namespace
				}

				pool, err = MakeGenericPool(gpm.logger,
					gpm.fissionClient, gpm.kubernetesClient, req.env, poolsize,
					ns, gpm.namespace, gpm.fsCache, gpm.fetcherConfig, gpm.instanceId, gpm.enableIstio)
				if err != nil {
					req.responseChannel <- &response{error: err}
					continue
				}
				gpm.pools[crd.CacheKey(&req.env.ObjectMeta)] = pool
			}
			req.responseChannel <- &response{pool: pool}
		case CLEANUP_POOLS:
			latestEnvPoolsize := make(map[string]int)
			for _, env := range req.envList {
				latestEnvPoolsize[crd.CacheKey(&env.ObjectMeta)] = int(gpm.getEnvPoolsize(&env))
			}
			for key, pool := range gpm.pools {
				poolsize, ok := latestEnvPoolsize[key]
				if !ok || poolsize == 0 {
					// Env no longer exists or pool size changed to zero

					gpm.logger.Info("destroying generic pool", zap.Any("environment", pool.env.ObjectMeta))
					delete(gpm.pools, key)

					// and delete the pool asynchronously.
					go pool.destroy()
				}
			}
			// no response, caller doesn't wait
		}
	}
}

func (gpm *GenericPoolManager) getPool(env *fv1.Environment) (*GenericPool, error) {
	c := make(chan *response)
	gpm.requestChannel <- &request{
		requestType:     GET_POOL,
		env:             env,
		responseChannel: c,
	}
	resp := <-c
	return resp.pool, resp.error
}

func (gpm *GenericPoolManager) cleanupPools(envs []fv1.Environment) {
	gpm.requestChannel <- &request{
		requestType: CLEANUP_POOLS,
		envList:     envs,
	}
}

func (gpm *GenericPoolManager) getFunctionEnv(fn *fv1.Function) (*fv1.Environment, error) {
	var env *fv1.Environment

	// Cached ?
	// TODO: the cache should be able to search by <env name, fn namespace> instead of function metadata.
	result, err := gpm.functionEnv.Get(crd.CacheKey(&fn.ObjectMeta))
	if err == nil {
		env = result.(*fv1.Environment)
		return env, nil
	}

	// Get env from controller
	env, err = gpm.fissionClient.CoreV1().Environments(fn.Spec.Environment.Namespace).Get(fn.Spec.Environment.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// cache for future lookups
	m := fn.ObjectMeta
	gpm.functionEnv.Set(crd.CacheKey(&m), env)

	return env, nil
}

func (gpm *GenericPoolManager) eagerPoolCreator() {
	pollSleep := 2 * time.Second
	for {
		// get list of envs from controller
		envs, err := gpm.fissionClient.CoreV1().Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if utils.IsNetworkError(err) {
				gpm.logger.Error("encountered network error, retrying", zap.Error(err))
			} else {
				gpm.logger.Error("failed to get environment list", zap.Error(err))
			}
			time.Sleep(5 * time.Second)
			continue
		}

		// Create pools for all envs.  TODO: we should make this a bit less eager, only
		// creating pools for envs that are actually used by functions.  Also we might want
		// to keep these eagerly created pools smaller than the ones created when there are
		// actual function calls.

		wg := &sync.WaitGroup{}

		for i := range envs.Items {
			env := envs.Items[i]
			// Create pool only if poolsize greater than zero
			if gpm.getEnvPoolsize(&env) > 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, err := gpm.getPool(&env)
					if err != nil {
						gpm.logger.Error("eager-create pool failed", zap.Error(err))
					}
				}()
			}
		}

		// Clean up pools whose env was deleted
		gpm.cleanupPools(envs.Items)
		wg.Wait()
		time.Sleep(pollSleep)
	}
}

func (gpm *GenericPoolManager) getEnvPoolsize(env *fv1.Environment) int32 {
	var poolsize int32
	if env.Spec.Version < 3 {
		poolsize = 3
	} else {
		poolsize = int32(env.Spec.Poolsize)
	}
	return poolsize
}

// idleObjectReaper reaps objects after certain idle time
func (gpm *GenericPoolManager) idleObjectReaper() {

	pollSleep := 5 * time.Second
	for {
		time.Sleep(pollSleep)

		envs, err := gpm.fissionClient.CoreV1().Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			gpm.logger.Error("failed to get environment list", zap.Error(err))
			continue
		}

		envList := make(map[k8sTypes.UID]struct{})
		for _, env := range envs.Items {
			envList[env.ObjectMeta.UID] = struct{}{}
		}

		fns, err := gpm.fissionClient.CoreV1().Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			gpm.logger.Error("failed to get environment list", zap.Error(err))
			continue
		}

		fnList := make(map[k8sTypes.UID]fv1.Function)
		for i, fn := range fns.Items {
			fnList[fn.ObjectMeta.UID] = fns.Items[i]
		}

		funcSvcs, err := gpm.fsCache.ListOldForPool(pollSleep)
		if err != nil {
			gpm.logger.Error("error reaping idle pods", zap.Error(err))
			continue
		}

		for i := range funcSvcs {
			fsvc := funcSvcs[i]

			if fsvc.Executor != fv1.ExecutorTypePoolmgr {
				continue
			}

			// For function with the environment that no longer exists, executor
			// cleanups the idle pod as usual and prints log to notify user.
			if _, ok := envList[fsvc.Environment.ObjectMeta.UID]; !ok {
				gpm.logger.Warn("function environment no longer exists",
					zap.String("environment", fsvc.Environment.ObjectMeta.Name),
					zap.String("function", fsvc.Name))
			}

			if fsvc.Environment.Spec.AllowedFunctionsPerContainer == fv1.AllowedFunctionsPerContainerInfinite {
				continue
			}

			idlePodReapTime := gpm.defaultIdlePodReapTime
			if fn, ok := fnList[fsvc.Function.UID]; ok {
				if fn.Spec.IdleTimeout != nil {
					idlePodReapTime = time.Duration(*fn.Spec.IdleTimeout) * time.Second
				}
			}

			if time.Since(fsvc.Atime) < idlePodReapTime {
				continue
			}

			go func() {
				deleted, err := gpm.fsCache.DeleteOldPoolCache(fsvc, idlePodReapTime)
				if err != nil {
					gpm.logger.Error("error deleting Kubernetes objects for function service",
						zap.Error(err),
						zap.Any("service", fsvc))
				}
				if deleted {
					for i := range fsvc.KubernetesObjects {
						gpm.logger.Info("release idle function resources",
							zap.String("function", fsvc.Function.Name),
							zap.String("address", fsvc.Address),
							zap.String("executor", string(fsvc.Executor)),
							zap.String("pod", fsvc.Name),
						)
						reaper.CleanupKubeObject(gpm.logger, gpm.kubernetesClient, &fsvc.KubernetesObjects[i])
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()
		}
	}
}
