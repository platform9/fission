package fscache

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// funcname: the function's name
	// funcuid: the function's version id
	coldStarts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_cold_starts_total",
			Help: "How many cold starts are made by funcname, funcuid.",
		},
		[]string{"funcname", "funcuid"},
	)
	funcRunningSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_func_running_seconds_summary",
			Help:       "The running time (last access - create) in seconds of the function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"funcname", "funcuid"},
	)
	funcAliveSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_func_alive_seconds_summary",
			Help:       "The alive time in seconds of the function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"funcname", "funcuid"},
	)
	funcIsAlive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fission_func_is_alive",
			Help: "A binary value indicating is the funcname, funcuid alive",
		},
		[]string{"funcname", "funcuid"},
	)
)

func init() {
	// Register the function calls counter with Prometheus's default registry.
	prometheus.MustRegister(coldStarts)
	prometheus.MustRegister(funcRunningSummary)
	prometheus.MustRegister(funcAliveSummary)
	prometheus.MustRegister(funcIsAlive)
}

// IncreaseColdStarts increments the counter by 1.
func (fsc *FunctionServiceCache) IncreaseColdStarts(funcname, funcuid string) {
	coldStarts.WithLabelValues(funcname, funcuid).Inc()
}

func (fsc *FunctionServiceCache) observeFuncRunningTime(funcname, funcuid string, running float64) {
	funcRunningSummary.WithLabelValues(funcname, funcuid).Observe(running)
}

func (fsc *FunctionServiceCache) observeFuncAliveTime(funcname, funcuid string, alive float64) {
	funcAliveSummary.WithLabelValues(funcname, funcuid).Observe(alive)
}

func (fsc *FunctionServiceCache) setFuncAlive(funcname, funcuid string, isAlive bool) {
	count := 0
	if isAlive {
		count = 1
	}
	funcIsAlive.WithLabelValues(funcname, funcuid).Set(float64(count))
}
