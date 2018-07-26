package router

import (
	"github.com/fission/fission/crd"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"
	"time"
)

type RecorderSet struct {
	parent *HTTPTriggerSet

	functionRecorderMap map[string]*crd.Recorder
	triggerRecorderMap  map[string]*crd.Recorder

	crdClient *rest.RESTClient

	recorders     []crd.Recorder
	recStore      k8sCache.Store
	recController k8sCache.Controller
}

func MakeRecorderSet(parent *HTTPTriggerSet, crdClient *rest.RESTClient) (*RecorderSet, k8sCache.Store) {
	var rStore k8sCache.Store
	recorderSet := &RecorderSet{
		parent:              parent,
		functionRecorderMap: make(map[string]*crd.Recorder),
		triggerRecorderMap:  make(map[string]*crd.Recorder),
		crdClient:           crdClient,
		recorders:           []crd.Recorder{},
		recStore:            rStore,
	}
	_, recorderSet.recController = recorderSet.initRecorderController()
	return recorderSet, rStore
}

func (rs *RecorderSet) initRecorderController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 45 * time.Second

	listWatch := k8sCache.NewListWatchFromClient(rs.crdClient, "recorders", metav1.NamespaceAll, fields.Everything())
	store, controller := k8sCache.NewInformer(listWatch, &crd.Recorder{}, resyncPeriod,
		k8sCache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				recorder := obj.(*crd.Recorder)
				rs.newRecorder(recorder)
			},
			DeleteFunc: func(obj interface{}) {
				recorder := obj.(*crd.Recorder)
				rs.disableRecorder(recorder)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				oldRecorder := oldObj.(*crd.Recorder)
				newRecorder := newObj.(*crd.Recorder)
				rs.updateRecorder(oldRecorder, newRecorder)
			},
		},
	)
	return store, controller
}

// All new recorders are by default enabled
func (rs *RecorderSet) newRecorder(r *crd.Recorder) {
	function := r.Spec.Function
	triggers := r.Spec.Triggers

	// If triggers are not explicitly specified during the creation of this recorder,
	// keep track of those associated with the function(s) specified [implicitly added triggers]
	needTrack := len(triggers) == 0

	trackFunction := make(map[string]bool)

	log.Info("Creating/enabling recorder ! Need to track implicit triggers? ", needTrack)

	rs.functionRecorderMap[function] = r
	if needTrack {
		trackFunction[function] = true
	}

	// Account for implicitly added triggers
	if needTrack {
		for _, t := range rs.parent.triggerStore.List() {
			trigger := *t.(*crd.HTTPTrigger)
			if trackFunction[trigger.Spec.FunctionReference.Name] {
				rs.triggerRecorderMap[trigger.Metadata.Name] = r
			}
		}
	} else {
		// Only record for the explicitly added triggers otherwise
		for _, trigger := range triggers {
			rs.triggerRecorderMap[trigger] = r
		}
	}

	// TODO: Should we force a reset here? If this is renabling a disabled recorder, we have to
	// Reset doRecord
	rs.parent.mutableRouter.updateRouter(rs.parent.getRouter())
}

// TODO: Delete or disable?
func (rs *RecorderSet) disableRecorder(r *crd.Recorder) {
	function := r.Spec.Function
	triggers := r.Spec.Triggers

	log.Info("Disabling recorder!")

	// Account for function
	delete(rs.functionRecorderMap, function)

	log.Info("Disable recorder !")

	// Account for explicitly added triggers
	if len(triggers) != 0 {
		for _, trigger := range triggers {
			delete(rs.triggerRecorderMap, trigger)
		}
	}

	// Account for implicitly added triggers
	for _, t := range rs.parent.triggerStore.List() {
		trigger := *t.(*crd.HTTPTrigger)
		if trigger.Spec.FunctionReference.Name == function {
			delete(rs.triggerRecorderMap, trigger.Metadata.Name) // Use function defined for this purpose below?
		}
	}

	// Reset doRecord
	rs.parent.mutableRouter.updateRouter(rs.parent.getRouter())
}

func (rs *RecorderSet) updateRecorder(old *crd.Recorder, newer *crd.Recorder) {
	if newer.Spec.Enabled == true {
		rs.newRecorder(newer) // TODO: Test this
	} else {
		rs.disableRecorder(old)
	}
}

func (rs *RecorderSet) TriggerDeleted(trigger *crd.HTTPTrigger) {
	delete(rs.triggerRecorderMap, trigger.Metadata.Name)
}

func (rs *RecorderSet) FunctionDeleted(function *crd.Function) {
	delete(rs.functionRecorderMap, function.Metadata.Name)
}
