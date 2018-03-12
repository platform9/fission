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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
)

func (executor *Executor) getServiceForFunctionApi(w http.ResponseWriter, r *http.Request) {
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

	serviceName, err := executor.getServiceForFunction(&m)
	if err != nil {
		code, msg := fission.GetHTTPError(err)
		log.Printf("Error: %v: %v", code, msg)
		http.Error(w, msg, code)
		return
	}

	w.Write([]byte(serviceName))
}

func (executor *Executor) getServiceForFunction(m *metav1.ObjectMeta) (string, error) {
	// Check function -> svc cache
	log.Printf("[%v] Checking for cached function service", m.Name)
	fsvc, err := executor.fsCache.GetByFunction(m)
	if err == nil {
		// Cached, return svc address
		return fsvc.Address, nil
	}

	respChan := make(chan *createFuncServiceResponse)
	executor.requestChan <- &createFuncServiceRequest{
		funcMeta: m,
		respChan: respChan,
	}
	resp := <-respChan
	if resp.err != nil {
		return "", resp.err
	}
	return resp.funcSvc.Address, resp.err
}

// find funcSvc and update its atime
func (executor *Executor) tapService(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", 500)
		return
	}
	svcName := string(body)
	svcHost := strings.TrimPrefix(svcName, "http://")

	err = executor.fsCache.TouchByAddress(svcHost)
	if err != nil {
		log.Printf("funcSvc tap error: %v", err)
		http.Error(w, "Not found", 404)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// invalidateCacheEntryForFunction receives the function metadata whose podIP:port has become stale and removes the
// function entry from cache. It also makes a create function service request and finally returns the podIP:port of
// the newly specialized pod.
// TODO : Does it look sneaky to do both, invalidation and creation, in one single call?
//	  The other way this can be implemented is for this function to only invalidate cache entry and send a response.
//	  Now, when the router receives the response, it can send another http request subsequently to executor to create a new service for function.
//	  But in the interest of time, i feel there's no need of an extra http request and response between them. more so because this is in the path of cold start.
//	  What is your opinion?
func (executor *Executor) invalidateCacheEntryForFunction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", 500)
		return
	}

	// get function metadata
	var invalidateCacheRequest fission.CacheInvalidationRequest
	err = json.Unmarshal(body, &invalidateCacheRequest)
	if err != nil {
		http.Error(w, "Failed to parse request", 400)
		return
	}

	respChan := make(chan *CacheInvalidationResponse)
	executor.invalidateCacheRequestChan <- &invalidateCacheChanRequest{
		request:  invalidateCacheRequest,
		response: respChan,
	}
	response := <-respChan
	if response.err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		respChan := make(chan *createFuncServiceResponse)
		executor.requestChan <- &createFuncServiceRequest{
			funcMeta: invalidateCacheRequest.FunctionMetadata,
			respChan: respChan,
		}
		resp := <-respChan
		if resp.err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(resp.funcSvc.Address))
	}
}

func (executor *Executor) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (executor *Executor) Serve(port int) {
	r := mux.NewRouter()
	r.HandleFunc("/v2/getServiceForFunction", executor.getServiceForFunctionApi).Methods("POST")
	r.HandleFunc("/v2/tapService", executor.tapService).Methods("POST")
	r.HandleFunc("/v2/invalidateCacheEntryForFunction", executor.invalidateCacheEntryForFunction).Methods("POST")
	r.HandleFunc("/healthz", executor.healthHandler).Methods("GET")
	address := fmt.Sprintf(":%v", port)
	log.Printf("starting executor at port %v", port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executor.ndm.Run(ctx)
	r.Use(fission.LoggingMiddleware)
	log.Fatal(http.ListenAndServe(address, r))
}
