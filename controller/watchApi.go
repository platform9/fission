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

package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func (a *API) WatchApiList(w http.ResponseWriter, r *http.Request) {
	watches, err := a.fissionClient.KubernetesWatchTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(watches.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) WatchApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var watch crd.KubernetesWatchTrigger
	err = json.Unmarshal(body, &watch)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// TODO check for duplicate watches

	wnew, err := a.fissionClient.KubernetesWatchTriggers(watch.Metadata.Namespace).Create(&watch)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(wnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	a.respondWithSuccess(w, resp)
}

func (a *API) WatchApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["watch"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	watch, err := a.fissionClient.KubernetesWatchTriggers(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(watch)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) WatchApiUpdate(w http.ResponseWriter, r *http.Request) {
	a.respondWithError(w, fission.MakeError(fission.ErrorNotImplmented,
		"Not implemented"))
}

func (a *API) WatchApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["watch"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.KubernetesWatchTriggers(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
