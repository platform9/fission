package controller

import (
	"net/http"
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

	resp, err := redis.RecordsFilterByFunction(query, recorders, triggers)
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

	resp, err := redis.RecordsFilterByTrigger(query, recorders, triggers)
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