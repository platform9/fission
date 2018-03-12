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
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
)

// request url ---[mux]---> Function(name,uid) ----[fmap]----> k8s service url

// request url ---[trigger]---> Function(name, deployment) ----[deployment]----> Function(name, uid) ----[pool mgr]---> k8s service url

func router(ctx context.Context, httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver) *mutableRouter {
	muxRouter := mux.NewRouter()
	mr := NewMutableRouter(muxRouter)
	muxRouter.Use(fission.LoggingMiddleware)
	httpTriggerSet.subscribeRouter(ctx, mr, resolver)
	return mr
}

func serve(ctx context.Context, port int, httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver) {
	mr := router(ctx, httpTriggerSet, resolver)
	url := fmt.Sprintf(":%v", port)
	http.ListenAndServe(url, mr)
}

func Start(port int, executorUrl string) {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fmap := makeFunctionServiceMap(time.Minute)

	fissionClient, _, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Fatalf("Error connecting to kubernetes API: %v", err)
	}

	restClient := fissionClient.GetCrdClient()

	executor := executorClient.MakeClient(executorUrl)
	triggers, _, fnStore := makeHTTPTriggerSet(fmap, fissionClient, executor, restClient)
	resolver := makeFunctionReferenceResolver(fnStore)

	log.Printf("Starting router at port %v\n", port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve(ctx, port, triggers, resolver)
}
