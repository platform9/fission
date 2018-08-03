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
	triggerSet *HTTPTriggerSet

	crdClient *rest.RESTClient

	recStore      k8sCache.Store
	recController k8sCache.Controller

	functionRecorderMap *functionRecorderMap
	triggerRecorderMap  *triggerRecorderMap
}

func MakeRecorderSet(parent *HTTPTriggerSet, crdClient *rest.RESTClient, rStore k8sCache.Store, frmap *functionRecorderMap, trmap *triggerRecorderMap) *RecorderSet {
	recorderSet := &RecorderSet{
		triggerSet:          parent,
		crdClient:           crdClient,
		recStore:            rStore,
		functionRecorderMap: frmap,
		triggerRecorderMap:  trmap,
	}
	_, recorderSet.recController = recorderSet.initRecorderController()
	return recorderSet
}

func (rs *RecorderSet) initRecorderController() (k8sCache.Store, k8sCache.Controller) {
	resyncPeriod := 30 * time.Second

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
	// keep track of those associated with the function specified [implicitly added triggers]
	needTrack := len(triggers) == 0

	rs.functionRecorderMap.assign(function, r)

	if needTrack {
		for _, t := range rs.triggerSet.triggerStore.List() {
			trigger := *t.(*crd.HTTPTrigger)
			if trigger.Spec.FunctionReference.Name == function {
				rs.triggerRecorderMap.assign(trigger.Metadata.Name, r)
			}
		}
	} else {
		for _, trigger := range triggers {
			rs.triggerRecorderMap.assign(trigger, r)
		}
	}

	rs.triggerSet.mutableRouter.updateRouter(rs.triggerSet.getRouter())
}

// TODO: Delete or disable?
func (rs *RecorderSet) disableRecorder(r *crd.Recorder) {
	function := r.Spec.Function
	triggers := r.Spec.Triggers

	log.Info("Disabling recorder!")

	// Account for function
	err := rs.functionRecorderMap.remove(function)
	if err != nil {
		log.Error("Unable to disable recorder")
	}

	// Account for explicitly added triggers
	if len(triggers) != 0 {
		for _, trigger := range triggers {
			err := rs.triggerRecorderMap.remove(trigger)
			if err != nil {
				log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
			}
		}
	} else {
		// Account for implicitly added triggers
		for _, t := range rs.triggerSet.triggerStore.List() {
			trigger := *t.(*crd.HTTPTrigger)
			if trigger.Spec.FunctionReference.Name == function {
				err := rs.triggerRecorderMap.remove(trigger.Metadata.Name)
				if err != nil {
					log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
				}
			}
		}
	}

	rs.triggerSet.mutableRouter.updateRouter(rs.triggerSet.getRouter())
}

func (rs *RecorderSet) updateRecorder(old *crd.Recorder, newer *crd.Recorder) {
	if newer.Spec.Enabled == true {
		rs.newRecorder(newer) // TODO: Test this
	} else {
		rs.disableRecorder(old)
	}
}

func (rs *RecorderSet) DeleteTriggerFromRecorderMap(trigger *crd.HTTPTrigger) {
	err := rs.triggerRecorderMap.remove(trigger.Metadata.Name)
	if err != nil {
		log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
	}
}

func (rs *RecorderSet) DeleteFunctionFromRecorderMap(function *crd.Function) {
	err := rs.functionRecorderMap.remove(function.Metadata.Name)
	if err != nil {
		log.Error("Failed to remove function from functionRecorderMap: ", err)
	}
}
