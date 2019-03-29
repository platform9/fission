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
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/fscache"
	"github.com/fission/fission/executor/reaper"
)

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

		enableIstio       bool
		collectorEndpoint string

		funcStore      k8sCache.Store
		funcController k8sCache.Controller
		pkgStore       k8sCache.Store
		pkgController  k8sCache.Controller

		idlePodReapTime time.Duration
	}
	request struct {
		requestType
		env             *crd.Environment
		envList         []crd.Environment
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
	instanceId string) *GenericPoolManager {

	gpmLogger := logger.Named("generic_pool_manager")

	gpm := &GenericPoolManager{
		logger:            gpmLogger,
		pools:             make(map[string]*GenericPool),
		kubernetesClient:  kubernetesClient,
		namespace:         functionNamespace,
		fissionClient:     fissionClient,
		functionEnv:       cache.MakeCache(10*time.Second, 0),
		fsCache:           fscache.MakeFunctionServiceCache(gpmLogger),
		instanceId:        instanceId,
		requestChannel:    make(chan *request),
		idlePodReapTime:   2 * time.Minute,
		collectorEndpoint: os.Getenv("TRACE_JAEGER_COLLECTOR_ENDPOINT"),
	}
	go gpm.service()
	go gpm.eagerPoolCreator()

	if len(os.Getenv("ENABLE_ISTIO")) > 0 {
		istio, err := strconv.ParseBool(os.Getenv("ENABLE_ISTIO"))
		if err != nil {
			gpmLogger.Info("failed to parse ENABLE_ISTIO")
		}
		gpm.enableIstio = istio
	}

	gpm.funcStore, gpm.funcController = gpm.makeFuncController(
		gpm.fissionClient, gpm.kubernetesClient, gpm.namespace, gpm.enableIstio)

	gpm.pkgStore, gpm.pkgController = gpm.makePkgController(gpm.fissionClient, gpm.kubernetesClient, gpm.namespace)

	return gpm
}

func (gpm *GenericPoolManager) Run(ctx context.Context) {
	go gpm.funcController.Run(ctx.Done())
	go gpm.pkgController.Run(ctx.Done())
	go gpm.idleObjectReaper()
}

func (gpm *GenericPoolManager) service() {
	for {
		req := <-gpm.requestChannel
		switch req.requestType {
		case GET_POOL:
			// just because they are missing in the cache, we end up creating another duplicate pool.
			var err error
			pool, ok := gpm.pools[crd.CacheKey(&req.env.Metadata)]
			if !ok {
				poolsize := gpm.getEnvPoolsize(req.env)
				switch req.env.Spec.AllowedFunctionsPerContainer {
				case fission.AllowedFunctionsPerContainerInfinite:
					poolsize = 1
				}

				// To support backward compatibility, if envs are created in default ns, we go ahead
				// and create pools in fission-function ns as earlier.
				ns := gpm.namespace
				if req.env.Metadata.Namespace != metav1.NamespaceDefault {
					ns = req.env.Metadata.Namespace
				}

				pool, err = MakeGenericPool(gpm.logger,
					gpm.fissionClient, gpm.kubernetesClient, req.env, poolsize,
					ns, gpm.namespace, gpm.fsCache, gpm.instanceId, gpm.enableIstio,
					gpm.collectorEndpoint)
				if err != nil {
					req.responseChannel <- &response{error: err}
					continue
				}
				gpm.pools[crd.CacheKey(&req.env.Metadata)] = pool
			}
			req.responseChannel <- &response{pool: pool}
		case CLEANUP_POOLS:
			latestEnvPoolsize := make(map[string]int)
			for _, env := range req.envList {
				latestEnvPoolsize[crd.CacheKey(&env.Metadata)] = int(gpm.getEnvPoolsize(&env))
			}
			for key, pool := range gpm.pools {
				poolsize, ok := latestEnvPoolsize[key]
				if !ok || poolsize == 0 {
					// Env no longer exists or pool size changed to zero

					gpm.logger.Info("eestroying generic pool", zap.Any("environment", pool.env.Metadata))
					delete(gpm.pools, key)

					// and delete the pool asynchronously.
					go pool.destroy()
				}
			}
			// no response, caller doesn't wait
		}
	}
}

func (gpm *GenericPoolManager) GetPool(env *crd.Environment) (*GenericPool, error) {
	c := make(chan *response)
	gpm.requestChannel <- &request{
		requestType:     GET_POOL,
		env:             env,
		responseChannel: c,
	}
	resp := <-c
	return resp.pool, resp.error
}

func (gpm *GenericPoolManager) CleanupPools(envs []crd.Environment) {
	gpm.requestChannel <- &request{
		requestType: CLEANUP_POOLS,
		envList:     envs,
	}
}

func (gpm *GenericPoolManager) GetFuncSvc(ctx context.Context, metadata *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	// from Func -> get Env
	gpm.logger.Info("getting environment for function", zap.String("function", metadata.Name))
	env, err := gpm.getFunctionEnv(metadata)
	if err != nil {
		return nil, err
	}

	pool, err := gpm.GetPool(env)
	if err != nil {
		return nil, err
	}
	// from GenericPool -> get one function container
	// (this also adds to the cache)
	gpm.logger.Info("getting function service from pool", zap.String("function", metadata.Name))
	return pool.GetFuncSvc(ctx, metadata)
}

