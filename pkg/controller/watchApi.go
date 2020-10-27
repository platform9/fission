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

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fv1 "github.com/fission/fission/pkg/apis/core/v1"
	ferror "github.com/fission/fission/pkg/error"
)

func RegisterWatchRoute(ws *restful.WebService) {
	tags := []string{"KubernetesWatch"}
	specTag = append(specTag, spec.Tag{TagProps: spec.TagProps{Name: "KubernetesWatch", Description: "KubernetesWatch Operation"}})

	ws.Route(
		ws.GET("/v2/watches").
			Doc("List all kubernetes watch").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.QueryParameter("namespace", "Namespace of kubernetesWatch").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Writes([]fv1.KubernetesWatchTrigger{}).
			Returns(http.StatusOK, "List of kubernetesWatchs", []fv1.KubernetesWatchTrigger{}))

	ws.Route(
		ws.POST("/v2/watches").
			Doc("Create kubernetes watch").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Produces(restful.MIME_JSON).
			Reads(fv1.KubernetesWatchTrigger{}).
			Writes(metav1.ObjectMeta{}).
			Returns(http.StatusCreated, "ObjectMeta of created kubernetesWatch", metav1.ObjectMeta{}))

	ws.Route(
		ws.GET("/v2/watches/{watch}").
			Doc("Get detail of kubernetes watch").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("watch", "KubernetesWatch name").DataType("string").DefaultValue("").Required(true)).
			Param(ws.QueryParameter("namespace", "Namespace of kubernetesWatch").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Writes(fv1.KubernetesWatchTrigger{}). // on the response
			Returns(http.StatusOK, "A kubernetesWatch", fv1.KubernetesWatchTrigger{}))

	ws.Route(
		ws.PUT("/v2/watches/{watch}").
			Doc("Update kubernetes watch").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("watch", "KubernetesWatch name").DataType("string").DefaultValue("").Required(true)).
			Produces(restful.MIME_JSON).
			Reads(fv1.KubernetesWatchTrigger{}).
			Writes(metav1.ObjectMeta{}). // on the response
			Returns(http.StatusOK, "ObjectMeta of updated kubernetesWatch", metav1.ObjectMeta{}))

	ws.Route(
		ws.DELETE("/v2/watches/{watch}").
			Doc("Delete kubernetes watch").
			Metadata(restfulspec.KeyOpenAPITags, tags).
			To(func(req *restful.Request, resp *restful.Response) {
				resp.ResponseWriter.WriteHeader(http.StatusOK)
			}).
			Param(ws.PathParameter("watch", "KubernetesWatch name").DataType("string").DefaultValue("").Required(true)).
			Param(ws.QueryParameter("namespace", "Namespace of kubernetesWatch").DataType("string").DefaultValue(metav1.NamespaceAll).Required(false)).
			Produces(restful.MIME_JSON).
			Returns(http.StatusOK, "Only HTTP status returned", nil))
}

func (a *API) WatchApiList(w http.ResponseWriter, r *http.Request) {
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	watches, err := a.fissionClient.CoreV1().KubernetesWatchTriggers(ns).List(metav1.ListOptions{})
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

	var watch fv1.KubernetesWatchTrigger
	err = json.Unmarshal(body, &watch)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// TODO check for duplicate watches
	// TODO check for duplicate watches -> we probably wont need it?
	// check if namespace exists, if not create it.
	err = a.createNsIfNotExists(watch.ObjectMeta.Namespace)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	wnew, err := a.fissionClient.CoreV1().KubernetesWatchTriggers(watch.ObjectMeta.Namespace).Create(&watch)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(wnew.ObjectMeta)
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
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	watch, err := a.fissionClient.CoreV1().KubernetesWatchTriggers(ns).Get(name, metav1.GetOptions{})
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
	a.respondWithError(w, ferror.MakeError(ferror.ErrorNotImplemented,
		"Not implemented"))
}

func (a *API) WatchApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["watch"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.CoreV1().KubernetesWatchTriggers(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
