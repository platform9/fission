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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
)

type HTTPTriggerSet struct {
	*functionServiceMap
	*mutableRouter

	fissionClient              *crd.FissionClient
	kubeClient                 *kubernetes.Clientset
	executor                   *executorClient.Client
	resolver                   *functionReferenceResolver
	crdClient                  *rest.RESTClient
	triggers                   []crd.HTTPTrigger
	triggerStore               k8sCache.Store
	triggerController          k8sCache.Controller
	functions                  []crd.Function
	funcStore                  k8sCache.Store
	funcController             k8sCache.Controller
	recorderSet                *RecorderSet
	updateRouterRequestChannel chan struct{}
	tsRoundTripperParams       *tsRoundTripperParams
}

func makeHTTPTriggerSet(fmap *functionServiceMap, frmap *functionRecorderMap, trmap *triggerRecorderMap, fissionClient *crd.FissionClient,
	kubeClient *kubernetes.Clientset, executor *executorClient.Client, crdClient *rest.RESTClient, params *tsRoundTripperParams) (*HTTPTriggerSet, k8sCache.Store, k8sCache.Store) {
	httpTriggerSet := &HTTPTriggerSet{
		functionServiceMap:         fmap,
		triggers:                   []crd.HTTPTrigger{},
		fissionClient:              fissionClient,
		kubeClient:                 kubeClient,
		executor:                   executor,
		crdClient:                  crdClient,
		updateRouterRequestChannel: make(chan struct{}),
		tsRoundTripperParams:       params,
	}
	var tStore, fnStore, rStore k8sCache.Store
	var tController, fnController k8sCache.Controller
	var recorderSet *RecorderSet
	if httpTriggerSet.crdClient != nil {
		tStore, tController = httpTriggerSet.initTriggerController()
		httpTriggerSet.triggerStore = tStore
		httpTriggerSet.triggerController = tController
		fnStore, fnController = httpTriggerSet.initFunctionController()
		httpTriggerSet.funcStore = fnStore
		httpTriggerSet.funcController = fnController
	}
	recorderSet = MakeRecorderSet(httpTriggerSet, crdClient, rStore, frmap, trmap)
	httpTriggerSet.recorderSet = recorderSet
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
	go ts.updateRouter()
	go ts.runWatcher(ctx, ts.funcController)
	go ts.runWatcher(ctx, ts.triggerController)
	if ts.recorderSet.recController != nil {
		go ts.runWatcher(ctx, ts.recorderSet.recController)
	} else {
		log.Fatal("Failed to run recorder Controller")
	}
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
	for i := range ts.triggers {
		trigger := ts.triggers[i]

		// resolve function reference
		rr, err := ts.resolver.resolve(trigger)
		if err != nil {
			//log.Printf("Error resolving trigger: %s, err : %v", trigger.Metadata.Name, err)

			// Unresolvable function reference. Report the error via
			// the trigger's status.
			go ts.updateTriggerStatusFailed(&trigger, err)

			// Ignore this route and let it 404.
			continue
		}

		var recorderName string
		recorder, err := ts.recorderSet.triggerRecorderMap.lookup(trigger.Metadata.Name)
		if err == nil && recorder != nil {
			recorderName = recorder.Spec.Name
		}

		//log.Printf("The trigger %v should be recorded: %v", trigger.Metadata.Name, doRecord)

		if rr.resolveResultType != resolveResultSingleFunction && rr.resolveResultType != resolveResultMultipleFunctions {
			// not implemented yet
			log.Panicf("resolve result type not implemented (%v)", rr.resolveResultType)
		}

		fh := &functionHandler{
			fmap:                     ts.functionServiceMap,
			frmap:                    ts.recorderSet.functionRecorderMap,
			trmap:                    ts.recorderSet.triggerRecorderMap,
			executor:                 ts.executor,
			httpTrigger:              &trigger,
			functionMetadataMap:      rr.functionMetadataMap,
			fnWeightDistributionList: rr.functionWtDistributionList,
			tsRoundTripperParams:     ts.tsRoundTripperParams,
			recorderName:             recorderName,
		}

		//log.Printf("Dumping fh object : %+v", fh)

		if rr.resolveResultType == resolveResultSingleFunction {
			for _, metadata := range fh.functionMetadataMap {
				fh.function = metadata
			}
		}

		//log.Printf("Setting up url %s handler %+v", trigger.Spec.RelativeURL, *fh)

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

		var recorderName string
		recorder, err := ts.recorderSet.functionRecorderMap.lookup(m.Name)
		if err == nil && recorder != nil {
			recorderName = recorder.Spec.Name
		}

		fh := &functionHandler{
			fmap:                 ts.functionServiceMap,
			frmap:                ts.recorderSet.functionRecorderMap,
			trmap:                ts.recorderSet.triggerRecorderMap,
			function:             &m,
			executor:             ts.executor,
			tsRoundTripperParams: ts.tsRoundTripperParams,
			recorderName:         recorderName,
		}
		muxRouter.HandleFunc(fission.UrlForFunction(function.Metadata.Name, function.Metadata.Namespace), fh.handler)
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
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "httptriggers", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.HTTPTrigger{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				trigger := obj.(*crd.HTTPTrigger)
				go createIngress(trigger, ts.kubeClient)
				ts.syncTriggers()
				// Check if this trigger's function needs to be recorded
				fnRef := trigger.Spec.FunctionReference.Name
				recorder, err := ts.recorderSet.functionRecorderMap.lookup(fnRef)
				if err == nil && recorder != nil {
					if len(recorder.Spec.Triggers) == 0 {
						ts.recorderSet.triggerRecorderMap.assign(trigger.Metadata.Name, recorder)
					}
				} else {
					log.Print("Unable to lookup function in functionRecorderMap")
				}
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
				trigger := obj.(*crd.HTTPTrigger)
				go deleteIngress(trigger, ts.kubeClient)
				go ts.recorderSet.DeleteTriggerFromRecorderMap(trigger)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldTrigger := oldObj.(*crd.HTTPTrigger)
				newTrigger := newObj.(*crd.HTTPTrigger)

				if oldTrigger.Metadata.ResourceVersion == newTrigger.Metadata.ResourceVersion {
					return
				}

				go updateIngress(oldTrigger, newTrigger, ts.kubeClient)
				ts.syncTriggers()
			},
		})
	return store, controller
}

