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

	executorClient "github.com/fission/fission/executor/client"
)

type functionHandler struct {
	fmap         *functionServiceMap
	executor     *executorClient.Client
	function     *metav1.ObjectMeta
	roundTripper *RetryingRoundTripper
}

// A layer on top of http.DefaultTransport, with retries.
type RetryingRoundTripper struct {
	maxRetries    int
	initalTimeout time.Duration
}

func isNetDialError(err error) bool {
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}
	netOpErr, ok := netErr.(*net.OpError)
	if !ok {
		return false
	}
	if netOpErr.Op == "dial" {
		return true
	}
	return false
}

// RoundTrip is a custom transport for http requests
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
// Such a case can be handled by retrying requests for newly returned address for 5 times perhaps and then setting
// isNewService flag to false. That'll automatically take care of requesting new service for function.
// But 5 is just a heuristic, what if newly created services sometimes takes more than 5 retries to be available?
//
// If GetServiceForFunction returns an error or if RoundTripper exits with an error, it get's translated into 502
// inside ServeHttp function of the reverseProxy.
// Earlier, GetServiceForFunction was called inside handler function and fission explicitly set http status code to 500
// if it returned an error.
// There is a plan in the future to differentiate fission failures from user bugs and we can do so by creating a custom
// modifyResponse function which is called by the ServeHttp function of the reverseProxy.
func (fh functionHandler) RoundTrip(req *http.Request) (resp *http.Response, roundTripErr error) {
	transport := http.DefaultTransport.(*http.Transport)
	defer transport.CloseIdleConnections()

	timeout := fh.roundTripper.initalTimeout
	isNewService := false

	for i := 0; i < fh.roundTripper.maxRetries; i++ {
		if len(req.Header.Get("X-Custom-CacheMiss")) > 0 || (roundTripErr != nil && !isNewService) {
			log.Printf("Calling getServiceForFunction for function: %s", fh.function.Name)
			reqStartTime := time.Now()

			// send a request to  executor to specialize a new pod
			serviceUrl, err := fh.executor.GetServiceForFunction(fh.function)
			if err != nil {
				// We might want a specific error code or header for fission failures as opposed to
				// user function bugs.
				return nil, err
			}

			// measure cold start
			delay := time.Since(reqStartTime)
			if delay > 100*time.Millisecond {
				log.Printf("Request delay for %v: %v", fh.function.Name, delay)
			}

			// modify the request to reflect the new service address
			req.URL.Scheme = serviceUrl.Scheme
			req.URL.Host = serviceUrl.Host
			req.URL.Path = "/"
			req.Host = serviceUrl.Host

			// add the address in router's cache
			fh.fmap.assign(fh.function, serviceUrl)
			isNewService = true
			if header := req.Header.Get("X-Custom-CacheMiss"); len(header) > 0 {
				req.Header.Del("X-Custom-CacheMiss")
			}
		}

		// over-riding default settings.
		transport.DialContext = (&net.Dialer{
			Timeout:   fh.roundTripper.initalTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext

		// make the actual call with the request
		resp, roundTripErr = transport.RoundTrip(req)
		if roundTripErr != nil {
			if isNewService {
				// if it's a newly created service and transport.RoundTrip returns error,
				// retry without invalidating cache after backing off for timeout period.
				log.Printf("request to %s errored out. backing off for %v before retrying",
					req.URL.Host, timeout)
				timeout *= time.Duration(2)
				time.Sleep(timeout)
				continue
			} else if isNetDialError(roundTripErr) {
				// if transport.RoundTrip returns a network dial error and service wasn't newly created,
				// it means, the entry in router cache is stale, so invalidate it.
				log.Printf("request to %s errored out. removing function:%s from router's cache"+
					"and requesting for new service for function",
					req.URL.Host, fh.function.Name)
				fh.fmap.remove(fh.function)
			} else if !isNetDialError(roundTripErr) {
				// if transport.RoundTrip returns a non-network dial error, then relay it back to user
				return resp, roundTripErr
			}
		} else {
			// if transport.RoundTrip succeeds and it was a cached entry, then tapService
			if !isNewService {
				serviceUrl, _ := fh.fmap.lookup(fh.function)
				// if we're using our cache, asynchronously tell
				// executor we're using this service
				go fh.tapService(serviceUrl)
			}
			// return response back to user
			return resp, nil
		}
	}

	// after maxRetries, if transport.RoundTrip still receives an error response, just give up and return error.
	log.Printf("Giving up serving function : %s after retrying for %v times", fh.function.Name,
		fh.roundTripper.maxRetries)
	return resp, roundTripErr
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

	// Proxy off our request to the serviceUrl, and send the response back.
	// TODO: As an optimization we may want to cache proxies too -- this might get us
	// connection reuse and possibly better performance
	director := func(req *http.Request) {
		serviceUrl, err := fh.fmap.lookup(fh.function)
		if err == nil {
			log.Printf("Proxying request for %v to %v", req.URL, serviceUrl.Host)
			// send this request to serviceurl
			req.URL.Scheme = serviceUrl.Scheme
			req.URL.Host = serviceUrl.Host

			// To keep the function run container simple, it
			// doesn't do any routing.  In the future if we have
			// multiple functions per container, we could use the
			// function metadata here.
			req.URL.Path = "/"

			// Overwrite request host with internal host,
			// or request will be blocked in some situations
			// (e.g. istio-proxy)
			req.Host = serviceUrl.Host
		} else {
			// adding a custom header to request as a way to let RoundTripper know that the address
			// wasn't cached
			req.Header.Add("X-Custom-CacheMiss", "miss")
		}

		// leave the query string intact (req.URL.RawQuery)

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: *fh, // fh implements RoundTrip method of http.RoundTripper interface
	}

	proxy.ServeHTTP(responseWriter, request)
}
