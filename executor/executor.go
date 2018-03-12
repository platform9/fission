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
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/executor/fscache"
	"github.com/fission/fission/executor/newdeploy"
	"github.com/fission/fission/executor/poolmgr"
)

type (
	Executor struct {
		gpm           *poolmgr.GenericPoolManager
		ndm           *newdeploy.NewDeploy
		functionEnv   *cache.Cache
		fissionClient *crd.FissionClient
		fsCache       *fscache.FunctionServiceCache

		requestChan                chan *createFuncServiceRequest
		fsCreateWg                 map[string]*sync.WaitGroup
		invalidateCacheRequestChan chan *invalidateCacheChanRequest
	}
	createFuncServiceRequest struct {
		funcMeta *metav1.ObjectMeta
		respChan chan *createFuncServiceResponse
	}

	createFuncServiceResponse struct {
		funcSvc *fscache.FuncSvc
		err     error
	}

	invalidateCacheChanRequest struct {
		request  fission.CacheInvalidationRequest
		response chan *CacheInvalidationResponse
	}

	CacheInvalidationResponse struct {
		err error
	}
)

func MakeExecutor(gpm *poolmgr.GenericPoolManager, ndm *newdeploy.NewDeploy, fissionClient *crd.FissionClient, fsCache *fscache.FunctionServiceCache) *Executor {
	executor := &Executor{
		gpm:           gpm,
		ndm:           ndm,
		functionEnv:   cache.MakeCache(10*time.Second, 0),
		fissionClient: fissionClient,
		fsCache:       fsCache,

		requestChan:                make(chan *createFuncServiceRequest),
		fsCreateWg:                 make(map[string]*sync.WaitGroup),
		invalidateCacheRequestChan: make(chan *invalidateCacheChanRequest),
	}
	go executor.serveCreateFuncServices()
	go executor.serveInvalidateCacheEntryRequests()
	return executor
}

// All non-cached function service requests go through this goroutine
// serially. It parallelizes requests for different functions, and
// ensures that for a given function, only one request causes a pod to
// get specialized. In other words, it ensures that when there's an
// ongoing request for a certain function, all other requests wait for
// that request to complete.
func (executor *Executor) serveCreateFuncServices() {
	for {
		req := <-executor.requestChan
		m := req.funcMeta

		// Cache miss -- is this first one to request the func?
		wg, found := executor.fsCreateWg[crd.CacheKey(m)]
		if !found {
			// create a waitgroup for other requests for
			// the same function to wait on
			wg := &sync.WaitGroup{}
			wg.Add(1)
			executor.fsCreateWg[crd.CacheKey(m)] = wg

			// launch a goroutine for each request, to parallelize
			// the specialization of different functions
			go func() {
				fsvc, err := executor.createServiceForFunction(m)
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
				delete(executor.fsCreateWg, crd.CacheKey(m))
				wg.Done()
			}()
		} else {
			// There's an existing request for this function, wait for it to finish
			go func() {
				log.Printf("Waiting for concurrent request for the same function: %v", m)
				wg.Wait()

				// get the function service from the cache
				fsvc, err := executor.fsCache.GetByFunction(m)
				req.respChan <- &createFuncServiceResponse{
					funcSvc: fsvc,
					err:     err,
				}
			}()
		}
	}
}

func (executor *Executor) getFunctionExecutorType(meta *metav1.ObjectMeta) (string, error) {
	fn, err := executor.fissionClient.Functions(meta.Namespace).Get(meta.Name)
	if err != nil {
		return "", err
	}
	return string(fn.Spec.InvokeStrategy.ExecutionStrategy.ExecutorType), nil

}

func (executor *Executor) createServiceForFunction(meta *metav1.ObjectMeta) (*fscache.FuncSvc, error) {
	log.Printf("[%v] No cached function service found, creating one", meta.Name)

	// from Func -> get Env
	log.Printf("[%v] getting environment for function", meta.Name)
	env, err := executor.getFunctionEnv(meta)
	if err != nil {
		return nil, err
	}

	executorType, err := executor.getFunctionExecutorType(meta)
	if err != nil {
		return nil, err
	}

	switch executorType {
	case fission.ExecutorTypeNewdeploy:
		fs, err := executor.ndm.GetFuncSvc(meta)
		return fs, err
	default:
		pool, err := executor.gpm.GetPool(env)
		if err != nil {
			return nil, err
		}
		// from GenericPool -> get one function container
		// (this also adds to the cache)
		log.Printf("[%v] getting function service from pool", meta.Name)
		fsvc, err := pool.GetFuncSvc(meta)
		return fsvc, err
	}
}

