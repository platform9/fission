package router

import (
	"k8s.io/client-go/kubernetes"
	k8sCache "k8s.io/client-go/tools/cache"

	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
	"k8s.io/client-go/rest"
	"log"
	"github.com/fission/fission/environments/go/context"
	"net/http"
	"github.com/fission/fission"
	"github.com/gorilla/mux"
	"time"
)

type RecorderSet struct {
	*functionRecorderMap
	*mutableRouter				// o_O

	fissionClient    *crd.FissionClient
	kubeClient       *kubernetes.Clientset
	executor         *executorClient.Client
	resolver         *functionReferenceResolver
	crdClient        *rest.RESTClient
	recorders        []crd.Recorder
	recorderStore    k8sCache.Store				// TODO: What is this for?
	recorderController    k8sCache.Controller
}

// TODO: How many stores should we use?
func makeRecorderSet(frmap *functionRecorderMap, fissionClient *crd.FissionClient,
	kubeClient *kubernetes.Clientset, executor *executorClient.Client, crdClient *rest.RESTClient) (*RecorderSet, k8sCache.Store) {
	recorderSet := &RecorderSet{
		functionRecorderMap: frmap,
		recorders: []crd.Recorder{},
		fissionClient: fissionClient,
		kubeClient: kubeClient,
		executor: executor,
		crdClient: crdClient,
	}
	var rStore k8sCache.Store
	var rController k8sCache.Controller
	if recorderSet.crdClient != nil {
		rStore, rController = recorderSet.initRecorderController()
		recorderSet.recorderStore = rStore
		recorderSet.recorderController = rController
	}
	return recorderSet, rStore
}
/*
func makeHTTPTriggerSet(fmap *functionServiceMap, fissionClient *crd.FissionClient,
	kubeClient *kubernetes.Clientset, executor *executorClient.Client, crdClient *rest.RESTClient) (*HTTPTriggerSet, k8sCache.Store, k8sCache.Store) {
	httpTriggerSet := &HTTPTriggerSet{
		functionServiceMap: fmap,
		triggers:           []crd.HTTPTrigger{},
		fissionClient:      fissionClient,
		kubeClient:         kubeClient,
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
*/

// TODO: Third argument?
func (rs *RecorderSet) subscribeRouter(ctx context.Context, mr *mutableRouter, resolver *functionReferenceResolver) {
	rs.resolver = resolver
	rs.mutableRouter = mr
	mr.updateRouter(rs.getRouter())

	if rs.fissionClient == nil {
		// ???
		return
	}
	go rs.runWatcher(ctx, rs.recorderController)
}

/*
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
*/

func (rs *RecorderSet) getRouter() *mux.Router {
	muxRouter := mux.NewRouter()

	for _, recorder := range rs.recorders {

		// Resolve function reference if recorder is tied to function(s)
		if len(recorder.Spec.Functions) != 0 {
			for _, functionRef := range recorder.Spec.Functions {
				rr, err := rs.resolver.resolve(recorder.Metadata.Namespace, &functionRef)
				if err != nil {
					// Unresolvable function reference. Report the error via the recorder's status.
					go rs.updateRecorderStatusFailed(&recorder, err)

					continue
				}
				if rr.resolveResultType != resolveResultSingleFunction {
					// not implemented yet
					log.Panicf("resolve result type not implemented (%v)", rr.resolveResultType)
				}
			}
		}

		// Resolve trigger reference is recorder is tied to trigger(s)

		if len(recorder.Spec.Triggers) != 0 {

		}

		// TODO function handler?

	}
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
			httpTrigger: &trigger,
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
		muxRouter.HandleFunc(fission.UrlForFunction(function.Metadata.Name, function.Metadata.Namespace), fh.handler)
	}

	// Healthz endpoint for the router.
	muxRouter.HandleFunc("/router-healthz", routerHealthHandler).Methods("GET")

	return muxRouter
}

func (rs *RecorderSet) initRecorderController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(rs.crdClient, "recorders", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Recorder{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

			},
			DeleteFunc: func(obj interface{}) {

			},
			UpdateFunc: func(oldObj, newObj interface{}) {

			},
		},
	)
	return store, controller
}

/*
func (ts *HTTPTriggerSet)  initTriggerController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second
	listWatch := k8sCache.NewListWatchFromClient(ts.crdClient, "httptriggers", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.HTTPTrigger{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				trigger := obj.(*crd.HTTPTrigger)
				go createIngress(trigger, ts.kubeClient)
				ts.syncTriggers()
			},
			DeleteFunc: func(obj interface{}) {
				ts.syncTriggers()
				trigger := obj.(*crd.HTTPTrigger)
				go deleteIngress(trigger, ts.kubeClient)
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				oldTrigger := oldObj.(*crd.HTTPTrigger)
				newTrigger := newObj.(*crd.HTTPTrigger)
				go updateIngress(oldTrigger, newTrigger, ts.kubeClient)
				ts.syncTriggers()
			},
		})
	return store, controller
}
*/

func (rs *RecorderSet) runWatcher(ctx context.Context, controller k8sCache.Controller) {
	go func() {
		controller.Run(ctx.Done())
	}()
}

/*
func (ts *HTTPTriggerSet) runWatcher(ctx context.Context, controller k8sCache.Controller) {
	go func() {
		controller.Run(ctx.Done())
	}()
}
*/
