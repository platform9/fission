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

type chanRequest struct {
	request  *fission.CacheInvalidationRequest
	response chan *cacheInvalidationResponse
}

type cacheInvalidationResponse struct {
	serviceUrl *url.URL
	err        error
}

type functionHandler struct {
	fmap        *functionServiceMap
	executor    *executorClient.Client
	function    *metav1.ObjectMeta
	requestChan chan *chanRequest
	quitChan    chan bool
}

func (fh *functionHandler) getServiceForFunction() (*url.URL, error) {
	// call executor, get a url for a function
	svcName, err := fh.executor.GetServiceForFunction(fh.function)
	if err != nil {
		return nil, err
	}
	svcUrl, err := url.Parse(fmt.Sprintf("http://%v", svcName))
	if err != nil {
		return nil, err
	}
	return svcUrl, nil
}

// A layer on top of http.DefaultTransport, with retries.
type RetryingRoundTripper struct {
	maxRetries    int
	initalTimeout time.Duration
}

func (rrt RetryingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	timeout := rrt.initalTimeout
	transport := http.DefaultTransport.(*http.Transport)

	// Do max-1 retries; the last one uses default transport timeouts
	for i := rrt.maxRetries - 1; i > 0; i-- {
		// update timeout in transport
		transport.DialContext = (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext

		resp, err := transport.RoundTrip(req)
		if err == nil {
			return resp, nil
		}

		timeout *= time.Duration(2)
		log.Printf("Retrying request to %v in %v", req.URL.Host, timeout)
		time.Sleep(timeout)
	}

	// finally, one more retry with the default timeout
	return http.DefaultTransport.RoundTrip(req)
}

// TODO : Remove all debug logs before review.
// TODO : 1. Need clarification if starting a new goroutine to make http request to executor is the right thing to do. Does it look hacky?
// 	  2. executor client shared between multiple function handler objects? Is this intended?
func (fh functionHandler) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	// in order to stop the go-routine in any of the exit paths.
	defer func() {
		fh.quitChan <- true
	}()

	transport := http.DefaultTransport.(*http.Transport)

	// TODO : Fixed this to 2 because the entry for a function is cached for just 1 minute in router. there would be no point in retrying it more times. Is this ok?
	// 	  In any case, I'll remove the hard-coded value once I hear the opinion.
	maxRetries := 2
	for i := 0; i < maxRetries; i++ {
		resp, err = transport.RoundTrip(req)
		if err != nil {
			netErr, ok := err.(net.Error)
			if !ok {
				return resp, err
			}
			netOpErr, ok := netErr.(*net.OpError)
			if !ok {
				return resp, err
			}
			if netOpErr.Op == "dial" {
				// 1. invalidate router's fmap
				err = fh.fmap.remove(fh.function)
				if err != nil {
					log.Println("Unable to delete function from router cache," +
						"ignoring it for now.")
				}

				// 2. send invalidationRequest to executor.
				// This will basically invalidate executor's fsCache and make a request to specialize a new pod
				invalidationReq := &fission.CacheInvalidationRequest{
					FunctionMetadata:   fh.function,
					FunctionPodAddress: netOpErr.Addr.String(),
				}
				responseChan := make(chan *cacheInvalidationResponse)
				request := &chanRequest{
					request:  invalidationReq,
					response: responseChan,
				}

				fh.requestChan <- request
				response := <-request.response

				// 3. retry req until err is nil or max of 2 times.
				// The response will have the address of the newly specialized pod
				if response.err == nil {
					// make one more transport.RoundTrip.
					// This final call should be against the address of newly specialized pod, so changing the request to route it accordingly.
					req.URL.Host = response.serviceUrl.Host
					req.Host = response.serviceUrl.Host
					fh.fmap.assign(fh.function, response.serviceUrl)
					return transport.RoundTrip(req)
				}
			} else {
				return resp, err
			}
		} else {
			serviceUrl, _ := fh.fmap.lookup(fh.function)
			// if we're using our cache, asynchronously tell
			// executor we're using this service
			go fh.tapService(serviceUrl)
		}
	}

	return resp, err
}

func (fh *functionHandler) tapService(serviceUrl *url.URL) {
	if fh.executor == nil {
		return
	}
	fh.executor.TapService(serviceUrl)
}

// invalidateCacheService is spawned as a separate go-routine for each invocation of fh.handler.
// It waits on the requestChan for request to invalidate cache entry for a given function. On receiving a request, it makes a http post call to executor asking it to invalidate it's cache and to specialize a new pod.
// Once the function's route is served, it's useless to have this routine running, so it receives a signal to quit at which point it returns.
func (fh *functionHandler) invalidateCacheService() {
	for {
		select {
		case <-fh.quitChan:
			return

		case chanRequest := <-fh.requestChan:
			svcName, err := fh.executor.InvalidateCacheEntryForFunction(chanRequest.request)
			if err != nil {
				chanRequest.response <- &cacheInvalidationResponse{
					err: err,
				}
			}
			svcUrl, err := url.Parse(fmt.Sprintf("http://%v", svcName))
			if err != nil {
				chanRequest.response <- &cacheInvalidationResponse{
					err: err,
				}
			}
			chanRequest.response <- &cacheInvalidationResponse{
				serviceUrl: svcUrl,
				err:        err,
			}
		}
	}
}

func (fh *functionHandler) handler(responseWriter http.ResponseWriter, request *http.Request) {
	fh.quitChan = make(chan bool)
	go fh.invalidateCacheService()

	reqStartTime := time.Now()

	// retrieve url params and add them to request header
	vars := mux.Vars(request)
	for k, v := range vars {
		request.Header.Add(fmt.Sprintf("X-Fission-Params-%v", k), v)
	}

	// System Params
	MetadataToHeaders(HEADERS_FISSION_FUNCTION_PREFIX, fh.function, request)

	// cache lookup
	isCacheHit := false
	serviceUrl, err := fh.fmap.lookup(fh.function)
	if err != nil {
		// Cache miss: request the Pool Manager to make a new service.
		log.Printf("Not cached, getting new service for %v", fh.function)

		var poolErr error
		serviceUrl, poolErr = fh.getServiceForFunction()
		if poolErr != nil {
			log.Printf("Failed to get service for function %v: %v", fh.function.Name, poolErr)
			// We might want a specific error code or header for fission
			// failures as opposed to user function bugs.
			http.Error(responseWriter, "Internal server error (fission)", 500)
			return
		}

		// add it to the map
		fh.fmap.assign(fh.function, serviceUrl)
	} else {
		isCacheHit = true
	}

	// Proxy off our request to the serviceUrl, and send the response back.
	// TODO: As an optimization we may want to cache proxies too -- this might get us
	// connection reuse and possibly better performance
	director := func(req *http.Request) {
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

		// leave the query string intact (req.URL.RawQuery)

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	var transport http.RoundTripper
	if isCacheHit {
		// Use the default transport associated with this function handler object.
		// only in case of a cache hit.
		transport = *fh
	} else {
		// Initial requests to new k8s services sometimes seem to fail, but retries work.
		// So use a transport that does retries in case of a cache miss
		transport = RetryingRoundTripper{
			maxRetries:    10,
			initalTimeout: 50 * time.Millisecond,
		}
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: transport,
	}
	delay := time.Since(reqStartTime)
	if delay > 100*time.Millisecond {
		log.Printf("Request delay for %v: %v", serviceUrl, delay)
	}
	proxy.ServeHTTP(responseWriter, request)
}
