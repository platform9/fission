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
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/fission/fission/pkg/crd"
	executorClient "github.com/fission/fission/pkg/executor/client"
	"github.com/fission/fission/pkg/throttler"
)

// request url ---[mux]---> Function(name,uid) ----[fmap]----> k8s service url

// request url ---[trigger]---> Function(name, deployment) ----[deployment]----> Function(name, uid) ----[pool mgr]---> k8s service url

func router(ctx context.Context, logger *zap.Logger, httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver) *mutableRouter {
	var mr *mutableRouter

	// see issue https://github.com/fission/fission/issues/1317
	useEncodedPath, _ := strconv.ParseBool(os.Getenv("USE_ENCODED_PATH"))
	if useEncodedPath {
		mr = newMutableRouter(logger, mux.NewRouter().UseEncodedPath())
	} else {
		mr = newMutableRouter(logger, mux.NewRouter())
	}

	httpTriggerSet.subscribeRouter(ctx, mr, resolver)
	return mr
}

func serve(ctx context.Context, logger *zap.Logger, port int, tracingSamplingRate float64,
	httpTriggerSet *HTTPTriggerSet, resolver *functionReferenceResolver, displayAccessLog bool) {
	mr := router(ctx, logger, httpTriggerSet, resolver)
	url := fmt.Sprintf(":%v", port)

	http.ListenAndServe(url, &ochttp.Handler{
		Handler: mr,
		GetStartOptions: func(r *http.Request) trace.StartOptions {
			// do not trace router healthz endpoint
			if strings.Compare(r.URL.Path, "/router-healthz") == 0 {
				return trace.StartOptions{
					Sampler: trace.NeverSample(),
				}
			}
			if displayAccessLog {
				logger.Info("path", zap.String("path", r.URL.Path),
					zap.String("method", r.Method), zap.Any("header", r.Header))
			}
			return trace.StartOptions{
				Sampler: trace.ProbabilitySampler(tracingSamplingRate),
			}
		},
	})
}

func serveMetric(logger *zap.Logger) {
	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(metricAddr, nil)

	logger.Fatal("done listening on metrics endpoint", zap.Error(err))
}

// Start starts a router
func Start(logger *zap.Logger, port int, executorURL string) {
	_ = MakeAnalytics("")

	fmap := makeFunctionServiceMap(logger, time.Minute)

	fissionClient, kubeClient, _, err := crd.MakeFissionClient()
	if err != nil {
		logger.Fatal("error connecting to kubernetes API", zap.Error(err))
	}

	err = fissionClient.WaitForCRDs()
	if err != nil {
		logger.Fatal("error waiting for CRDs", zap.Error(err))
	}

	executor := executorClient.MakeClient(logger, executorURL)

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

	keepAliveTimeStr := os.Getenv("ROUTER_ROUND_TRIP_KEEP_ALIVE_TIME")
	keepAliveTime, err := time.ParseDuration(keepAliveTimeStr)
	if err != nil {
		logger.Fatal("failed to parse keep alive duration from 'ROUTER_ROUND_TRIP_KEEP_ALIVE_TIME'",
			zap.Error(err),
			zap.String("value", keepAliveTimeStr))
	}

	disableKeepAliveStr := os.Getenv("ROUTER_ROUND_TRIP_DISABLE_KEEP_ALIVE")
	disableKeepAlive, err := strconv.ParseBool(disableKeepAliveStr)
	if err != nil {
		disableKeepAlive = true
		logger.Fatal("failed to parse enable keep alive from 'ROUTER_ROUND_TRIP_DISABLE_KEEP_ALIVE'",
			zap.Error(err),
			zap.String("value", disableKeepAliveStr))
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
	svcAddrRetryCountStr := os.Getenv("ROUTER_SVC_ADDRESS_MAX_RETRIES")
	svcAddrRetryCount, err := strconv.Atoi(svcAddrRetryCountStr)
	if err != nil {
		svcAddrRetryCount = 5
		logger.Error("failed to parse service address retry count from 'ROUTER_SVC_ADDRESS_MAX_RETRIES' - set to the default value",
			zap.Error(err),
			zap.String("value", svcAddrRetryCountStr),
			zap.Int("default", svcAddrRetryCount))
	}

	// svcAddrUpdateTimeout is the timeout setting for a goroutine to wait for the update of a service entry.
	// If the update process cannot be done within the timeout window, consider it failed.
	svcAddrUpdateTimeoutStr := os.Getenv("ROUTER_SVC_ADDRESS_UPDATE_TIMEOUT")
	svcAddrUpdateTimeout, err := time.ParseDuration(os.Getenv("ROUTER_SVC_ADDRESS_UPDATE_TIMEOUT"))
	if err != nil {
		svcAddrUpdateTimeout = 30 * time.Second
		logger.Error("failed to parse service address update timeout duration from 'ROUTER_ROUND_TRIP_SVC_ADDRESS_UPDATE_TIMEOUT' - set to the default value",
			zap.Error(err),
			zap.String("value", svcAddrUpdateTimeoutStr),
			zap.Duration("default", svcAddrUpdateTimeout))
	}

	// unTapServiceTimeout is the timeout used as timeout in the request context of unTapService
	unTapServiceTimeoutstr := os.Getenv("ROUTER_UNTAP_SERVICE_TIMEOUT")
	unTapServiceTimeout, err := time.ParseDuration(unTapServiceTimeoutstr)
	if err != nil {
		unTapServiceTimeout = 3600 * time.Second
		logger.Error("failed to parse unTap service timeout duration from 'ROUTER_UNTAP_SERVICE_TIMEOUT' - set to the default value",
			zap.Error(err),
			zap.String("value", unTapServiceTimeoutstr),
			zap.Duration("default", unTapServiceTimeout))
	}

	tracingSamplingRateStr := os.Getenv("TRACING_SAMPLING_RATE")
	tracingSamplingRate, err := strconv.ParseFloat(tracingSamplingRateStr, 64)
	if err != nil {
		tracingSamplingRate = .5
		logger.Error("failed to parse tracing sampling rate from 'TRACING_SAMPLING_RATE' - set to the default value",
			zap.Error(err),
			zap.String("value", tracingSamplingRateStr),
			zap.Float64("default", tracingSamplingRate))
	}

	displayAccessLogStr := os.Getenv("DISPLAY_ACCESS_LOG")
	displayAccessLog, err := strconv.ParseBool(displayAccessLogStr)
	if err != nil {
		displayAccessLog = false
		logger.Error("failed to parse 'DISPLAY_ACCESS_LOG' - set to the default value",
			zap.Error(err),
			zap.String("value", displayAccessLogStr),
			zap.Bool("default", displayAccessLog))
	}

	triggers, _, fnStore := makeHTTPTriggerSet(logger.Named("triggerset"), fmap, fissionClient, kubeClient, executor, fissionClient.CoreV1().RESTClient(), &tsRoundTripperParams{
		timeout:           timeout,
		timeoutExponent:   timeoutExponent,
		disableKeepAlive:  disableKeepAlive,
		keepAliveTime:     keepAliveTime,
		maxRetries:        maxRetries,
		svcAddrRetryCount: svcAddrRetryCount,
	}, isDebugEnv, unTapServiceTimeout, throttler.MakeThrottler(svcAddrUpdateTimeout))

	resolver := makeFunctionReferenceResolver(fnStore)

	go serveMetric(logger)

	logger.Info("starting router", zap.Int("port", port))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serve(ctx, logger, port, tracingSamplingRate, triggers, resolver, displayAccessLog)
}
