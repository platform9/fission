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
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *API) SecretGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["secret"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	secret, err := a.kubernetesClient.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		a.logger.Error("error getting secret",
			zap.Error(err),
			zap.String("secret_name", name),
			zap.String("namespace", ns))
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(secret)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}