func (executor *Executor) getFunctionEnv(m *metav1.ObjectMeta) (*crd.Environment, error) {
	var env *crd.Environment

	// Cached ?
	result, err := executor.functionEnv.Get(crd.CacheKey(m))
	if err == nil {
		env = result.(*crd.Environment)
		return env, nil
	}

	// Cache miss -- get func from controller
	f, err := executor.fissionClient.Functions(m.Namespace).Get(m.Name)
	if err != nil {
		return nil, err
	}

	// Get env from metadata
	log.Printf("[%v] getting env", m)
	env, err = executor.fissionClient.Environments(f.Spec.Environment.Namespace).Get(f.Spec.Environment.Name)
	if err != nil {
		return nil, err
	}

	// cache for future lookups
	executor.functionEnv.Set(crd.CacheKey(m), env)

	return env, nil
}

// serveInvalidateCacheEntryRequests is setup to listen to any cache invalidation requests.
// On receiving a request, it first extracts the podAddress from request and checks if there is an entry in the cache for this. If yes, then remove. else, silently ignore.
// This is needed if there is more than 1 router instance and all of them try to send invalidation requests for the same function.
func (executor *Executor) serveInvalidateCacheEntryRequests() {
	for {
		chanReq := <-executor.invalidateCacheRequestChan
		funcMeta := chanReq.request.FunctionMetadata

		log.Printf("Received a request for invalidating cache entry for function %s", funcMeta.Name)

		executorType, err := executor.getFunctionExecutorType(funcMeta)
		if err != nil {
			chanReq.response <- &CacheInvalidationResponse{
				err: err,
			}
		}

		switch executorType {
		case fission.ExecutorTypeNewdeploy:
			// TODO : it appears to me like I don’t have to handle this case for new deploy backend. Explanation :
			// newdeployBacked code is creating a service for each deployment , also creating deployments with replicas set to at least one (if zero input by the user) or number specified by the user and also setting up the fetcher container to specialize on startup,
			// I would expect that if someone deleted any function pod, the deployment will take care of spawning a new function pod and also fetcher will specialize on startup.
			// But I want to be sure I'm not missing anything.
		default:
			// for gpm, we first check if the podIP from the request is same as the podIP in the cache.
			// if not, it means that the cache entry for this function with non-existent podIP is already removed.
			fsvc, err := executor.fsCache.GetByFunction(funcMeta)
			if err != nil {
				log.Printf("Error getting function %s object", funcMeta.Name)
				chanReq.response <- &CacheInvalidationResponse{
					err: err,
				}
			}

			if chanReq.request.FunctionPodAddress == fsvc.Address {
				log.Printf("Deleting cache entry for function : %s, address : %s", fsvc.Name, fsvc.Address)
				executor.fsCache.DeleteEntry(fsvc)
				chanReq.response <- &CacheInvalidationResponse{
					err: nil,
				}
			}
		}
	}
}

func dumpStackTrace() {
	debug.PrintStack()
}

// StartExecutor Starts executor and the executor components such as Poolmgr,
// deploymgr and potential future executor types
func StartExecutor(fissionNamespace string, functionNamespace string, port int) error {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fissionClient, kubernetesClient, _, err := crd.MakeFissionClient()
	restClient := fissionClient.GetCrdClient()
	if err != nil {
		log.Printf("Failed to get kubernetes client: %v", err)
		return err
	}

	fsCache := fscache.MakeFunctionServiceCache()

	poolID := strings.ToLower(uniuri.NewLen(8))
	cleanupObjects(kubernetesClient, functionNamespace, poolID)
	go idleObjectReaper(kubernetesClient, fissionClient, fsCache, time.Minute*2)
	gpm := poolmgr.MakeGenericPoolManager(
		fissionClient, kubernetesClient, fissionNamespace,
		functionNamespace, fsCache, poolID)

	ndm := newdeploy.MakeNewDeploy(
		fissionClient, kubernetesClient, restClient,
		functionNamespace, fsCache, poolID)

	api := MakeExecutor(gpm, ndm, fissionClient, fsCache)

	go api.Serve(port)

	return nil
}
