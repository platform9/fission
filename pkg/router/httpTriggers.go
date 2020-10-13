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
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	"github.com/fission/fission/pkg/crd"
	executorClient "github.com/fission/fission/pkg/executor/client"
	"github.com/fission/fission/pkg/throttler"
	"github.com/fission/fission/pkg/utils"
)

// HTTPTriggerSet represents an HTTP trigger set
type HTTPTriggerSet struct {
	*functionServiceMap
	*mutableRouter

	logger                     *zap.Logger
	fissionClient              *crd.FissionClient
	kubeClient                 *kubernetes.Clientset
	executor                   *executorClient.Client
	resolver                   *functionReferenceResolver
	crdClient                  rest.Interface
	triggers                   []fv1.HTTPTrigger
	triggerStore               k8sCache.Store
	triggerController          k8sCache.Controller
	functions                  []fv1.Function
	funcStore                  k8sCache.Store
	funcController             k8sCache.Controller
	updateRouterRequestChannel chan struct{}
	tsRoundTripperParams       *tsRoundTripperParams
	isDebugEnv                 bool
	svcAddrUpdateThrottler     *throttler.Throttler
	unTapServiceTimeout        time.Duration
}

func makeHTTPTriggerSet(logger *zap.Logger, fmap *functionServiceMap, fissionClient *crd.FissionClient,
	kubeClient *kubernetes.Clientset, executor *executorClient.Client, crdClient rest.Interface, params *tsRoundTripperParams, isDebugEnv bool, unTapServiceTimeout time.Duration, actionThrottler *throttler.Throttler) (*HTTPTriggerSet, k8sCache.Store, k8sCache.Store) {

	httpTriggerSet := &HTTPTriggerSet{
		logger:                     logger.Named("http_trigger_set"),
		functionServiceMap:         fmap,
		triggers:                   []fv1.HTTPTrigger{},
		fissionClient:              fissionClient,
		kubeClient:                 kubeClient,
		executor:                   executor,
		crdClient:                  crdClient,
		updateRouterRequestChannel: make(chan struct{}, 10), // use buffer channel
		tsRoundTripperParams:       params,
		isDebugEnv:                 isDebugEnv,
		svcAddrUpdateThrottler:     actionThrottler,
		unTapServiceTimeout:        unTapServiceTimeout,
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

	if ts.fissionClient == nil {
		// Used in tests only.
		mr.updateRouter(ts.getRouter(nil))
		ts.logger.Info("skipping continuous trigger updates")
		return
	}
	go ts.updateRouter()
	go ts.syncTriggers()
	go ts.runWatcher(ctx, ts.funcController)
	go ts.runWatcher(ctx, ts.triggerController)
}

func defaultHomeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func routerHealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (ts *HTTPTriggerSet) getRouter(fnTimeoutMap map[types.UID]int) *mux.Router {
	muxRouter := mux.NewRouter()

	// HTTP triggers setup by the user
	homeHandled := false
	for i := range ts.triggers {
		trigger := ts.triggers[i]

		// resolve function reference
		rr, err := ts.resolver.resolve(trigger)
		if err != nil {
			// Unresolvable function reference. Report the error via
			// the trigger's status.
			go ts.updateTriggerStatusFailed(&trigger, err)

			// Ignore this route and let it 404.
			continue
		}

		if rr.resolveResultType != resolveResultSingleFunction && rr.resolveResultType != resolveResultMultipleFunctions {
			// not implemented yet
			ts.logger.Panic("resolve result type not implemented", zap.Any("type", rr.resolveResultType))
		}

		fh := &functionHandler{
			logger:                   ts.logger.Named(trigger.ObjectMeta.Name),
			fmap:                     ts.functionServiceMap,
			executor:                 ts.executor,
			httpTrigger:              &trigger,
			functionMap:              rr.functionMap,
			fnWeightDistributionList: rr.functionWtDistributionList,
			tsRoundTripperParams:     ts.tsRoundTripperParams,
			isDebugEnv:               ts.isDebugEnv,
			svcAddrUpdateThrottler:   ts.svcAddrUpdateThrottler,
			functionTimeoutMap:       fnTimeoutMap,
			unTapServiceTimeout:      ts.unTapServiceTimeout,
		}

		// The functionHandler for HTTP trigger with fn reference type "FunctionReferenceTypeFunctionName",
		// it's function metadata is set here.

		// The functionHandler For HTTP trigger with fn reference type "FunctionReferenceTypeFunctionWeights",
		// it's function metadata is decided dynamically before proxying the request in order to support canary
		// deployment. For more details, please check "handler" function of functionHandler.

		if rr.resolveResultType == resolveResultSingleFunction {
			for _, fn := range fh.functionMap {
				fh.function = fn
			}
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
	for i := range ts.functions {
		fn := ts.functions[i]
		fh := &functionHandler{
			logger:                 ts.logger.Named(fn.ObjectMeta.Name),
			fmap:                   ts.functionServiceMap,
			function:               &fn,
			executor:               ts.executor,
			tsRoundTripperParams:   ts.tsRoundTripperParams,
			isDebugEnv:             ts.isDebugEnv,
			svcAddrUpdateThrottler: ts.svcAddrUpdateThrottler,
			functionTimeoutMap:     fnTimeoutMap,
			unTapServiceTimeout:    ts.unTapServiceTimeout,
		}
		muxRouter.HandleFunc(utils.UrlForFunction(fn.ObjectMeta.Name, fn.ObjectMeta.Namespace), fh.handler)
	}

	// Healthz endpoint for the router.
	muxRouter.HandleFunc("/router-healthz", routerHealthHandler).Methods("GET")

	return muxRouter
}

func (ts *HTTPTriggerSet) updateTriggerStatusFailed(ht *fv1.HTTPTrigger, err error) {
	// TODO
}

func (ts *HTTPTriggerSet) initTriggerController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "httptriggers", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &fv1.HTTPTrigger{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				trigger := obj.(*fv1.HTTPTrigger)
				go createIngress(ts.logger, trigger, ts.kubeClient)
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
				trigger := obj.(*fv1.HTTPTrigger)
				go deleteIngress(ts.logger, trigger, ts.kubeClient)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldTrigger := oldObj.(*fv1.HTTPTrigger)
				newTrigger := newObj.(*fv1.HTTPTrigger)

				if oldTrigger.ObjectMeta.ResourceVersion == newTrigger.ObjectMeta.ResourceVersion {
					return
				}

				go updateIngress(ts.logger, oldTrigger, newTrigger, ts.kubeClient)
				ts.syncTriggers()
			},
		})
	return store, controller
}

