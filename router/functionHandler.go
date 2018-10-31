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
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/fission/fission"
	"github.com/fission/fission/crd"
	executorClient "github.com/fission/fission/executor/client"
	"github.com/fission/fission/redis"
)

const (
	FORWARDED        = "Forwarded"
	X_FORWARDED_HOST = "X-Forwarded-Host"
)

type tsRoundTripperParams struct {
	timeout         time.Duration
	timeoutExponent int
	keepAlive       time.Duration

	// maxRetires is the max times for RetryingRoundTripper to retry a request.
	// Default maxRetries is 10, which means router will retry for
	// up to 10 times and abort it if still not succeeded.
	maxRetries int

	// svcAddrRetryCount is the max times for RetryingRoundTripper to retry with a specific service address
	// Router sends requests to a specific service address for each function.
	// A service address is considered as an invalid one if amount of non-network
	// errors router received is higher than svcAddrRetryCount. In this situation,
	// remove it from cache and try to get a new one from executor.
	// Default svcAddrRetryCount is 5.
	svcAddrRetryCount int
}

type functionHandler struct {
	fmap                     *functionServiceMap
	frmap                    *functionRecorderMap
	trmap                    *triggerRecorderMap
	executor                 *executorClient.Client
	function                 *metav1.ObjectMeta
	httpTrigger              *crd.HTTPTrigger
	functionMetadataMap      map[string]*metav1.ObjectMeta
	fnWeightDistributionList []FunctionWeightDistribution
	tsRoundTripperParams     *tsRoundTripperParams
	recorderName             string
	isDebugEnv               bool
	svcAddrUpdateLocks       *svcAddrUpdateLocks
}

// A layer on top of http.DefaultTransport, with retries.
type RetryingRoundTripper struct {
	funcHandler *functionHandler
}

