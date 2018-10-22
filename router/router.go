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
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

func serveMetric() {
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(metricAddr, nil))
}

func Start(port int, executorUrl string) {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fmap := makeFunctionServiceMap(time.Minute)

	frmap := makeFunctionRecorderMap(time.Minute)

	trmap := makeTriggerRecorderMap(time.Minute)

	fissionClient, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		log.Fatalf("Error connecting to kubernetes API: %v", err)
	}

	err = fissionClient.WaitForCRDs()
	if err != nil {
		log.Fatalf("Error waiting for CRDs: %v", err)
	}

	restClient := fissionClient.GetCrdClient()

	executor := executorClient.MakeClient(executorUrl)

	timeout, err := time.ParseDuration(os.Getenv("ROUTER_ROUND_TRIP_TIMEOUT"))
	if err != nil {
		log.Fatalf("Failed to parse timeout: %v", err)
	}

	timeoutExponent, err := strconv.Atoi(os.Getenv("ROUTER_ROUNDTRIP_TIMEOUT_EXPONENT"))
	if err != nil {
		log.Fatalf("Failed to parse timeout exponent: %v", err)
	}

	keepAlive, err := time.ParseDuration(os.Getenv("ROUTER_ROUND_TRIP_KEEP_ALIVE_TIME"))
	if err != nil {
		log.Fatalf("Failed to parse keep alive time: %v", err)
	}

	maxRetries, err := strconv.Atoi(os.Getenv("ROUTER_ROUND_TRIP_MAX_RETRIES"))
	if err != nil {
		log.Fatalf("Failed to parse max retry times: %v", err)
	}

	isDebugEnv, err := strconv.ParseBool(os.Getenv("DEBUG_ENV"))
	if err != nil {
		log.Fatalf("Failed to parse DEBUG_ENV: %v", err)
	}

	// svcAddrRetryCount is the max times for RetryingRoundTripper to retry with a specific service address
	svcAddrRetryCount, err := strconv.Atoi(os.Getenv("ROUTER_ROUND_TRIP_SVC_ADDRESS_MAX_RETRIES"))
	if err != nil {
		svcAddrRetryCount = 5
		log.Printf("Failed to parse svc address retry conunt, set it to default value(5): %v", err)
	}

	// svcAddrUpdateTimeout is the timeout setting for a goroutine to wait for the update of a service entry.
	// If the update process cannot be done within the timeout window, consider it failed.
	svcAddrUpdateTimeout, err := time.ParseDuration(os.Getenv("ROUTER_ROUND_TRIP_SVC_ADDRESS_UPDATE_TIMEOUT"))
	if err != nil {
		svcAddrUpdateTimeout = 30 * time.Second
		log.Printf("Failed to parse svc address update timeout, set it to default value(30): %v", err)
	}

	triggers, _, fnStore := makeHTTPTriggerSet(fmap, frmap, trmap, fissionClient, kubeClient, executor, restClient, &tsRoundTripperParams{
		timeout:           timeout,
		timeoutExponent:   timeoutExponent,
		keepAlive:         keepAlive,
		maxRetries:        maxRetries,
		svcAddrRetryCount: svcAddrRetryCount,
	}, isDebugEnv, MakeUpdateLocks(svcAddrUpdateTimeout))

	resolver := makeFunctionReferenceResolver(fnStore)

	go serveMetric()

	log.Printf("Starting router at port %v\n", port)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve(ctx, port, triggers, resolver)
}
