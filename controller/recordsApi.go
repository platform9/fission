/*
Copyright 2018 The Fission Authors.

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
	"net/http"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission/redis"
)

func (a *API) RecordsApiListAll(w http.ResponseWriter, r *http.Request) {
	resp, err := redis.RecordsListAll(a.logger.Named("redis"))
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["function"]

	recorders, err := a.fissionClient.Recorders(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	triggers, err := a.fissionClient.HTTPTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := redis.RecordsFilterByFunction(a.logger.Named("redis"), query, recorders, triggers)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["trigger"]

	recorders, err := a.fissionClient.Recorders(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	triggers, err := a.fissionClient.HTTPTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := redis.RecordsFilterByTrigger(a.logger.Named("redis"), query, recorders, triggers)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByTime(w http.ResponseWriter, r *http.Request) {
	from := r.FormValue("from")
	to := r.FormValue("to")

	resp, err := redis.RecordsFilterByTime(a.logger.Named("redis"), from, to)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}
