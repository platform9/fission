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
	"net/http"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	poolmgrClient "github.com/fission/fission/poolmgr/client"
)

type HTTPTriggerSet struct {
	*functionServiceMap
	*mutableRouter
	fissionClient *crd.FissionClient
	poolmgr       *poolmgrClient.Client
	resolver      *functionReferenceResolver
	triggers      []crd.Httptrigger
	functions     []crd.Function
}

func makeHTTPTriggerSet(fmap *functionServiceMap, fissionClient *crd.FissionClient, poolmgr *poolmgrClient.Client, resolver *functionReferenceResolver) *HTTPTriggerSet {
	triggers := make([]crd.Httptrigger, 1)
	return &HTTPTriggerSet{
		functionServiceMap: fmap,
		triggers:           triggers,
		fissionClient:      fissionClient,
		poolmgr:            poolmgr,
		resolver:           resolver,
	}
}

func (ts *HTTPTriggerSet) subscribeRouter(mr *mutableRouter) {
	ts.mutableRouter = mr
	mr.updateRouter(ts.getRouter())

	if ts.fissionClient == nil {
		// Used in tests only.
		log.Printf("Skipping continuous trigger updates")
		return
	}
	go ts.watchTriggers()
	go ts.watchFunctions()
}

func defaultHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ts *HTTPTriggerSet) getRouter() *mux.Router {
	muxRouter := mux.NewRouter()

	// HTTP triggers setup by the user
	homeHandled := false
	for _, trigger := range ts.triggers {

		// resolve function reference
		rr, err := ts.resolver.resolve(trigger.Metadata.Namespace, &trigger.Spec.FunctionReference)
		if err != nil {
			// Unresolvable function reference. Report the error via
			// the trigger's status.
			go ts.updateTriggerStatusFailed(&trigger, err)

			// Ignore this route and let it 404.
			continue
		}

		if rr.resolveResultType != resolveResultSingleFunction {
			// not implemented yet
			log.Panicf("resolve result type not implemented (%v)", rr.resolveResultType)
		}

		fh := &functionHandler{
			fmap:     ts.functionServiceMap,
			function: rr.functionMetadata,
			poolmgr:  ts.poolmgr,
		}
		muxRouter.HandleFunc(trigger.Spec.RelativeURL, fh.handler).Methods(trigger.Spec.Method)
		if trigger.Spec.RelativeURL == "/" && trigger.Spec.Method == "GET" {
			homeHandled = true
		}
	}
	if !homeHandled {
		//
		// This adds a no-op handler that returns 200-OK to make sure that the
		// "GET /" request succeeds.  This route is used by GKE Ingress (and
		// perhaps other ingress implementations) as a health check, so we don't
		// want it to be a 404 even if the user doesn't have a function mapped to
		// this route.
		//
		muxRouter.HandleFunc("/", defaultHomeHandler).Methods("GET")
	}

	// Internal triggers for each function by name. Non-http
	// triggers route into these.
	for _, function := range ts.functions {
		m := function.Metadata
		fh := &functionHandler{
			fmap:     ts.functionServiceMap,
			function: &m,
			poolmgr:  ts.poolmgr,
		}
		muxRouter.HandleFunc(fission.UrlForFunction(function.Metadata.Name), fh.handler)
	}

	return muxRouter
}

func (ts *HTTPTriggerSet) updateTriggerStatusFailed(ht *crd.Httptrigger, err error) {
	// TODO
}

func (ts *HTTPTriggerSet) watchTriggers() {
	// sync all http triggers
	ts.syncTriggers()

	// Watch controller for updates to triggers and update the router accordingly.
	rv := ""
	for {
		wi, err := ts.fissionClient.Httptriggers(metav1.NamespaceAll).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			log.Fatalf("Failed to watch http trigger list: %v", err)
		}

		for {
			ev, more := <-wi.ResultChan()
			if !more {
				// restart watch from last rv
				break
			}
			if ev.Type == watch.Error {
				// restart watch from the start
				rv = ""
				time.Sleep(time.Second)
				break
			}
			ht := ev.Object.(*crd.Httptrigger)
			rv = ht.Metadata.ResourceVersion
			ts.syncTriggers()
		}
	}
}

func (ts *HTTPTriggerSet) watchFunctions() {
	rv := ""
	for {
		wi, err := ts.fissionClient.Functions(metav1.NamespaceAll).Watch(metav1.ListOptions{
			ResourceVersion: rv,
		})
		if err != nil {
			log.Fatalf("Failed to watch function list: %v", err)
		}

		for {
			ev, more := <-wi.ResultChan()
			if !more {
				// restart watch from last rv
				break
			}
			if ev.Type == watch.Error {
				// restart watch from the start
				rv = ""
				time.Sleep(time.Second)
				break
			}
			fn := ev.Object.(*crd.Function)
			rv = fn.Metadata.ResourceVersion

			// update resolver function reference cache
			for key, rr := range ts.resolver.copy() {
				if key.functionReference.Name == fn.Metadata.Name &&
					rr.functionMetadata.ResourceVersion != fn.Metadata.ResourceVersion {
					err := ts.resolver.delete(key.namespace, &key.functionReference)
					if err != nil {
						log.Printf("Error deleting functionReferenceResolver cache: %v", err)
					}
					break
				}
			}

			ts.syncTriggers()
		}
	}
}

func (ts *HTTPTriggerSet) syncTriggers() {
	log.Printf("Syncing http triggers")

	// get triggers
	triggers, err := ts.fissionClient.Httptriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to get http trigger list: %v", err)
	}
	ts.triggers = triggers.Items

	// get functions
	functions, err := ts.fissionClient.Functions(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to get function list: %v", err)
	}
	ts.functions = functions.Items

	// make a new router and use it
	ts.mutableRouter.updateRouter(ts.getRouter())
}