func (ts *HTTPTriggerSet) initFunctionController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "functions", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &fv1.Function{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldFn := oldObj.(*fv1.Function)
				fn := newObj.(*fv1.Function)

				if oldFn.ObjectMeta.ResourceVersion == fn.ObjectMeta.ResourceVersion {
					return
				}

				// update resolver function reference cache
				for key, rr := range ts.resolver.copy() {
					if key.namespace == fn.ObjectMeta.Namespace &&
						rr.functionMap[fn.ObjectMeta.Name] != nil &&
						rr.functionMap[fn.ObjectMeta.Name].ObjectMeta.ResourceVersion != fn.ObjectMeta.ResourceVersion {
						// invalidate resolver cache
						ts.logger.Debug("invalidating resolver cache")
						err := ts.resolver.delete(key.namespace, key.triggerName, key.triggerResourceVersion)
						if err != nil {
							ts.logger.Error("error deleting functionReferenceResolver cache", zap.Error(err))
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
	ts.updateRouterRequestChannel <- struct{}{}
}

func (ts *HTTPTriggerSet) updateRouter() {
	for range ts.updateRouterRequestChannel {
		// get triggers
		latestTriggers := ts.triggerStore.List()
		triggers := make([]fv1.HTTPTrigger, len(latestTriggers))
		for _, t := range latestTriggers {
			triggers = append(triggers, *t.(*fv1.HTTPTrigger))
		}
		ts.triggers = triggers

		// get functions
		latestFunctions := ts.funcStore.List()
		functionTimeout := make(map[types.UID]int, len(latestFunctions))
		functions := make([]fv1.Function, len(latestFunctions))
		for _, f := range latestFunctions {
			fn := *f.(*fv1.Function)
			functionTimeout[fn.ObjectMeta.UID] = fn.Spec.FunctionTimeout
			functions = append(functions, *f.(*fv1.Function))
		}
		ts.functions = functions

		// make a new router and use it
		ts.mutableRouter.updateRouter(ts.getRouter(functionTimeout))
	}
}
