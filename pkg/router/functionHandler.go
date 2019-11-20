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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	fv1 "github.com/fission/fission/pkg/apis/fission.io/v1"
	"github.com/fission/fission/pkg/crd"
	ferror "github.com/fission/fission/pkg/error"
	"github.com/fission/fission/pkg/error/network"
	executorClient "github.com/fission/fission/pkg/executor/client"
	"github.com/fission/fission/pkg/throttler"
	"github.com/fission/fission/pkg/types"
)

const (
	FORWARDED        = "Forwarded"
	X_FORWARDED_HOST = "X-Forwarded-Host"
)

type (
	functionHandler struct {
		logger                   *zap.Logger
		fmap                     *functionServiceMap
		executor                 *executorClient.Client
		function                 *metav1.ObjectMeta
		httpTrigger              *fv1.HTTPTrigger
		functionMetadataMap      map[string]*metav1.ObjectMeta
		fnWeightDistributionList []FunctionWeightDistribution
		tsRoundTripperParams     *tsRoundTripperParams
		isDebugEnv               bool
		svcAddrUpdateThrottler   *throttler.Throttler
		functionTimeoutMap       map[k8stypes.UID]int
	}

	tsRoundTripperParams struct {
		timeout          time.Duration
		timeoutExponent  int
		disableKeepAlive bool
		keepAliveTime    time.Duration

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

	// A layer on top of http.DefaultTransport, with retries.
	RetryingRoundTripper struct {
		logger           *zap.Logger
		funcHandler      *functionHandler
		funcTimeout      time.Duration
		closeContextFunc *context.CancelFunc
	}

	// To keep the request body open during retries, we create an interface with Close operation being a no-op.
	// Details : https://github.com/flynn/flynn/pull/875
	fakeCloseReadCloser struct {
		io.ReadCloser
	}

	svcEntryRecord struct {
		svcUrl    *url.URL
		fromCache bool
	}
)

func init() {
	// just seeding the random number for getting the canary function
	rand.Seed(time.Now().UnixNano())
}

func (w *fakeCloseReadCloser) Close() error {
	return nil
}

func (w *fakeCloseReadCloser) RealClose() error {
	if w.ReadCloser == nil {
		return nil
	}
	return w.ReadCloser.Close()
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
func (roundTripper *RetryingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set forwarded host header if not exists
	roundTripper.addForwardedHostHeader(req)

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
	transport := roundTripper.getDefaultTransport()
	ocRoundTripper := &ochttp.Transport{Base: transport}

	executingTimeout := roundTripper.funcHandler.tsRoundTripperParams.timeout

	// wrap the req.Body with another ReadCloser interface.
	if req.Body != nil {
		req.Body = &fakeCloseReadCloser{req.Body}
	}

	// close req body
	defer func() {
		if req.Body != nil {
			req.Body.(*fakeCloseReadCloser).RealClose()
		}
	}()

	roundTripper.logger.Debug("request headers", zap.Any("headers", req.Header))

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
	var retryCounter int

	var serviceUrl *url.URL
	var serviceUrlFromCache bool
	var err error

	var resp *http.Response

	for i := 0; i < roundTripper.funcHandler.tsRoundTripperParams.maxRetries; i++ {
		// set service url of target service of request only when
		// trying to get new service url from cache/executor.
		if retryCounter == 0 {
			// get function service url from cache or executor
			serviceUrl, serviceUrlFromCache, err = roundTripper.funcHandler.getServiceEntry()
			if err != nil {
				// We might want a specific error code or header for fission failures as opposed to
				// user function bugs.
				statusCode, errMsg := ferror.GetHTTPError(err)
				if roundTripper.funcHandler.isDebugEnv {
					return &http.Response{
						StatusCode:    statusCode,
						Proto:         req.Proto,
						ProtoMajor:    req.ProtoMajor,
						ProtoMinor:    req.ProtoMinor,
						Body:          ioutil.NopCloser(bytes.NewBufferString(errMsg)),
						ContentLength: int64(len(errMsg)),
						Request:       req,
						Header:        make(http.Header),
					}, nil
				}
				return nil, ferror.MakeError(http.StatusInternalServerError, err.Error())
			}

			// service url maybe nil if router cannot find one in cache,
			// so here we retry to get service url again
			if serviceUrl == nil {
				time.Sleep(executingTimeout)
				executingTimeout = executingTimeout * time.Duration(roundTripper.funcHandler.tsRoundTripperParams.timeoutExponent)
				continue
			}

			// tapService before invoking roundTrip for the serviceUrl
			if serviceUrlFromCache {
				go roundTripper.funcHandler.tapService(serviceUrl)
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
		}

		// over-riding default settings.
		transport.DialContext = (&net.Dialer{
			Timeout:   executingTimeout,
			KeepAlive: roundTripper.funcHandler.tsRoundTripperParams.keepAliveTime,
		}).DialContext

		overhead := time.Since(startTime)

		// Do NOT assign returned request to "req"
		// because the request used in the last round
		// will be canceled when calling setContext.
		newReq := roundTripper.setContext(req)

		// forward the request to the function service
		resp, err = ocRoundTripper.RoundTrip(newReq)

		if err == nil {
			// Track metrics
			httpMetricLabels.code = resp.StatusCode
			funcMetricLabels.cached = serviceUrlFromCache

			go functionCallCompleted(funcMetricLabels, httpMetricLabels,
				overhead, time.Since(startTime), resp.ContentLength)

			// return response back to user
			return resp, nil
		} else if i >= roundTripper.funcHandler.tsRoundTripperParams.maxRetries-1 {
			// return here if we are in the last round
			roundTripper.logger.Error("error getting response from function",
				zap.String("function_name", fnMeta.Name),
				zap.Error(err))
			return nil, err
		}

		// if transport.RoundTrip returns a non-network dial error, then relay it back to user
		netErr := network.Adapter(err)

		// dial timeout or dial network errors goes here
		var isNetDialErr, isNetTimeoutErr bool
		if netErr != nil {
			isNetDialErr = netErr.IsDialError()
			isNetTimeoutErr = netErr.IsTimeoutError()
		}

		// if transport.RoundTrip returns a non-network dial error (e.g. "context canceled"), then relay it back to user
		if !isNetDialErr {
			return resp, err
		}

		// Check whether an error is an timeout error ("dial tcp i/o timeout").
		// If it's not a timeout error or retryCounter exceeded pre-defined threshold,
		// we assume the entry in router cache is stale, invalidate it.
		if !isNetTimeoutErr || retryCounter >= roundTripper.funcHandler.tsRoundTripperParams.svcAddrRetryCount {
			if serviceUrlFromCache {
				// if transport.RoundTrip returns a network dial error and serviceUrl was from cache,
				// it means, the entry in router cache is stale, so invalidate it.
				roundTripper.logger.Debug("request errored out - removing function from router's cache and requesting a new service for function",
					zap.String("url", req.URL.Host),
					zap.String("function_name", fnMeta.Name),
					zap.Error(err))

				roundTripper.funcHandler.fmap.remove(fnMeta)
			}
			retryCounter = 0
		} else {
			roundTripper.logger.Debug("request errored out - backing off before retrying",
				zap.String("url", req.URL.Host),
				zap.String("function_name", fnMeta.Name),
				zap.Error(err))
			retryCounter++
		}

		roundTripper.logger.Debug("Backing off before retrying", zap.Any("backoff_time", executingTimeout), zap.Error(err))
		time.Sleep(executingTimeout)
		executingTimeout = executingTimeout * time.Duration(roundTripper.funcHandler.tsRoundTripperParams.timeoutExponent)

		// close response body before entering next loop
		if resp != nil {
			resp.Body.Close()
		}
	}

	e := errors.New("Unable to get service url for connection")
	roundTripper.logger.Error(e.Error(), zap.String("function_name", fnMeta.Name))
	return nil, e
}

// getDefaultTransport returns a pointer to new copy of http.Transport object to prevent
// the value of http.DefaultTransport from being changed by goroutines.
func (roundTripper RetryingRoundTripper) getDefaultTransport() *http.Transport {
	// The transport setup here follows the configurations of http.DefaultTransport
	// but without Dialer since we will change it later.
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Default disables caching, Please refer to issue and specifically comment:
		// https://github.com/fission/fission/issues/723#issuecomment-398781995
		// You can change it by setting environment variable "ROUTER_ROUND_TRIP_DISABLE_KEEP_ALIVE"
		// of router or helm variable "disableKeepAlive" before installation to false.
		DisableKeepAlives: roundTripper.funcHandler.tsRoundTripperParams.disableKeepAlive,
	}
}

// setContext returns a shallow copy of request with a new timeout context.
func (roundTripper *RetryingRoundTripper) setContext(req *http.Request) *http.Request {
	if roundTripper.closeContextFunc != nil {
		(*roundTripper.closeContextFunc)()
	}
	// pass request context as parent context for the case
	// that user aborts connection before timeout. Otherwise,
	// the request won't be canceled until the deadline exceeded
	// which may be a potential security issue.
	ctx, closeCtx := context.WithTimeout(req.Context(), roundTripper.funcTimeout)
	roundTripper.closeContextFunc = &closeCtx

	return req.WithContext(ctx)
}

// closeContext closes the context to release resources.
func (roundTripper *RetryingRoundTripper) closeContext() {
	if roundTripper.closeContextFunc != nil {
		(*roundTripper.closeContextFunc)()
	}
}

func (fh *functionHandler) tapService(serviceUrl *url.URL) {
	if fh.executor == nil {
		return
	}
	fh.executor.TapService(serviceUrl)
}

func (fh functionHandler) handler(responseWriter http.ResponseWriter, request *http.Request) {
	if fh.httpTrigger != nil && fh.httpTrigger.Spec.FunctionReference.Type == types.FunctionReferenceTypeFunctionWeights {
		// canary deployment. need to determine the function to send request to now
		fnMetadata := getCanaryBackend(fh.functionMetadataMap, fh.fnWeightDistributionList)
		if fnMetadata == nil {
			fh.logger.Error("could not get canary backend",
				zap.Any("metadataMap", fh.functionMetadataMap),
				zap.Any("distributionList", fh.fnWeightDistributionList))
			// TODO : write error to responseWrite and return response
			return
		}
		fh.function = fnMetadata
		fh.logger.Debug("chosen function backend's metadata", zap.Any("metadata", fh.function))
	}

	// url path
	setPathInfoToHeader(request)

	// system params
	setFunctionMetadataToHeader(fh.function, request)

	director := func(req *http.Request) {
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	fnTimeout := fh.functionTimeoutMap[fh.function.GetUID()]
	if fnTimeout == 0 {
		fnTimeout = fv1.DEFAULT_FUNCTION_TIMEOUT
	}

	rrt := &RetryingRoundTripper{
		logger:      fh.logger.Named("roundtripper"),
		funcHandler: &fh,
		funcTimeout: time.Duration(fnTimeout) * time.Second,
	}

	proxy := &httputil.ReverseProxy{
		Director:     director,
		Transport:    rrt,
		ErrorHandler: getProxyErrorHandler(fh.logger, fh.function),
	}

	defer func() {
		// If the context is closed when RoundTrip returns, client may receive
		// truncated response body due to "context canceled" error. To avoid
		// this, we need to close request context after proxy.ServeHTTP finished.
		//
		// NOTE: rrt.closeContext() must be put in the defer function; otherwise,
		// reverseProxy may panic when failed to write response and the context
		// will not be closed.
		//
		// ref: https://github.com/golang/go/issues/28239
		rrt.closeContext()
	}()

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

// getProxyErrorHandler returns a reverse proxy error handler
func getProxyErrorHandler(logger *zap.Logger, fnMeta *metav1.ObjectMeta) func(rw http.ResponseWriter, req *http.Request, err error) {
	return func(rw http.ResponseWriter, req *http.Request, err error) {
		status := http.StatusBadGateway
		switch err {
		case context.Canceled:
			// 499 CLIENT CLOSED REQUEST
			// A non-standard status code introduced by nginx for the case
			// when a client closes the connection while nginx is processing the request.
			// Reference: https://httpstatuses.com/499
			status = 499
			logger.Debug("client closes the connection",
				zap.Any("function", fnMeta), zap.Any("request_header", req.Header))
		case context.DeadlineExceeded:
			status = http.StatusGatewayTimeout
			logger.Error("function not responses before the timeout",
				zap.Any("function", fnMeta), zap.Any("request_header", req.Header))
		default:
			logger.Error("error sending request to function",
				zap.Error(err), zap.Any("function", fnMeta), zap.Any("request_header", req.Header))
		}
		// TODO: return error message that contains traceable UUID back to user. Issue #693
		rw.WriteHeader(status)
	}
}

// addForwardedHostHeader add "forwarded host" to request header
func (roundTripper RetryingRoundTripper) addForwardedHostHeader(req *http.Request) {
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
		roundTripper.logger.Error("error parsing request url while adding forwarded host headers",
			zap.Error(err),
			zap.String("url", reqUrl))
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

// getServiceEntry is a short-hand for developers to get service url entry that may returns from executor or cache
func (fh *functionHandler) getServiceEntry() (serviceUrl *url.URL, serviceUrlFromCache bool, err error) {
	// try to find service url from cache first
	serviceUrl, err = fh.getServiceEntryFromCache()
	if err == nil && serviceUrl != nil {
		return serviceUrl, true, nil
	} else if err != nil {
		return nil, false, err
	}

	// cache miss or nil entry in cache
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use throttle to limit the total amount of requests sent
	// to the executor to prevent it from overloaded.
	recordObj, err := fh.svcAddrUpdateThrottler.RunOnce(
		crd.CacheKey(fh.function),
		func(firstToTheLock bool) (interface{}, error) {
			var u *url.URL
			// Get service entry from executor and update cache if its the first goroutine
			if firstToTheLock { // first to the service url
				fh.logger.Debug("calling getServiceForFunction",
					zap.String("function_name", fh.function.Name))
				u, err = fh.getServiceEntryFromExecutor(ctx)
				if err != nil {
					fh.logger.Error("error getting service url from executor",
						zap.Error(err),
						zap.String("function_name", fh.function.Name))
					return nil, err
				}
				// add the address in router's cache
				fh.logger.Info("assigning service url for function",
					zap.String("url", u.String()),
					zap.String("function_name", fh.function.Name))
				fh.fmap.assign(fh.function, u)
			} else {
				u, err = fh.getServiceEntryFromCache()
				if err != nil {
					return nil, err
				}
			}

			return svcEntryRecord{
				svcUrl:    u,
				fromCache: firstToTheLock,
			}, err
		},
	)
	if err != nil {
		e := "error updating service address entry for function"
		fh.logger.Error(e,
			zap.Error(err),
			zap.String("function_name", fh.function.Name),
			zap.String("function_namespace", fh.function.Namespace))
		return nil, false, errors.Wrapf(err, "%s %s_%s", e, fh.function.Name, fh.function.Namespace)
	}

	record, ok := recordObj.(svcEntryRecord)
	if !ok {
		return nil, false, errors.Errorf("Received unknown service record type")
	}

	return record.svcUrl, record.fromCache, nil
}

// getServiceEntryFromCache returns service url entry returns from cache
func (fh *functionHandler) getServiceEntryFromCache() (serviceUrl *url.URL, err error) {
	// cache lookup to get serviceUrl
	serviceUrl, err = fh.fmap.lookup(fh.function)
	if err != nil {
		var errMsg string

		e, ok := err.(ferror.Error)
		if !ok {
			errMsg = fmt.Sprintf("Unknown error when looking up service entry: %v", err)
		} else {
			// Ignore ErrorNotFound error here, it's an expected error,
			// roundTripper will try to get service url later.
			if e.Code == ferror.ErrorNotFound {
				return nil, nil
			}
			errMsg = fmt.Sprintf("Error getting function %v;s service entry from cache: %v", fh.function.Name, err)
		}
		return nil, ferror.MakeError(http.StatusInternalServerError, errMsg)
	}
	return serviceUrl, nil
}

// getServiceEntryFromExecutor returns service url entry returns from executor
func (fh *functionHandler) getServiceEntryFromExecutor(ctx context.Context) (*url.URL, error) {
	// send a request to executor to specialize a new pod
	service, err := fh.executor.GetServiceForFunction(ctx, fh.function)
	if err != nil {
		statusCode, errMsg := ferror.GetHTTPError(err)
		fh.logger.Error("error from GetServiceForFunction",
			zap.Error(err),
			zap.String("error_message", errMsg),
			zap.Any("function", fh.function),
			zap.Int("status_code", statusCode))
		return nil, err
	}

	// parse the address into url
	serviceUrl, err := url.Parse(fmt.Sprintf("http://%v", service))
	if err != nil {
		fh.logger.Error("error parsing service url",
			zap.Error(err),
			zap.String("service_url", serviceUrl.String()))
		return nil, err
	}

	return serviceUrl, nil
}
