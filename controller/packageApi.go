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
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/1.5/pkg/api"

	"github.com/fission/fission"
	"github.com/fission/fission/tpr"
)

func (a *API) PackageApiList(w http.ResponseWriter, r *http.Request) {
	funcs, err := a.fissionClient.Packages(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(funcs.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) PackageApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f tpr.Package
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	err = validateResourceName(f.Metadata.Name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// Ensure size limits
	if len(f.Spec.Literal) > 256*1024 {
		err := fission.MakeError(fission.ErrorInvalidArgument, "Package literal larger than 256K")
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.Packages(f.Metadata.Namespace).Create(&f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(fnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	a.respondWithSuccess(w, resp)
}

func (a *API) PackageApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["package"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = api.NamespaceDefault
	}
	raw := r.FormValue("raw") // just the deployment pkg

	f, err := a.fissionClient.Packages(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var resp []byte
	if raw != "" {
		resp = []byte(f.Spec.Literal)
	} else {
		resp, err = json.Marshal(f)
		if err != nil {
			a.respondWithError(w, err)
			return
		}
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) PackageApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["package"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f tpr.Package
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != f.Metadata.Name {
		err = fission.MakeError(fission.ErrorInvalidArgument, "Package name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.Packages(f.Metadata.Namespace).Update(&f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(fnew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) PackageApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["package"]
	ns := vars["namespace"]
	if len(ns) == 0 {
		ns = api.NamespaceDefault
	}

	err := a.fissionClient.Packages(ns).Delete(name, &api.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
