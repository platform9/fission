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
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	config "github.com/fission/fission/featureconfig"
)

func (a *API) CanaryConfigApiCreate(w http.ResponseWriter, r *http.Request) {
	featureErr := a.featureStatus[config.CanaryFeature]
	if len(featureErr) > 0 {
		a.respondWithError(w, fission.MakeError(http.StatusInternalServerError, fmt.Sprintf("Error enabling canary feature: %v", featureErr)))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var canaryCfg crd.CanaryConfig
	err = json.Unmarshal(body, &canaryCfg)
	if err != nil {
		a.logger.Error("failed to unmarshal request body", zap.Error(err), zap.Binary("body", body))
		a.respondWithError(w, err)
		return
	}

	canaryCfgNew, err := a.fissionClient.CanaryConfigs(canaryCfg.Metadata.Namespace).Create(&canaryCfg)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(canaryCfgNew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	a.respondWithSuccess(w, resp)
}

func (a *API) CanaryConfigApiGet(w http.ResponseWriter, r *http.Request) {
	featureErr := a.featureStatus[config.CanaryFeature]
	if len(featureErr) > 0 {
		a.respondWithError(w, fission.MakeError(http.StatusInternalServerError, fmt.Sprintf("Error enabling canary feature: %v", featureErr)))
		return
	}

	vars := mux.Vars(r)
	name := vars["canaryConfig"]

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	canaryCfg, err := a.fissionClient.CanaryConfigs(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(canaryCfg)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) CanaryConfigApiList(w http.ResponseWriter, r *http.Request) {
	featureErr := a.featureStatus[config.CanaryFeature]
	if len(featureErr) > 0 {
		a.respondWithError(w, fission.MakeError(http.StatusInternalServerError, fmt.Sprintf("Error enabling canary feature: %v", featureErr)))
		return
	}

	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	canaryCfgs, err := a.fissionClient.CanaryConfigs(ns).List(metav1.ListOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(canaryCfgs.Items)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) CanaryConfigApiUpdate(w http.ResponseWriter, r *http.Request) {
	featureErr := a.featureStatus[config.CanaryFeature]
	if len(featureErr) > 0 {
		a.respondWithError(w, fission.MakeError(http.StatusInternalServerError, fmt.Sprintf("Error enabling canary feature: %v", featureErr)))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var c crd.CanaryConfig
	err = json.Unmarshal(body, &c)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	canayCfgNew, err := a.fissionClient.CanaryConfigs(c.Metadata.Namespace).Update(&c)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(canayCfgNew.Metadata)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, resp)
}

func (a *API) CanaryConfigApiDelete(w http.ResponseWriter, r *http.Request) {
	featureErr := a.featureStatus[config.CanaryFeature]
	if len(featureErr) > 0 {
		a.respondWithError(w, fission.MakeError(http.StatusInternalServerError, fmt.Sprintf("Error enabling canary feature: %v", featureErr)))
		return
	}

	vars := mux.Vars(r)
	name := vars["canaryConfig"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.CanaryConfigs(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}
