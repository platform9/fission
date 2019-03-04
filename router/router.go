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
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
	"github.com/fission/fission/throttler"
)

// request url ---[mux]---> Function(name,uid) ----[fmap]----> k8s service url

// request url ---[trigger]---> Function(name, deployment) ----[deployment]----> Function(name, uid) ----[pool mgr]---> k8s service url

func router(ctx context.Context, logger *zap.Logger, httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver) *mutableRouter {
	muxRouter := mux.NewRouter()
	mr := NewMutableRouter(logger, muxRouter)
	muxRouter.Use(fission.LoggingMiddleware(logger))
	httpTriggerSet.subscribeRouter(ctx, mr, resolver)
	return mr
}

func serve(ctx context.Context, logger *zap.Logger, port int, httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver) {
	mr := router(ctx, logger, httpTriggerSet, resolver)
	url := fmt.Sprintf(":%v", port)
	http.ListenAndServe(url, &ochttp.Handler{
		Handler: mr,
		StartOptions: trace.StartOptions{
			Sampler: trace.AlwaysSample(),
		},
	})
}

func serveMetric(logger *zap.Logger) {
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(metricAddr, nil)

	logger.Fatal("done listening on metrics endpoint", zap.Error(err))
}

func Start(logger *zap.Logger, port int, executorUrl string) {
	// setup a signal handler for SIGTERM
	fission.SetupStackTraceHandler()

	fmap := makeFunctionServiceMap(logger, time.Minute)

	frmap := makeFunctionRecorderMap(logger, time.Minute)

	trmap := makeTriggerRecorderMap(logger, time.Minute)

	fissionClient, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		logger.Fatal("error connecting to kubernetes API", zap.Error(err))
	}

	err = fissionClient.WaitForCRDs()
	if err != nil {
		logger.Fatal("error waiting for CRDs", zap.Error(err))
	}

	restClient := fissionClient.GetCrdClient()

	executor := executorClient.MakeClient(logger, executorUrl)

	timeoutStr := os.Getenv("ROUTER_ROUND_TRIP_TIMEOUT")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		logger.Fatal("failed to parse timeout duration from 'ROUTER_ROUND_TRIP_TIMEOUT'",
			zap.Error(err),
			zap.String("value", timeoutStr))
	}

	timeoutExponentStr := os.Getenv("ROUTER_ROUNDTRIP_TIMEOUT_EXPONENT")
	timeoutExponent, err := strconv.Atoi(timeoutExponentStr)
	if err != nil {
		logger.Fatal("failed to parse timeout exponent from 'ROUTER_ROUNDTRIP_TIMEOUT_EXPONENT'",
			zap.Error(err),
			zap.String("value", timeoutExponentStr))
	}

	keepAliveStr := os.Getenv("ROUTER_ROUND_TRIP_KEEP_ALIVE_TIME")
	keepAlive, err := time.ParseDuration(keepAliveStr)
	if err != nil {
		logger.Fatal("failed to parse keep alive duration from 'ROUTER_ROUND_TRIP_KEEP_ALIVE_TIME'",
			zap.Error(err),
			zap.String("value", keepAliveStr))
	}

	maxRetriesStr := os.Getenv("ROUTER_ROUND_TRIP_MAX_RETRIES")
	maxRetries, err := strconv.Atoi(maxRetriesStr)
	if err != nil {
		logger.Fatal("failed to parse max retries from 'ROUTER_ROUND_TRIP_MAX_RETRIES'",
			zap.Error(err),
			zap.String("value", maxRetriesStr))
	}

	isDebugEnvStr := os.Getenv("DEBUG_ENV")
	isDebugEnv, err := strconv.ParseBool(isDebugEnvStr)
	if err != nil {
		logger.Fatal("failed to parse debug env from 'DEBUG_ENV'",
			zap.Error(err),
			zap.String("value", isDebugEnvStr))
	}

	// svcAddrRetryCount is the max times for RetryingRoundTripper to retry with a specific service address
	svcAddrRetryCountStr := os.Getenv("ROUTER_ROUND_TRIP_SVC_ADDRESS_MAX_RETRIES")
	svcAddrRetryCount, err := strconv.Atoi(svcAddrRetryCountStr)
	if err != nil {
		svcAddrRetryCount = 5
		logger.Info("failed to parse service address retry count from 'ROUTER_ROUND_TRIP_SVC_ADDRESS_MAX_RETRIES' - set to the default value",
			zap.Error(err),
			zap.String("value", svcAddrRetryCountStr),
			zap.Int("default", svcAddrRetryCount))
	}

	// svcAddrUpdateTimeout is the timeout setting for a goroutine to wait for the update of a service entry.
	// If the update process cannot be done within the timeout window, consider it failed.
	svcAddrUpdateTimeoutStr := os.Getenv("ROUTER_ROUND_TRIP_SVC_ADDRESS_UPDATE_TIMEOUT")
	svcAddrUpdateTimeout, err := time.ParseDuration(os.Getenv("ROUTER_ROUND_TRIP_SVC_ADDRESS_UPDATE_TIMEOUT"))
	if err != nil {
		svcAddrUpdateTimeout = 30 * time.Second
		logger.Info("failed to parse service address update timeout duration from 'ROUTER_ROUND_TRIP_SVC_ADDRESS_UPDATE_TIMEOUT' - set to the default value",
			zap.Error(err),
			zap.String("value", svcAddrUpdateTimeoutStr),
			zap.Duration("default", svcAddrUpdateTimeout))
	}

	triggers, _, fnStore := makeHTTPTriggerSet(logger.Named("triggerset"), fmap, frmap, trmap, fissionClient, kubeClient, executor, restClient, &tsRoundTripperParams{
		timeout:           timeout,
		timeoutExponent:   timeoutExponent,
		keepAlive:         keepAlive,
		maxRetries:        maxRetries,
		svcAddrRetryCount: svcAddrRetryCount,
	}, isDebugEnv, throttler.MakeThrottler(svcAddrUpdateTimeout))

	resolver := makeFunctionReferenceResolver(fnStore)

	go serveMetric(logger)

	logger.Info("starting router", zap.Int("port", port))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve(ctx, logger, port, triggers, resolver)
}