func (ts *HTTPTriggerSet) initFunctionController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "functions", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				function := obj.(*crd.Function)
				ts.syncTriggers()
				go ts.recorderSet.DeleteFunctionFromRecorderMap(function)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldFn := oldObj.(*crd.Function)
				fn := newObj.(*crd.Function)

				if oldFn.Metadata.ResourceVersion == fn.Metadata.ResourceVersion {
					return
				}

				// update resolver function reference cache
				for key, rr := range ts.resolver.copy() {
					if key.namespace == fn.Metadata.Namespace &&
						rr.functionMetadataMap[fn.Metadata.Name] != nil &&
						rr.functionMetadataMap[fn.Metadata.Name].ResourceVersion != fn.Metadata.ResourceVersion {
						// invalidate resolver cache
						log.Printf("Invalidating resolver cache")
						err := ts.resolver.delete(key.namespace, key.triggerName, key.triggerResourceVersion)
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

func (ts *HTTPTriggerSet) initRecorderController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "recorders", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Recorder{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				recorder := obj.(*crd.Recorder)
				ts.recorderSet.newRecorder(recorder)
			},
			DeleteFunc: func(obj interface{}) {
				recorder := obj.(*crd.Recorder)
				ts.recorderSet.disableRecorder(recorder)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldRecorder := oldObj.(*crd.Recorder)
				newRecorder := newObj.(*crd.Recorder)
				ts.recorderSet.updateRecorder(oldRecorder, newRecorder)
			},
		},
	)
	return store, controller
}

func (ts *HTTPTriggerSet) runWatcher(ctx context.Context, controller k8sCache.Controller) {
	go func() {
		controller.Run(ctx.Done())
	}()
}

func (ts *HTTPTriggerSet) syncTriggers() {
	ts.updateRouterRequestChannel <- struct{}{}
}

func (ts *HTTPTriggerSet) updateRouter() {
	for range ts.updateRouterRequestChannel {
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
}
