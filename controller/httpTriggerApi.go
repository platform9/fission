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
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func (a *API) HTTPTriggerApiList(w http.ResponseWriter, r *http.Request) {
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	triggers, err := a.fissionClient.HTTPTriggers(ns).List(metav1.ListOptions{})
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

func (a *API) checkHTTPTriggerDuplicates(t *crd.HTTPTrigger) error {
	triggers, err := a.fissionClient.HTTPTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ht := range triggers.Items {
		if ht.Spec.RelativeURL == t.Spec.RelativeURL && ht.Spec.Method == t.Spec.Method && ht.Spec.Host == t.Spec.Host {
			return fission.MakeError(fission.ErrorNameExists,
				fmt.Sprintf("HTTPTrigger with same Host, URL & method already exists (%v)",
					ht.Metadata.Name))
		}
	}
	return nil
}

func (a *API) HTTPTriggerApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var t crd.HTTPTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// Ensure we don't have a duplicate HTTP route defined (same URL and method)
	err = a.checkHTTPTriggerDuplicates(&t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// check if namespace exists, if not create it.
	err = a.createNsIfNotExists(t.Metadata.Namespace)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	tnew, err := a.fissionClient.HTTPTriggers(t.Metadata.Namespace).Create(&t)
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

func (a *API) HTTPTriggerApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["httpTrigger"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	t, err := a.fissionClient.HTTPTriggers(ns).Get(name)
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

func (a *API) HTTPTriggerApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["httpTrigger"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var t crd.HTTPTrigger
	err = json.Unmarshal(body, &t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != t.Metadata.Name {
		err = fission.MakeError(fission.ErrorInvalidArgument, "HTTPTrigger name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	err = a.checkHTTPTriggerDuplicates(&t)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	tnew, err := a.fissionClient.HTTPTriggers(t.Metadata.Namespace).Update(&t)
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

func (a *API) HTTPTriggerApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["httpTrigger"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.HTTPTriggers(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
