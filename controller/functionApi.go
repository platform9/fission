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
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
)

func (a *API) getIstioServiceLabels(fnName string) map[string]string {
	return map[string]string{
		"functionName": fnName,
	}
}

func (a *API) FunctionApiList(w http.ResponseWriter, r *http.Request) {
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceAll
	}

	funcs, err := a.fissionClient.Functions(ns).List(metav1.ListOptions{})
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

func (a *API) FunctionApiCreate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f crd.Function
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// check if namespace exists, if not create it.
	err = a.createNsIfNotExists(f.Metadata.Namespace)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.Functions(f.Metadata.Namespace).Create(&f)
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

func (a *API) FunctionApiGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	f, err := a.fissionClient.Functions(ns).Get(name)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	resp, err := json.Marshal(f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	a.respondWithSuccess(w, resp)
}

func (a *API) FunctionApiUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	var f crd.Function
	err = json.Unmarshal(body, &f)
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	if name != f.Metadata.Name {
		err = fission.MakeError(fission.ErrorInvalidArgument, "Function name doesn't match URL")
		a.respondWithError(w, err)
		return
	}

	fnew, err := a.fissionClient.Functions(f.Metadata.Namespace).Update(&f)
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

func (a *API) FunctionApiDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["function"]
	ns := a.extractQueryParamFromRequest(r, "namespace")
	if len(ns) == 0 {
		ns = metav1.NamespaceDefault
	}

	err := a.fissionClient.Functions(ns).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	a.respondWithSuccess(w, []byte(""))
}

// FunctionLogsApiPost establishes a proxy server to log database, and redirect
// query command send from client to database then proxy back the db response.
func (a *API) FunctionLogsApiPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	// get dbType from url
	dbType := vars["dbType"]

	// find correspond db http url
	dbCnf := a.getLogDBConfig(dbType)

	svcUrl, err := url.Parse(dbCnf.httpURL)
	if err != nil {
		a.logger.Error("failed parse url to establish proxy to database for function logs",
			zap.Error(err),
			zap.String("database_url", dbCnf.httpURL))
	}
	// set up proxy server director
	director := func(req *http.Request) {
		// only replace url Scheme and Host to remote influxDB
		// and leave query string intact
		req.URL.Scheme = svcUrl.Scheme
		req.URL.Host = svcUrl.Host
		req.URL.Path = svcUrl.Path
		// set up http basic auth for database authentication
		req.SetBasicAuth(dbCnf.username, dbCnf.password)
	}
	proxy := &httputil.ReverseProxy{
		Director: director,
	}
	proxy.ServeHTTP(w, r)
}

// FunctionPodLogs : Get logs for a function directly from pod
func (a *API) FunctionPodLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fnName := vars["function"]
	ns := vars["namespace"]

	if len(ns) == 0 {
		ns = "fission-function"
	}

	f, err := a.fissionClient.Functions(ns).Get(fnName)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	envName := f.Spec.Environment.Name

	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// Get function Pods first
	selector := "functionName=" + fnName
	podList, err := a.kubernetesClient.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		a.respondWithError(w, err)
		return
	}

	// Get the logs for last Pod executed
	pods := podList.Items
	sort.Slice(pods, func(i, j int) bool {
		itime := pods[i].ObjectMeta.CreationTimestamp.Time
		jtime := pods[j].ObjectMeta.CreationTimestamp.Time
		return itime.After(jtime)
	})

	podLogOpts := apiv1.PodLogOptions{Container: envName} // Only the env container, not fetcher
	var podLogsReq *restclient.Request
	if len(pods) > 0 {
		podLogsReq = a.kubernetesClient.CoreV1().Pods(ns).GetLogs(pods[0].ObjectMeta.Name, &podLogOpts)
	} else {
		a.respondWithError(w, errors.New("No active pods found"))
		return
	}

	podLogs, err := podLogsReq.Stream()
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	defer podLogs.Close()

	_, err = io.Copy(w, podLogs)
	if err != nil {
		a.respondWithError(w, err)
		return
	}
	return
}
