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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	poolmgrClient "github.com/fission/fission/poolmgr/client"
	"github.com/fission/fission/tpr"
)

type HTTPTriggerSet struct {
	*functionServiceMap
	*mutableRouter
	fissionClient *tpr.FissionClient
	poolmgr       *poolmgrClient.Client
	resolver      *functionReferenceResolver
	triggers      []tpr.Httptrigger
	triggerStore  k8sCache.Store
	functions     []tpr.Function
	funcStore     k8sCache.Store
	tprClient     *rest.RESTClient
}

func makeHTTPTriggerSet(fmap *functionServiceMap, fissionClient *tpr.FissionClient,
	poolmgr *poolmgrClient.Client, resolver *functionReferenceResolver, tprClient *rest.RESTClient) *HTTPTriggerSet {
	triggers := make([]tpr.Httptrigger, 1)
	return &HTTPTriggerSet{
		functionServiceMap: fmap,
		triggers:           triggers,
		fissionClient:      fissionClient,
		poolmgr:            poolmgr,
		resolver:           resolver,
		tprClient:          tprClient,
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

func (ts *HTTPTriggerSet) updateTriggerStatusFailed(ht *tpr.Httptrigger, err error) {
	// TODO
}

func (ts *HTTPTriggerSet) watchTriggers() {
	watchlist := k8sCache.NewListWatchFromClient(ts.tprClient, "httptriggers", metav1.NamespaceDefault, fields.Everything())
	listWatch := &k8sCache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return watchlist.Watch(options)
		},
	}
	resyncPeriod := 30 * time.Second
	store, controller := k8sCache.NewInformer(listWatch, &tpr.Httptrigger{}, resyncPeriod,
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
	ts.triggerStore = store
	stop := make(chan struct{})
	defer func() {
		stop <- struct{}{}
	}()
	controller.Run(stop)
}

func (ts *HTTPTriggerSet) watchFunctions() {
	watchlist := k8sCache.NewListWatchFromClient(ts.tprClient, "functions", metav1.NamespaceDefault, fields.Everything())
	listWatch := &k8sCache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return watchlist.Watch(options)
		},
	}
	resyncPeriod := 30 * time.Second
	store, controller := k8sCache.NewInformer(listWatch, &tpr.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				fn := newObj.(*tpr.Function)
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
	ts.funcStore = store
	stop := make(chan struct{})
	defer func() {
		stop <- struct{}{}
	}()
	controller.Run(stop)
}

func (ts *HTTPTriggerSet) syncTriggers() {
	log.Printf("Syncing http triggers")

	// get triggers
	latestTriggers := ts.triggerStore.List()
	triggers := make([]tpr.Httptrigger, len(latestTriggers))
	for _, t := range latestTriggers {
		triggers = append(triggers, *t.(*tpr.Httptrigger))
	}
	ts.triggers = triggers

	// get functions
	latestFunctions := ts.funcStore.List()
	functions := make([]tpr.Function, len(latestFunctions))
	for _, f := range latestFunctions {
		functions = append(functions, *f.(*tpr.Function))
	}
	ts.functions = functions

	// make a new router and use it
	ts.mutableRouter.updateRouter(ts.getRouter())
}
