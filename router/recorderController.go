package router

import (
	"github.com/fission/fission/crd"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	k8sCache "k8s.io/client-go/tools/cache"
)

type RecorderSet struct {
	httpTriggerSet *HTTPTriggerSet

	crdClient *rest.RESTClient

	recStore      k8sCache.Store
	recController k8sCache.Controller

	functionRecorderMap *functionRecorderMap
	triggerRecorderMap  *triggerRecorderMap
}

func MakeRecorderSet(httpTriggerSet *HTTPTriggerSet, crdClient *rest.RESTClient, rStore k8sCache.Store, frmap *functionRecorderMap, trmap *triggerRecorderMap) *RecorderSet {
	recorderSet := &RecorderSet{
		httpTriggerSet:      httpTriggerSet,
		crdClient:           crdClient,
		recStore:            rStore,
		functionRecorderMap: frmap,
		triggerRecorderMap:  trmap,
	}
	recorderSet.recStore, recorderSet.recController = httpTriggerSet.initRecorderController()
	return recorderSet
}

// All new recorders are by default enabled
func (rs *RecorderSet) newRecorder(r *crd.Recorder) {
	function := r.Spec.Function
	triggers := r.Spec.Triggers

	// If triggers are not explicitly specified during the creation of this recorder,
	// keep track of those associated with the function specified [implicitly added triggers]
	needTrackByFunction := len(triggers) == 0

	rs.functionRecorderMap.assign(function, r)

	if needTrackByFunction {
		for _, t := range rs.httpTriggerSet.triggerStore.List() {
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

	rs.httpTriggerSet.mutableRouter.updateRouter(rs.httpTriggerSet.getRouter())
}

// TODO: Delete or disable?
func (rs *RecorderSet) disableRecorder(r *crd.Recorder) {
	function := r.Spec.Function
	triggers := r.Spec.Triggers

	log.Info("Disabling recorder!")

	// Account for function
	err := rs.functionRecorderMap.remove(function)
	if err != nil {
		log.Error("Error disabling recorder (failed to remove function from functionRecorderMap): ", err)
	}

	// Account for explicitly added triggers
	if len(triggers) != 0 {
		for _, trigger := range triggers {
			err := rs.triggerRecorderMap.remove(trigger)
			if err != nil {
				log.Error("Error disabling recorder (failed to remove triggers from triggerRecorderMap): ", err)
			}
		}
	} else {
		// Account for implicitly added triggers
		for _, t := range rs.httpTriggerSet.triggerStore.List() {
			trigger := *t.(*crd.HTTPTrigger)
			if trigger.Spec.FunctionReference.Name == function {
				err := rs.triggerRecorderMap.remove(trigger.Metadata.Name)
				if err != nil {
					log.Error("Failed to remove trigger from triggerRecorderMap: ", err)
				}
			}
		}
	}

	rs.httpTriggerSet.mutableRouter.updateRouter(rs.httpTriggerSet.getRouter())
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
