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
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
)

type HTTPTriggerSet struct {
	*functionServiceMap
	*mutableRouter

	fissionClient     *crd.FissionClient
	executor          *executorClient.Client
	resolver          *functionReferenceResolver
	crdClient         *rest.RESTClient
	triggers          []crd.HTTPTrigger
	triggerStore      k8sCache.Store
	triggerController k8sCache.Controller
	functions         []crd.Function
	funcStore         k8sCache.Store
	funcController    k8sCache.Controller
}

func makeHTTPTriggerSet(fmap *functionServiceMap, fissionClient *crd.FissionClient,
	executor *executorClient.Client, crdClient *rest.RESTClient) (*HTTPTriggerSet, k8sCache.Store, k8sCache.Store) {
	httpTriggerSet := &HTTPTriggerSet{
		functionServiceMap: fmap,
		triggers:           []crd.HTTPTrigger{},
		fissionClient:      fissionClient,
		executor:           executor,
		crdClient:          crdClient,
	}
	var tStore, fnStore k8sCache.Store
	var tController, fnController k8sCache.Controller
	if httpTriggerSet.crdClient != nil {
		tStore, tController = httpTriggerSet.initTriggerController()
		httpTriggerSet.triggerStore = tStore
		httpTriggerSet.triggerController = tController
		fnStore, fnController = httpTriggerSet.initFunctionController()
		httpTriggerSet.funcStore = fnStore
		httpTriggerSet.funcController = fnController
	}
	return httpTriggerSet, tStore, fnStore
}

func (ts *HTTPTriggerSet) subscribeRouter(ctx context.Context, mr *mutableRouter, resolver *functionReferenceResolver) {
	ts.resolver = resolver
	ts.mutableRouter = mr
	mr.updateRouter(ts.getRouter())

	if ts.fissionClient == nil {
		// Used in tests only.
		log.Printf("Skipping continuous trigger updates")
		return
	}
	go ts.runWatcher(ctx, ts.funcController)
	go ts.runWatcher(ctx, ts.triggerController)
}

func defaultHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func routerHealthHandler(w http.ResponseWriter, r *http.Request) {
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
			fmap:        ts.functionServiceMap,
			function:    rr.functionMetadata,
			executor:    ts.executor,
			requestChan: make(chan *chanRequest),
		}

		ht := muxRouter.HandleFunc(trigger.Spec.RelativeURL, fh.handler)
		ht.Methods(trigger.Spec.Method)
		if trigger.Spec.Host != "" {
			ht.Host(trigger.Spec.Host)
		}
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
			executor: ts.executor,
		}
		muxRouter.HandleFunc(fission.UrlForFunction(function.Metadata.Name), fh.handler)
	}

	// Healthz endpoint for the router.
	muxRouter.HandleFunc("/router-healthz", routerHealthHandler).Methods("GET")

	return muxRouter
}

func (ts *HTTPTriggerSet) updateTriggerStatusFailed(ht *crd.HTTPTrigger, err error) {
	// TODO
}

func (ts *HTTPTriggerSet) initTriggerController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "httptriggers", metav1.NamespaceDefault, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.HTTPTrigger{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				ts.syncTriggers()
			},
		})
	return store, controller
}

func (ts *HTTPTriggerSet) initFunctionController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "functions", metav1.NamespaceDefault, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				fn := newObj.(*crd.Function)
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
			},
		})
	return store, controller
}

func (ts *HTTPTriggerSet) runWatcher(ctx context.Context, controller k8sCache.Controller) {
	go func() {
		controller.Run(ctx.Done())
	}()
}

func (ts *HTTPTriggerSet) syncTriggers() {
	// get triggers
	latestTriggers := ts.triggerStore.List()
	triggers := make([]crd.HTTPTrigger, len(latestTriggers))
	for _, t := range latestTriggers {
		triggers = append(triggers, *t.(*crd.HTTPTrigger))
	}
	ts.triggers = triggers

	// get functions
	latestFunctions := ts.funcStore.List()
	functions := make([]crd.Function, len(latestFunctions))
	for _, f := range latestFunctions {
		functions = append(functions, *f.(*crd.Function))
	}
	ts.functions = functions

	// make a new router and use it
	ts.mutableRouter.updateRouter(ts.getRouter())
}
