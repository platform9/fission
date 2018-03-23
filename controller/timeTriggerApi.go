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
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func (a *API) TimeTriggerApiList(w http.ResponseWriter, r *http.Request) {
	triggers, err := a.fissionClient.TimeTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(triggers.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) TimeTriggerApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var t crd.TimeTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// validate
	_, err = cron.Parse(t.Spec.Cron)
	if err != nil {
		err = fission.MakeError(fission.ErrorInvalidArgument, "TimeTrigger cron spec is not valid")
		a.respondWithError(w, err)
		return
	}

	tnew, err := a.fissionClient.TimeTriggers(t.Metadata.Namespace).Create(&t)
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

func (a *API) TimeTriggerApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["timeTrigger"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	t, err := a.fissionClient.TimeTriggers(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) TimeTriggerApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["timeTrigger"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var t crd.TimeTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != t.Metadata.Name {
		err = fission.MakeError(fission.ErrorInvalidArgument, "TimeTrigger name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	_, err = cron.Parse(t.Spec.Cron)
	if err != nil {
		err = fission.MakeError(fission.ErrorInvalidArgument, "TimeTrigger cron spec is not valid")
		a.respondWithError(w, err)
		return
	}

	tnew, err := a.fissionClient.TimeTriggers(t.Metadata.Namespace).Update(&t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(tnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) TimeTriggerApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["timeTrigger"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.TimeTriggers(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
