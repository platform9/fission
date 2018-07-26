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

	crdClient *rest.RESTClient

	recorders     []crd.Recorder
	recStore      k8sCache.Store
	recController k8sCache.Controller
}

func MakeRecorderSet(parent *HTTPTriggerSet, crdClient *rest.RESTClient) (*RecorderSet, k8sCache.Store) {
	var rStore k8sCache.Store
	recorderSet := &RecorderSet{
		parent:              parent,
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

	rs.parent.functionRecorderMap.assign(function, r)

	if needTrack {
		trackFunction[function] = true
	}

	if needTrack {
		for _, t := range rs.parent.triggerStore.List() {
			trigger := *t.(*crd.HTTPTrigger)
			if trackFunction[trigger.Spec.FunctionReference.Name] {
				rs.parent.triggerRecorderMap.assign(trigger.Metadata.Name, r)
			}
		}
	} else {
		for _, trigger := range triggers {
			rs.parent.triggerRecorderMap.assign(trigger, r)
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
	err := rs.parent.functionRecorderMap.remove(function)
	if err != nil {
		log.Error("Unable to disable recorder")
	}

	// Account for explicitly added triggers
	if len(triggers) != 0 {
		for _, trigger := range triggers {
			//delete(rs.triggerRecorderMap, trigger)
			err := rs.parent.triggerRecorderMap.remove(trigger)
			if err != nil {
				log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
			}
		}
	}

	// Account for implicitly added triggers
	for _, t := range rs.parent.triggerStore.List() {
		trigger := *t.(*crd.HTTPTrigger)
		if trigger.Spec.FunctionReference.Name == function {
			err := rs.parent.triggerRecorderMap.remove(trigger.Metadata.Name)
			if err != nil {
				log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
			}
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
	err := rs.parent.triggerRecorderMap.remove(trigger.Metadata.Name)
	if err != nil {
		log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
	}
}

func (rs *RecorderSet) FunctionDeleted(function *crd.Function) {
	err := rs.parent.functionRecorderMap.remove(function.Metadata.Name)
	if err != nil {
		log.Error("Failed to remove function from functionRecorderMap: ", err)
	}
}
