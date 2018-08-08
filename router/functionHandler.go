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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	executorClient "github.com/fission/fission/executor/client"
	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	"github.com/fission/fission/redis"
)

type tsRoundTripperParams struct {
	timeout         time.Duration
	timeoutExponent int
	keepAlive       time.Duration
	maxRetries      int
}

type functionHandler struct {
	fmap                 *functionServiceMap
	frmap                *functionRecorderMap
	trmap                *triggerRecorderMap
	executor             *executorClient.Client
	function             *metav1.ObjectMeta
	httpTrigger          *crd.HTTPTrigger
	tsRoundTripperParams *tsRoundTripperParams
	recorderName         string
}

// A layer on top of http.DefaultTransport, with retries.
type RetryingRoundTripper struct {
	funcHandler *functionHandler
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

	// TODO: Keep? --> Needed for queries encoded in URL before they're stripped by the proxy
	var originalUrl url.URL
	originalUrl = *req.URL

	// Iff this request needs to be recorded, we save the body
	var postedBody string
	if len(roundTripper.funcHandler.recorderName) > 0 {
		if req.ContentLength > 0 {
			p := make([]byte, req.ContentLength)
			buf, _ := ioutil.ReadAll(req.Body)
			// We need two io readers because a single reader will drain the buffer, hence we keep a replacement copy
			rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
			rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))

			rdr1.Read(p)
			postedBody = string(p)
			logrus.Info(fmt.Sprintf("%v", postedBody))
			req.Body = rdr2
		}
	}

	// Metrics stuff
	startTime := time.Now()
	funcMetricLabels := &functionLabels{
		namespace: roundTripper.funcHandler.function.Namespace,
		name:      roundTripper.funcHandler.function.Name,
	}
	httpMetricLabels := &httpLabels{
		method: req.Method,
	}
	if roundTripper.funcHandler.httpTrigger != nil {
		httpMetricLabels.host = roundTripper.funcHandler.httpTrigger.Spec.Host
		httpMetricLabels.path = roundTripper.funcHandler.httpTrigger.Spec.RelativeURL
	}

	// set the timeout for transport context
	transport := http.DefaultTransport.(*http.Transport)
	// Disables caching, Please refer to issue and specifically comment: https://github.com/fission/fission/issues/723#issuecomment-398781995
	transport.DisableKeepAlives = true

	// cache lookup to get serviceUrl
	serviceUrl, err = roundTripper.funcHandler.fmap.lookup(roundTripper.funcHandler.function)
	if err != nil || serviceUrl == nil {
		// cache miss or nil entry in cache
		needExecutor = true
	}

	executingTimeout := roundTripper.funcHandler.tsRoundTripperParams.timeout

	for i := 0; i < roundTripper.funcHandler.tsRoundTripperParams.maxRetries-1; i++ {
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
			Timeout:   executingTimeout,
			KeepAlive: roundTripper.funcHandler.tsRoundTripperParams.keepAlive,
		}).DialContext

		overhead := time.Since(startTime)

		// tapService before invoking roundTrip for the serviceUrl
		if !serviceUrlFromExecutor {
			go roundTripper.funcHandler.tapService(serviceUrl)
		}

		// forward the request to the function service
		resp, err = transport.RoundTrip(req)
		if err == nil {
			// Track metrics
			httpMetricLabels.code = resp.StatusCode
			funcMetricLabels.cached = !serviceUrlFromExecutor

			functionCallCompleted(funcMetricLabels, httpMetricLabels,
				overhead, time.Since(startTime), resp.ContentLength)

			// if transport.RoundTrip succeeds and it was a cached entry, then tapService
			if !serviceUrlFromExecutor {
				go roundTripper.funcHandler.tapService(serviceUrl)
			}

			trigger := ""
			if roundTripper.funcHandler.httpTrigger != nil {
				trigger = roundTripper.funcHandler.httpTrigger.Metadata.Name
			} else {
				log.Println("No trigger attached.") // Wording?
			}

			if len(roundTripper.funcHandler.recorderName) > 0 {
				redis.Record(
					trigger,
					roundTripper.funcHandler.recorderName,
					req.Header.Get("X-Fission-ReqUID"), req, originalUrl, postedBody, resp, roundTripper.funcHandler.function.Namespace,
					time.Now().UnixNano(),
				)
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
				req.URL.Host, executingTimeout)

			executingTimeout = executingTimeout * time.Duration(roundTripper.funcHandler.tsRoundTripperParams.timeoutExponent)
			time.Sleep(executingTimeout)

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

	var reqUID string
	if len(fh.recorderName) > 0 {
		UID := strings.ToLower(uuid.NewV4().String())
		reqUID = "REQ" + UID
		request.Header.Add("X-Fission-ReqUID", reqUID)
		log.Print("Record request with ReqUID: ", reqUID)
	}

	// system params
	MetadataToHeaders(HEADERS_FISSION_FUNCTION_PREFIX, fh.function, request)

	director := func(req *http.Request) {
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		Transport: &RetryingRoundTripper{
			funcHandler: fh,
		},
	}

	proxy.ServeHTTP(responseWriter, request)
}