func init() {
	// just seeding the random number for getting the canary function
	rand.Seed(time.Now().UnixNano())
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
	var serviceUrlFromCache bool
	var serviceUrl *url.URL
	var retryCounter int

	// Set forwarded host header if not exists
	addForwardedHostHeader(req)

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
			log.Info(fmt.Sprintf("%v", postedBody))
			req.Body = rdr2
		}
	}

	fnMeta := roundTripper.funcHandler.function

	// Metrics stuff
	startTime := time.Now()
	funcMetricLabels := &functionLabels{
		namespace: fnMeta.Namespace,
		name:      fnMeta.Name,
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

	executingTimeout := roundTripper.funcHandler.tsRoundTripperParams.timeout

	var reqBody []byte
	if req.Body != nil {
		reqBody, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; i < roundTripper.funcHandler.tsRoundTripperParams.maxRetries-1; i++ {

		// cache lookup to get serviceUrl
		serviceUrl, err = roundTripper.funcHandler.fmap.lookup(fnMeta)
		if err != nil {
			e, ok := err.(fission.Error)
			if (ok && e.Code != fission.ErrorNotFound) || !ok {
				if ok {
					err = errors.Wrap(err, fmt.Sprintf("Error getting function %v;s service entry from cache", fnMeta.Name))
				} else {
					err = errors.Wrap(err, "Unknown error when looking up service entry")
				}
				return nil, err
			}
		} else {
			serviceUrlFromCache = true
		}

		if serviceUrl != nil {
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
			if serviceUrlFromCache {
				go roundTripper.funcHandler.tapService(serviceUrl)
			}

			// reset the body before every retry
			req.Body = ioutil.NopCloser(bytes.NewReader(reqBody))

			// forward the request to the function service
			resp, err = transport.RoundTrip(req)
			if err == nil {
				// Track metrics
				httpMetricLabels.code = resp.StatusCode
				funcMetricLabels.cached = serviceUrlFromCache

				functionCallCompleted(funcMetricLabels, httpMetricLabels,
					overhead, time.Since(startTime), resp.ContentLength)

				if len(roundTripper.funcHandler.recorderName) > 0 {
					if roundTripper.funcHandler.httpTrigger != nil {
						trigger := roundTripper.funcHandler.httpTrigger.Metadata.Name
						redis.Record(
							trigger,
							roundTripper.funcHandler.recorderName,
							req.Header.Get("X-Fission-ReqUID"), req, originalUrl, postedBody, resp, fnMeta.Namespace,
							time.Now().UnixNano(),
						)
					} else {
						log.Println("No http trigger attached for recorder: %v", roundTripper.funcHandler.recorderName)
					}
				}

				// return response back to user
				return resp, nil
			}

			// if transport.RoundTrip returns a non-network dial error, then relay it back to user
			if !fission.IsNetworkDialError(err) {
				err = errors.Wrapf(err, "Error sending request to function %v", fnMeta.Name)
				return resp, err
			}

			// dial timeout or dial network errors goes here

			// The reason for request failure may vary from case to case.
			// After some investigation, found most of the failure are due to
			// network timeout or target function is under heavy workload. In
			// such cases, if router keeps trying to get new function service
			// will increase executor burden and cause 502 error.
			//
			// The "retryCounter" was introduced to solve this problem by retrying
			// requests for "limited threshold". Once a request's retryCounter higher
			// than the predefined threshold, reset retryCounter and remove service
			// cache, then retry to get new svc record from executor again.
			if retryCounter < roundTripper.funcHandler.tsRoundTripperParams.svcAddrRetryCount {
				retryCounter++

				executingTimeout = executingTimeout * time.Duration(roundTripper.funcHandler.tsRoundTripperParams.timeoutExponent)

				log.Printf("request to %s errored out. backing off for %v before retrying",
					req.URL.Host, executingTimeout)

				time.Sleep(executingTimeout)

				if serviceUrlFromCache {
					continue
				}
			} else {
				// if transport.RoundTrip returns a network dial error and serviceUrl was from cache,
				// it means, the entry in router cache is stale, so invalidate it.
				log.Printf("request to %s errored out. removing function : %s from router's cache "+
					"and requesting a new service for function",
					req.URL.Host, fnMeta.Name)
				roundTripper.funcHandler.fmap.remove(fnMeta)
				retryCounter = 0
			}
		}

		// break directly if we still fail at the last round
		if i >= roundTripper.funcHandler.tsRoundTripperParams.maxRetries-1 {
			break
		}

		// cache miss or nil entry in cache
		lock, ableToUpdateCache := roundTripper.funcHandler.grabUpdateEntryLock(fnMeta)

		if !ableToUpdateCache {
			// This goroutine wait for update of service map to finish.
			err = lock.Wait()
			if err != nil {
				log.Println(errors.Wrap(err,
					fmt.Sprintf("Error updating service address entry for function %v_%v", fnMeta.Name, fnMeta.Namespace)))
			}
		} else {
			// This goroutine is the first one to grab update lock

			log.Printf("Calling getServiceForFunction for function: %s", fnMeta.Name)

			// send a request to executor to specialize a new pod
			service, err := roundTripper.funcHandler.executor.GetServiceForFunction(
				roundTripper.funcHandler.function)

			if err != nil {
				statusCode, errMsg := fission.GetHTTPError(err)
				log.Printf("Err from GetServiceForFunction for function (%v): %v : %v", roundTripper.funcHandler.function, statusCode, errMsg)

				// We might want a specific error code or header for fission failures as opposed to
				// user function bugs.
				if roundTripper.funcHandler.isDebugEnv {
					return &http.Response{
						StatusCode:    statusCode,
						Proto:         req.Proto,
						ProtoMajor:    req.ProtoMajor,
						ProtoMinor:    req.ProtoMinor,
						Body:          ioutil.NopCloser(bytes.NewBufferString(errMsg)),
						ContentLength: int64(len(errMsg)),
						Request:       req,
						Header:        make(http.Header, 0),
					}, nil
				}

				roundTripper.funcHandler.releaseUpdateEntryLock(fnMeta)
				return nil, err
			}

			// parse the address into url
			serviceUrl, err = url.Parse(fmt.Sprintf("http://%v", service))
			if err != nil {
				log.Printf("Error parsing service url (%v): %v", serviceUrl, err)
				roundTripper.funcHandler.releaseUpdateEntryLock(fnMeta)
				return nil, err
			}

			// add the address in router's cache
			log.Printf("Assigning serviceUrl : %s for function : %s", serviceUrl, roundTripper.funcHandler.function.Name)
			roundTripper.funcHandler.fmap.assign(roundTripper.funcHandler.function, serviceUrl)

			// flag denotes that service was not obtained from cache, instead, created just now by executor
			serviceUrlFromCache = false

			roundTripper.funcHandler.releaseUpdateEntryLock(fnMeta)
		}
	}

	// finally, one more retry with the default timeout
	resp, err = http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Printf("Error getting response from function %v: %v",
			fnMeta.Name, err)
	}

	return resp, err
}

