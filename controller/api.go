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
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/fission/fission"
	"github.com/fission/fission/fission/logdb"
	"github.com/fission/fission/tpr"
)

type (
	API struct {
		fissionClient     *tpr.FissionClient
		storageServiceUrl string
		builderManagerUrl string
		workflowApiUrl    string
	}

	logDBConfig struct {
		httpURL  string
		username string
		password string
	}
)

func MakeAPI() (*API, error) {
	api, err := makeTPRBackedAPI()

	u := os.Getenv("STORAGE_SERVICE_URL")
	if len(u) > 0 {
		api.storageServiceUrl = strings.TrimSuffix(u, "/")
	} else {
		api.storageServiceUrl = "http://storagesvc"
	}

	u = os.Getenv("BUILDER_MANAGER_URL")
	if len(u) > 0 {
		api.builderManagerUrl = strings.TrimSuffix(u, "/")
	} else {
		api.builderManagerUrl = "http://buildermgr"
	}

	wfEnv := os.Getenv("WORKFLOW_API_URL")
	if len(u) > 0 {
		api.workflowApiUrl = strings.TrimSuffix(wfEnv, "/")
	} else {
		api.workflowApiUrl = "http://workflows-apiserver"
	}

	return api, err
}

func (api *API) respondWithSuccess(w http.ResponseWriter, resp []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err := w.Write(resp)
	if err != nil {
		// this will probably fail too, but try anyway
		api.respondWithError(w, err)
	}
}

func (api *API) respondWithError(w http.ResponseWriter, err error) {
	debug.PrintStack()

	// this error type comes with an HTTP code, so just use that
	se, ok := err.(*kerrors.StatusError)
	if ok {
		http.Error(w, string(se.ErrStatus.Reason), int(se.ErrStatus.Code))
		return
	}

	code, msg := fission.GetHTTPError(err)
	log.Errorf("Error: %v: %v", code, msg)
	http.Error(w, msg, code)
}

func (api *API) getLogDBConfig(dbType string) logDBConfig {
	dbType = strings.ToUpper(dbType)
	// retrieve db auth config from the env
	url := os.Getenv(fmt.Sprintf("%s_URL", dbType))
	if url == "" {
		// set up default database url
		url = logdb.INFLUXDB_URL
	}
	username := os.Getenv(fmt.Sprintf("%s_USERNAME", dbType))
	password := os.Getenv(fmt.Sprintf("%s_PASSWORD", dbType))
	return logDBConfig{
		httpURL:  url,
		username: username,
		password: password,
	}
}

func (api *API) HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\"message\": \"Fission API\", \"version\": \"0.3.0\"}\n")
}

func (api *API) ApiVersionMismatchHandler(w http.ResponseWriter, r *http.Request) {
	err := fission.MakeError(fission.ErrorNotFound, "Fission server supports API v2 only -- v1 is not supported. Please upgrade your Fission client/CLI.")
	api.respondWithError(w, err)
}

func (api *API) Serve(port int) {
	r := mux.NewRouter()
	// Give a useful error message if an older CLI attempts to make a request
	r.HandleFunc(`/v1/{rest:[a-zA-Z0-9=\-\/]+}`, api.ApiVersionMismatchHandler)
	r.HandleFunc("/", api.HomeHandler)

	r.HandleFunc("/v2/packages", api.PackageApiList).Methods("GET")
	r.HandleFunc("/v2/packages", api.PackageApiCreate).Methods("POST")
	r.HandleFunc("/v2/packages/{package}", api.PackageApiGet).Methods("GET")
	r.HandleFunc("/v2/packages/{package}", api.PackageApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/packages/{package}", api.PackageApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/functions", api.FunctionApiList).Methods("GET")
	r.HandleFunc("/v2/functions", api.FunctionApiCreate).Methods("POST")
	r.HandleFunc("/v2/functions/{function}", api.FunctionApiGet).Methods("GET")
	r.HandleFunc("/v2/functions/{function}", api.FunctionApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/functions/{function}", api.FunctionApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/triggers/http", api.HTTPTriggerApiList).Methods("GET")
	r.HandleFunc("/v2/triggers/http", api.HTTPTriggerApiCreate).Methods("POST")
	r.HandleFunc("/v2/triggers/http/{httpTrigger}", api.HTTPTriggerApiGet).Methods("GET")
	r.HandleFunc("/v2/triggers/http/{httpTrigger}", api.HTTPTriggerApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/triggers/http/{httpTrigger}", api.HTTPTriggerApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/environments", api.EnvironmentApiList).Methods("GET")
	r.HandleFunc("/v2/environments", api.EnvironmentApiCreate).Methods("POST")
	r.HandleFunc("/v2/environments/{environment}", api.EnvironmentApiGet).Methods("GET")
	r.HandleFunc("/v2/environments/{environment}", api.EnvironmentApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/environments/{environment}", api.EnvironmentApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/watches", api.WatchApiList).Methods("GET")
	r.HandleFunc("/v2/watches", api.WatchApiCreate).Methods("POST")
	r.HandleFunc("/v2/watches/{watch}", api.WatchApiGet).Methods("GET")
	r.HandleFunc("/v2/watches/{watch}", api.WatchApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/watches/{watch}", api.WatchApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/triggers/time", api.TimeTriggerApiList).Methods("GET")
	r.HandleFunc("/v2/triggers/time", api.TimeTriggerApiCreate).Methods("POST")
	r.HandleFunc("/v2/triggers/time/{timeTrigger}", api.TimeTriggerApiGet).Methods("GET")
	r.HandleFunc("/v2/triggers/time/{timeTrigger}", api.TimeTriggerApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/triggers/time/{timeTrigger}", api.TimeTriggerApiDelete).Methods("DELETE")

	r.HandleFunc("/v2/triggers/messagequeue", api.MessageQueueTriggerApiList).Methods("GET")
	r.HandleFunc("/v2/triggers/messagequeue", api.MessageQueueTriggerApiCreate).Methods("POST")
	r.HandleFunc("/v2/triggers/messagequeue/{mqTrigger}", api.MessageQueueTriggerApiGet).Methods("GET")
	r.HandleFunc("/v2/triggers/messagequeue/{mqTrigger}", api.MessageQueueTriggerApiUpdate).Methods("PUT")
	r.HandleFunc("/v2/triggers/messagequeue/{mqTrigger}", api.MessageQueueTriggerApiDelete).Methods("DELETE")

	r.HandleFunc("/proxy/{dbType}", api.FunctionLogsApiPost).Methods("POST")
	r.HandleFunc("/proxy/storage/v1/archive", api.StorageServiceProxy)
	r.HandleFunc("/proxy/buildermgr/v1/build", api.BuilderManagerBuildProxy)
	r.HandleFunc("/proxy/workflows-apiserver/{path:.*}", api.WorkflowApiserverProxy)

	address := fmt.Sprintf(":%v", port)

	log.WithFields(log.Fields{"port": port}).Info("Server started")
	log.Fatal(http.ListenAndServe(address, handlers.LoggingHandler(os.Stdout, r)))
}
