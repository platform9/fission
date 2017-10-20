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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/cache"
	"github.com/fission/fission/crd"
)

type (
	createFuncServiceRequest struct {
		funcMeta *metav1.ObjectMeta
		respChan chan *createFuncServiceResponse
	}

	createFuncServiceResponse struct {
		address string
		err     error
	}

	Poolmgr struct {
		gpm           *GenericPoolManager
		functionEnv   *cache.Cache // map[string]crd.Environment
		fsCache       *functionServiceCache
		fissionClient *crd.FissionClient

		fsCreateChannels map[string]*sync.WaitGroup // xxx no channels here, rename this
		requestChan      chan *createFuncServiceRequest
	}
)

func MakePoolmgr(gpm *GenericPoolManager, fissionClient *crd.FissionClient, fissionNs string, fsCache *functionServiceCache) *Poolmgr {
	poolMgr := &Poolmgr{
		gpm:              gpm,
		functionEnv:      cache.MakeCache(10*time.Second, 0),
		fsCache:          fsCache,
		fissionClient:    fissionClient,
		fsCreateChannels: make(map[string]*sync.WaitGroup),
		requestChan:      make(chan *createFuncServiceRequest),
	}
	go poolMgr.serveCreateFuncServices()
	return poolMgr
}

// All non-cached function service requests go through this goroutine
// serially. It parallelizes requests for different functions, and
// ensures that for a given function, only one request causes a pod to
// get specialized. In other words, it ensures that when there's an
// ongoing request for a certain function, all other requests wait for
// that request to complete.
func (poolMgr *Poolmgr) serveCreateFuncServices() {
	for {
		req := <-poolMgr.requestChan
		m := req.funcMeta

		// Cache miss -- is this first one to request the func?
		wg, found := poolMgr.fsCreateChannels[crd.CacheKey(m)]
		if !found {
			// create a waitgroup for other requests for
			// the same function to wait on
			wg := &sync.WaitGroup{}
			wg.Add(1)
			poolMgr.fsCreateChannels[crd.CacheKey(m)] = wg

			// launch a goroutine for each request, to parallelize
			// the specialization of different functions
			go func() {
				address, err := poolMgr.createServiceForFunction(m)
				req.respChan <- &createFuncServiceResponse{
					address: address,
					err:     err,
				}
				delete(poolMgr.fsCreateChannels, crd.CacheKey(m))
				wg.Done()
			}()
		} else {
			// There's an existing request for this function, wait for it to finish
			go func() {
				log.Printf("Waiting for concurrent request for the same function: %v", m)
				wg.Wait()

				// get the function service from the cache
				fsvc, err := poolMgr.fsCache.GetByFunction(m)
				address := ""
				if err == nil {
					address = fsvc.address
				}
				req.respChan <- &createFuncServiceResponse{
					address: address,
					err:     err,
				}
			}()
		}
	}
}

func (poolMgr *Poolmgr) getServiceForFunctionApi(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", 500)
		return
	}

	// get function metadata
	m := metav1.ObjectMeta{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		http.Error(w, "Failed to parse request", 400)
		return
	}

	serviceName, err := poolMgr.getServiceForFunction(&m)
	if err != nil {
		code, msg := fission.GetHTTPError(err)
		log.Printf("Error: %v: %v", code, msg)
		http.Error(w, msg, code)
		return
	}

	w.Write([]byte(serviceName))
}

func (poolMgr *Poolmgr) getFunctionEnv(m *metav1.ObjectMeta) (*crd.Environment, error) {
	var env *crd.Environment

	// Cached ?
	result, err := poolMgr.functionEnv.Get(crd.CacheKey(m))
	if err == nil {
		env = result.(*crd.Environment)
		return env, nil
	}

	// Cache miss -- get func from controller
	f, err := poolMgr.fissionClient.Functions(m.Namespace).Get(m.Name)
	if err != nil {
		return nil, err
	}

	// Get env from metadata
	log.Printf("[%v] getting env", m)
	env, err = poolMgr.fissionClient.Environments(f.Spec.Environment.Namespace).Get(f.Spec.Environment.Name)
	if err != nil {
		return nil, err
	}

	// cache for future lookups
	poolMgr.functionEnv.Set(crd.CacheKey(m), env)

	return env, nil
}

func (poolMgr *Poolmgr) getServiceForFunction(m *metav1.ObjectMeta) (string, error) {
	// Check function -> svc cache
	log.Printf("[%v] Checking for cached function service", m.Name)
	fsvc, err := poolMgr.fsCache.GetByFunction(m)
	if err == nil {
		// Cached, return svc address
		return fsvc.address, nil
	}

	respChan := make(chan *createFuncServiceResponse)
	poolMgr.requestChan <- &createFuncServiceRequest{
		funcMeta: m,
		respChan: respChan,
	}
	resp := <-respChan
	return resp.address, resp.err
}

func (poolMgr *Poolmgr) createServiceForFunction(m *metav1.ObjectMeta) (string, error) {
	// None exists, so create a new funcSvc:
	log.Printf("[%v] No cached function service found, creating one", m.Name)

	// from Func -> get Env
	log.Printf("[%v] getting environment for function", m.Name)
	env, err := poolMgr.getFunctionEnv(m)
	if err != nil {
		return "", err
	}

	// from Env -> get GenericPool
	log.Printf("[%v] getting generic pool for env", m.Name)
	pool, err := poolMgr.gpm.GetPool(env)
	if err != nil {
		return "", err
	}

	// from GenericPool -> get one function container
	// (this also adds to the cache)
	log.Printf("[%v] getting function service from pool", m.Name)
	fsvc, err := pool.GetFuncSvc(m)
	if err != nil {
		return "", err
	}
	return fsvc.address, nil
}

// find funcSvc and update its atime
func (poolMgr *Poolmgr) tapService(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", 500)
		return
	}
	svcName := string(body)
	svcHost := strings.TrimPrefix(svcName, "http://")

	err = poolMgr.fsCache.TouchByAddress(svcHost)
	if err != nil {
		log.Printf("funcSvc tap error: %v", err)
		http.Error(w, "Not found", 404)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (poolMgr *Poolmgr) Serve(port int) {
	r := mux.NewRouter()
	r.HandleFunc("/v2/getServiceForFunction", poolMgr.getServiceForFunctionApi).Methods("POST")
	r.HandleFunc("/v2/tapService", poolMgr.tapService).Methods("POST")
	address := fmt.Sprintf(":%v", port)
	log.Printf("starting poolmgr at port %v", port)
	log.Fatal(http.ListenAndServe(address, handlers.LoggingHandler(os.Stdout, r)))
}