func (fh *functionHandler) tapService(serviceUrl *url.URL) {
	if fh.executor == nil {
		return
	}
	fh.executor.TapService(serviceUrl)
}

func (fh functionHandler) handler(responseWriter http.ResponseWriter, request *http.Request) {
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

	if fh.httpTrigger != nil && fh.httpTrigger.Spec.FunctionReference.Type == fission.FunctionReferenceTypeFunctionWeights {
		// canary deployment. need to determine the function to send request to now
		fnMetadata := getCanaryBackend(fh.functionMetadataMap, fh.fnWeightDistributionList)
		if fnMetadata == nil {
			log.Printf("Error getting canary backend ")
			// TODO : write error to responseWrite and return response
			return
		}
		fh.function = fnMetadata
		log.Debugf("chosen fnBackend's metadata : %+v", fh.function)
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
			funcHandler: &fh,
		},
	}

	proxy.ServeHTTP(responseWriter, request)
}

// findCeil picks a function from the functionWeightDistribution list based on the
// random number generated. It uses the prefix calculated for the function weights.
func findCeil(randomNumber int, wtDistrList []FunctionWeightDistribution) string {
	low := 0
	high := len(wtDistrList) - 1

	for {
		if low >= high {
			break
		}

		mid := low + high/2
		if randomNumber >= wtDistrList[mid].sumPrefix {
			low = mid + 1
		} else {
			high = mid
		}
	}

	if wtDistrList[low].sumPrefix >= randomNumber {
		return wtDistrList[low].name
	} else {
		return ""
	}
}

// picks a function to route to based on a random number generated
func getCanaryBackend(fnMetadatamap map[string]*metav1.ObjectMeta, fnWtDistributionList []FunctionWeightDistribution) *metav1.ObjectMeta {
	randomNumber := rand.Intn(fnWtDistributionList[len(fnWtDistributionList)-1].sumPrefix + 1)

	fnName := findCeil(randomNumber, fnWtDistributionList)

	return fnMetadatamap[fnName]
}

// addForwardedHostHeader add "forwarded host" to request header
func addForwardedHostHeader(req *http.Request) {
	// for more detailed information, please visit:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Forwarded

	if len(req.Header.Get(FORWARDED)) > 0 || len(req.Header.Get(X_FORWARDED_HOST)) > 0 {
		// forwarded headers were set by external proxy, leave them intact
		return
	}

	// Format of req.Host is <host>:<port>
	// We need to extract hostname from it, than
	// check whether a host is ipv4 or ipv6 or FQDN
	reqUrl := fmt.Sprintf("%s://%s", req.Proto, req.Host)
	u, err := url.Parse(reqUrl)
	if err != nil {
		log.Printf("Error parsing request url (%v): %v", reqUrl, err)
		return
	}

	var host string

	// ip will be nil if the Hostname is a FQDN string
	ip := net.ParseIP(u.Hostname())

	// ip == nil -> hostname is FQDN instead of ip address
	// The order of To4() and To16() here matters, To16() will
	// converts an IPv4 address to IPv6 format address and may
	// cause router append wrong host value to header. To prevent
	// this we need to check whether To4() is nil first.
	if ip == nil || (ip != nil && ip.To4() != nil) {
		host = fmt.Sprintf(`host=%s;`, req.Host)
	} else if ip != nil && ip.To16() != nil {
		// For the "Forwarded" header, if a host is an IPv6 address it should be quoted
		host = fmt.Sprintf(`host="%s";`, req.Host)
	}

	req.Header.Set(FORWARDED, host)
	req.Header.Set(X_FORWARDED_HOST, req.Host)
}

// grabUpdateEntryLock helps goroutine to grab update lock for updating function service cache.
// If the update lock exists, return old svcAddrUpdateLock.
func (fh *functionHandler) grabUpdateEntryLock(fnMeta *metav1.ObjectMeta) (lock *svcAddrUpdateLock, ableToUpdateCache bool) {
	return fh.svcAddrUpdateLocks.Get(fnMeta)
}

// releaseUpdateEntryLock release update lock so that other goroutines can take over the responsibility
// of updating the service map.
func (fh *functionHandler) releaseUpdateEntryLock(fnMeta *metav1.ObjectMeta) {
	fh.svcAddrUpdateLocks.Delete(fnMeta)
}
