package router

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var globalFunctionCallCount uint64

type (
	// functionLabels is the set of metrics labels that relate to
	// functions.
	//
	// cached indicates whether or not the function call hit the
	// cache in this service.
	//
	// namespace and name are the metadata of the function.
	functionLabels struct {
		cached    bool
		namespace string
		name      string
	}

	// httpLabels is the set of metrics labels that relate to HTTP
	// requests.
	//
	// host is the host that the HTTP request was made to
	// path is the relative URL of the request
	// method is the HTTP method ("GET", "POST", ...)
	// code is the HTTP status code
	httpLabels struct {
		host   string
		path   string
		method string
		code   int
	}
)

var (
	metricAddr = ":8080"

	// function + http labels as strings
	labelsStrings = []string{"namespace", "name", "host", "path", "method", "code", "funcuid"}

	labelsStringsForIncomingRequests = []string{"namespace", "name", "host", "path", "method", "funcuid"}

	// Function http calls count
	// cached: true | false, is this function service address cached locally
	// namespace: function namespace
	// name: function name
	// code: http status code
	// path: the client call the function on which http path
	// method: the function's http method
	functionCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_function_calls_total",
			Help: "Count of Fission function calls",
		},
		labelsStrings,
	)
	functionCallErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_function_errors_total",
			Help: "Count of Fission function errors",
		},
		labelsStrings,
	)
	functionCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fission_function_duration_seconds",
			Help: "Runtime duration of the Fission function.",
		},
		labelsStrings,
	)
	functionCallOverhead = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fission_function_overhead_seconds",
			Help: "The function call delay caused by fission.",
		},
		labelsStrings,
	)
	functionCallResponseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_function_response_size_bytes",
			Help:       "The response size of the http call to target function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)

	requestsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_requests_received",
			Help: "Incoming number of requests at Router before being processed",
		},
		labelsStringsForIncomingRequests,
	)

	serviceURLFetch = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fission_service_fetch_time",
			Help: "Time taken for Router to get the service URL",
		},
		[]string{"error"},
	)
)

func init() {
	prometheus.MustRegister(functionCalls)
	prometheus.MustRegister(functionCallErrors)
	prometheus.MustRegister(functionCallDuration)
	prometheus.MustRegister(functionCallOverhead)
	prometheus.MustRegister(functionCallResponseSize)
	prometheus.MustRegister(requestsReceived)
	prometheus.MustRegister(serviceURLFetch)
}

func labelsToStrings(f *functionLabels, h *httpLabels, funcuid string) []string {
	return []string{
		f.namespace,
		f.name,
		h.host,
		h.path,
		h.method,
		fmt.Sprint(h.code),
		funcuid,
	}
}

func labelsToStringsBeforeProcessing(f *functionLabels, h *httpLabels, funcuid string) []string {
	return []string{
		f.namespace,
		f.name,
		h.host,
		h.path,
		h.method,
		funcuid,
	}
}

func incrementRequest(f *functionLabels, h *httpLabels, funcuid string) {

	l := labelsToStringsBeforeProcessing(f, h, funcuid)

	// requestsReceived is the number of requests incoming before they're processed
	requestsReceived.WithLabelValues(l...).Inc()

}

func functionCallCompleted(f *functionLabels, h *httpLabels, overhead, duration time.Duration, respSize int64, funcuid string) {
	atomic.AddUint64(&globalFunctionCallCount, 1)

	l := labelsToStrings(f, h, funcuid)

	// overhead: time from request ingress into router up to proxing into function pod
	functionCallOverhead.WithLabelValues(l...).Observe(float64(overhead.Nanoseconds()) / 1e9)

	// total function call counter
	functionCalls.WithLabelValues(l...).Inc()

	// error counter
	if h.code >= 400 {
		functionCallErrors.WithLabelValues(l...).Inc()
	}

	// duration summary
	functionCallDuration.WithLabelValues(l...).Observe(float64(duration.Nanoseconds()) / 1e9)

	// Response size.  -1 means the size unknown, in which case we don't report it.
	if respSize != -1 {
		functionCallResponseSize.WithLabelValues(l...).Observe(float64(respSize))
	}
}

func observeServiceURLFetchTime(duration time.Duration, err bool) {
	serviceURLFetch.WithLabelValues(strconv.FormatBool(err)).Observe(float64(duration.Nanoseconds()) / 1e9)
}
