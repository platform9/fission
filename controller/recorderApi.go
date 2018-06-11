/*
Copyright 2017 The Fission Authors.

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

package controller

import (
	"encoding/json"
	"net/http"
	"io/ioutil"
	"github.com/fission/fission/crd"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/gorilla/mux"
	"github.com/fission/fission"
)

func (a *API) RecorderApiList(w http.ResponseWriter, r*http.Request) {
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	recorders, err := a.fissionClient.Recorders(ns).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	resp, err := json.Marshal(recorders.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecorderApiCreate(w http.ResponseWriter, r*http.Request) {
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		log.Info("In RecoderApiCreate, error with ioutil.ReadAll")	// TODO: Remove later
		a.respondWithError(w, err)
		return
	}

	var recorder crd.Recorder
	err = json.Unmarshal(body, &recorder)
	if err != nil {
		log.Info("In RecoderApiCreate, error with json.Unmarshal")
		a.respondWithError(w, err)
		return
	}

	// check if namespace exists, if not create it.
	/*
	err = a.createNsIfNotExists(mqTrigger.Metadata.Namespace)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	*/

	tnew, err := a.fissionClient.Recorders("default").Create(&recorder)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(tnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	a.respondWithSuccess(w, resp)
}

func (a *API) RecorderApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["recorder"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	recorder, err := a.fissionClient.Recorders(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	resp, err := json.Marshal(recorder)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}
/*
func (a *API) MessageQueueTriggerApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["mqTrigger"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	mqTrigger, err := a.fissionClient.MessageQueueTriggers(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	resp, err := json.Marshal(mqTrigger)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}
*/

func (a *API) RecorderApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["recorder"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var recorder crd.Recorder
	err = json.Unmarshal(body, &recorder)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != recorder.Metadata.Name {
		err = fission.MakeError(fission.ErrorInvalidArgument, "Recorder name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	rnew, err := a.fissionClient.Recorders(recorder.Metadata.Namespace).Update(&recorder)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(rnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) RecorderApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["recorder"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.Recorders(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, []byte(""))
}

/*
func (a *API) MessageQueueTriggerApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["mqTrigger"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.MessageQueueTriggers(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, []byte(""))
}
*/
