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

package router

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	executorClient "github.com/fission/fission/executor/client"
)

type functionHandler struct {
	fmap     *functionServiceMap
	executor *executorClient.Client
	function *metav1.ObjectMeta
}

// A layer on top of http.DefaultTransport, with retries.
type RetryingRoundTripper struct {
	maxRetries     int
	initialTimeout time.Duration
	funcHandler    *functionHandler
}

// RoundTrip is a custom transport with retries for http requests that forwards the request to the right serviceUrl, obtained
// from router's cache or from executor if router entry is stale.
//
// It first checks if the service address for this function came from router's cache.
// If it didn't, it makes a request to executor to get a new service for function. If that succeeds, it adds the address
// to it's cache and makes a request to that address with transport.RoundTrip call.
// Initial requests to new k8s services sometimes seem to fail, but retries work. So, it retries with an exponential
// back-off for maxRetries times.
//
// Else if it came from the cache, it makes a transport.RoundTrip with that cached address. If the response received is
// a network dial error (which means that the pod doesn't exist anymore), it removes the cache entry and makes a request
// to executor to get a new service for function. It then retries transport.RoundTrip with the new address.
//
// At any point in time, if the response received from transport.RoundTrip is other than dial network error, it is
// relayed as-is to the user, without any retries.
//
// While this RoundTripper handles the case where a previously cached address of the function pod isn't valid anymore
// (probably because the pod got deleted somehow), by making a request to executor to get a new service for this function,
// it doesn't handle a case where a newly specialized pod gets deleted just after the GetServiceForFunction succeeds.
// In such a case, the RoundTripper will retry requests against the new address and give up after maxRetries.
// However, the subsequent http call for this function will ensure the cache is invalidated.
//
// If GetServiceForFunction returns an error or if RoundTripper exits with an error, it get's translated into 502
// inside ServeHttp function of the reverseProxy.
// Earlier, GetServiceForFunction was called inside handler function and fission explicitly set http status code to 500
// if it returned an error.
func (roundTripper RetryingRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var needExecutor, serviceUrlFromExecutor bool
	var serviceUrl *url.URL

	// set the timeout for transport context
	timeout := roundTripper.initialTimeout
	transport := http.DefaultTransport.(*http.Transport)
	defer transport.CloseIdleConnections()

	// cache lookup to get serviceUrl
	serviceUrl, err = roundTripper.funcHandler.fmap.lookup(roundTripper.funcHandler.function)
	if err != nil || serviceUrl == nil {
		// cache miss or nil entry in cache
		needExecutor = true
	}

	for i := 0; i < roundTripper.maxRetries-1; i++ {
		if needExecutor {
			log.Printf("Calling getServiceForFunction for function: %s", roundTripper.funcHandler.function.Name)

			// send a request to executor to specialize a new pod
			service, err := roundTripper.funcHandler.executor.GetServiceForFunction(
				roundTripper.funcHandler.function)
			if err != nil {
				// We might want a specific error code or header for fission failures as opposed to
				// user function bugs.
				return nil, err
			}

			// parse the address into url
			serviceUrl, err = url.Parse(fmt.Sprintf("http://%v", service))
			if err != nil {
				return nil, err
			}

			// add the address in router's cache
			roundTripper.funcHandler.fmap.assign(roundTripper.funcHandler.function, serviceUrl)

			// flag denotes that service was not obtained from cache, instead, created just now by executor
			serviceUrlFromExecutor = true
		}

		// modify the request to reflect the service url
		// this service url may have come from the cache lookup or from executor response
		req.URL.Scheme = serviceUrl.Scheme
		req.URL.Host = serviceUrl.Host

		// To keep the function run container simple, it
		// doesn't do any routing.  In the future if we have
		// multiple functions per container, we could use the
		// function metadata here.
		// leave the query string intact (req.URL.RawQuery)
		req.URL.Path = "/"

		// Overwrite request host with internal host,
		// or request will be blocked in some situations
		// (e.g. istio-proxy)
		req.Host = serviceUrl.Host

		// over-riding default settings.
		transport.DialContext = (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext

		// forward the request to the function service
		resp, err = transport.RoundTrip(req)
		if err == nil {
			// if transport.RoundTrip succeeds and it was a cached entry, then tapService
			if !serviceUrlFromExecutor {
				go roundTripper.funcHandler.tapService(serviceUrl)
			}
			// return response back to user
			return resp, nil
		}

		// if transport.RoundTrip returns a non-network dial error, then relay it back to user
		if !fission.IsNetworkDialError(err) {
			return resp, err
		}

		// means its a newly created service and it returned a network dial error.
		// just retry after backing off for timeout period.
		if serviceUrlFromExecutor {
			log.Printf("request to %s errored out. backing off for %v before retrying",
				req.URL.Host, timeout)
			timeout *= time.Duration(2)
			time.Sleep(timeout)
			needExecutor = false
			continue
		} else {
			// if transport.RoundTrip returns a network dial error and serviceUrl was from cache,
			// it means, the entry in router cache is stale, so invalidate it.
			// also set needExecutor to true so a new service can be requested for function.
			log.Printf("request to %s errored out. removing function : %s from router's cache "+
				"and requesting a new service for function",
				req.URL.Host, roundTripper.funcHandler.function.Name)
			roundTripper.funcHandler.fmap.remove(roundTripper.funcHandler.function)
			needExecutor = true
		}
	}

	// finally, one more retry with the default timeout
	return http.DefaultTransport.RoundTrip(req)
}

func (fh *functionHandler) tapService(serviceUrl *url.URL) {
	if fh.executor == nil {
		return
	}
	fh.executor.TapService(serviceUrl)
}

func (fh *functionHandler) handler(responseWriter http.ResponseWriter, request *http.Request) {
	// retrieve url params and add them to request header
	vars := mux.Vars(request)
	for k, v := range vars {
		request.Header.Add(fmt.Sprintf("X-Fission-Params-%v", k), v)
	}

	// System Params
	MetadataToHeaders(HEADERS_FISSION_FUNCTION_PREFIX, fh.function, request)

	// TODO: As an optimization we may want to cache proxies too -- this might get us
	// connection reuse and possibly better performance
	director := func(req *http.Request) {
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &RetryingRoundTripper{
			initialTimeout: 50 * time.Millisecond,
			maxRetries:     10,
			funcHandler:    fh,
		},
	}

	proxy.ServeHTTP(responseWriter, request)
}