func (gpm *GenericPoolManager) getFunctionEnv(m *metav1.ObjectMeta) (*crd.Environment, error) {
	var env *crd.Environment

	// Cached ?
	result, err := gpm.functionEnv.Get(crd.CacheKey(m))
	if err == nil {
		env = result.(*crd.Environment)
		return env, nil
	}

	// Cache miss -- get func from controller
	f, err := gpm.fissionClient.Functions(m.Namespace).Get(m.Name)
	if err != nil {
		return nil, err
	}

	// Get env from metadata
	gpm.logger.Info("getting env", zap.Any("function", m))
	env, err = gpm.fissionClient.Environments(f.Spec.Environment.Namespace).Get(f.Spec.Environment.Name)
	if err != nil {
		return nil, err
	}

	// cache for future lookups
	gpm.functionEnv.Set(crd.CacheKey(m), env)

	return env, nil
}

func (gpm *GenericPoolManager) eagerPoolCreator() {
	pollSleep := time.Duration(2 * time.Second)
	for {
		// get list of envs from controller
		envs, err := gpm.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			if fission.IsNetworkError(err) {
				gpm.logger.Error("encountered network error, retrying", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}
			gpm.logger.Fatal("failed to get environment list", zap.Error(err))
		}

		// Create pools for all envs.  TODO: we should make this a bit less eager, only
		// creating pools for envs that are actually used by functions.  Also we might want
		// to keep these eagerly created pools smaller than the ones created when there are
		// actual function calls.
		for i := range envs.Items {
			env := envs.Items[i]
			// Create pool only if poolsize greater than zero
			if gpm.getEnvPoolsize(&env) > 0 {
				_, err := gpm.GetPool(&envs.Items[i])
				if err != nil {
					gpm.logger.Error("eager-create pool failed", zap.Error(err))
				}
			}
		}

		// Clean up pools whose env was deleted
		gpm.CleanupPools(envs.Items)
		time.Sleep(pollSleep)
	}
}

func (gpm *GenericPoolManager) getEnvPoolsize(env *crd.Environment) int32 {
	var poolsize int32
	if env.Spec.Version < 3 {
		poolsize = 3
	} else {
		poolsize = int32(env.Spec.Poolsize)
	}
	return poolsize
}

// IsValid checks if pod is not deleted and that it has the address passed as the argument. Also checks that all the
// containers in it are reporting a ready status for the healthCheck.
func (gpm *GenericPoolManager) IsValid(fsvc *fscache.FuncSvc) bool {
	for _, obj := range fsvc.KubernetesObjects {
		if obj.Kind == "pod" {
			pod, err := gpm.kubernetesClient.CoreV1().Pods(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
			if err == nil && strings.Contains(fsvc.Address, pod.Status.PodIP) && fission.IsReadyPod(pod) {
				gpm.logger.Info("valid pod address", zap.String("address", fsvc.Address))
				return true
			}
		}
	}
	return false
}

// idleObjectReaper reaps objects after certain idle time
func (gpm *GenericPoolManager) idleObjectReaper() {

	pollSleep := time.Duration(gpm.idlePodReapTime)
	for {
		time.Sleep(pollSleep)

		envs, err := gpm.fissionClient.Environments(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			gpm.logger.Fatal("failed to get environment list", zap.Error(err))
		}

		envList := make(map[types.UID]struct{})
		for _, env := range envs.Items {
			envList[env.Metadata.UID] = struct{}{}
		}

		funcSvcs, err := gpm.fsCache.ListOld(gpm.idlePodReapTime)
		if err != nil {
			gpm.logger.Error("error reaping idle pods", zap.Error(err))
			continue
		}

		for _, fsvc := range funcSvcs {
			if fsvc.Executor != fscache.POOLMGR {
				continue
			}

			// For function with the environment that no longer exists, executor
			// cleanups the idle pod as usual and prints log to notify user.
			if _, ok := envList[fsvc.Environment.Metadata.UID]; !ok {
				gpm.logger.Info("function environment no longer exists",
					zap.String("environment", fsvc.Environment.Metadata.Name),
					zap.String("function", fsvc.Name))
			}

			if fsvc.Environment.Spec.AllowedFunctionsPerContainer == fission.AllowedFunctionsPerContainerInfinite {
				continue
			}

			deleted, err := gpm.fsCache.DeleteOld(fsvc, gpm.idlePodReapTime)
			if err != nil {
				gpm.logger.Error("error deleting Kubernetes objects for function service",
					zap.Error(err),
					zap.Any("service", fsvc))
			}

			if !deleted {
				continue
			}

			for _, kubeobj := range fsvc.KubernetesObjects {
				reaper.CleanupKubeObject(gpm.logger, gpm.kubernetesClient, &kubeobj)
			}
		}
	}
}
