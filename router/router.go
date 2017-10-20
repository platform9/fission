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

/*

This is the Fission Router package.

Its job is to:

  1. Keep track of HTTP triggers and their mappings to functions

     Use the controller API to get and watch this state.

  2. Given a function, get a reference to a routable function run service

     Use the ContainerPoolManager API to get a service backed by one
     or more function run containers.  The container(s) backing the
     service may be newly created, or they might be reused.  The only
     requirement is that one or more containers backs the service.

  3. Forward the request to the service, and send the response back.

     Plain ol HTTP.

*/

package router

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/fission/fission/crd"
	poolmgrClient "github.com/fission/fission/poolmgr/client"
)

// request url ---[mux]---> Function(name,uid) ----[fmap]----> k8s service url

// request url ---[trigger]---> Function(name, deployment) ----[deployment]----> Function(name, uid) ----[pool mgr]---> k8s service url

func router(httpTriggerSet *HTTPTriggerSet) *mutableRouter {
	muxRouter := mux.NewRouter()
	mr := NewMutableRouter(muxRouter)
	httpTriggerSet.subscribeRouter(mr)
	return mr
}

func serve(port int, httpTriggerSet *HTTPTriggerSet) {
	mr := router(httpTriggerSet)
	url := fmt.Sprintf(":%v", port)
	http.ListenAndServe(url, handlers.LoggingHandler(os.Stdout, mr))
}

func Start(port int, poolmgrUrl string) {
	fmap := makeFunctionServiceMap(time.Minute)

	fissionClient, _, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Fatalf("Error connecting to kubernetes API: %v", err)
	}
	poolmgr := poolmgrClient.MakeClient(poolmgrUrl)
	resolver := makeFunctionReferenceResolver(fissionClient)
	triggers := makeHTTPTriggerSet(fmap, fissionClient, poolmgr, resolver)
	log.Printf("Starting router at port %v\n", port)
	serve(port, triggers)
}
