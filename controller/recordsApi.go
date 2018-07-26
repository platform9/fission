package controller

import (
	"net/http"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/fission/fission/redis"
	"github.com/gorilla/mux"
)

func (a *API) RecordsApiListAll(w http.ResponseWriter, r *http.Request) {
	resp, err := redis.RecordsListAll()
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["function"]

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	recorders, err := a.fissionClient.Recorders(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, errors.New("Problem with recorders list"))
		return
	}

	triggers, err := a.fissionClient.HTTPTriggers(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, errors.New("Problem with triggers list"))
		return
	}

	resp, err := redis.RecordsFilterByFunction(query, ns, recorders, triggers)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByTrigger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := vars["trigger"]

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

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

	resp, err := redis.RecordsFilterByTrigger(query, ns, recorders, triggers)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) RecordsApiFilterByTime(w http.ResponseWriter, r *http.Request) {
	from := r.FormValue("from")
	to := r.FormValue("to")

	resp, err := redis.RecordsFilterByTime(from, to)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}