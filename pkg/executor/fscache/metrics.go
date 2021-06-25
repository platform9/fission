package fscache

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labelsStrings = []string{"funcname", "funcuid", "cached"}
	// funcname: the function's name
	// funcuid: the function's version id
	coldStarts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fission_cold_starts_total",
			Help: "How many cold starts are made by funcname, funcuid.",
		},
		labelsStrings,
	)
	funcRunningSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_func_running_seconds_summary",
			Help:       "The running time (last access - create) in seconds of the function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)
	funcAliveSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_func_alive_seconds_summary",
			Help:       "The alive time in seconds of the function.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)
	funcIsAlive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fission_func_is_alive",
			Help: "A binary value indicating is the funcname, funcuid alive",
		},
		labelsStrings,
	)
	funcChoosePod = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_func_choose_pod",
			Help:       "Time taken to choose a pod, applicable to PoolManager only",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		labelsStrings,
	)

	funcSpecialisePod = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "fission_func_specialise_pod",
			Help: "Time taken to specialise a pod",
		},
		labelsStrings,
	)
	funcReapTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_pod_reaptime_seconds",
			Help:       "Amount of seconds to reap a pod",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"funcname", "funcaddress"},
	)
	idleTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "fission_idle_pod_time",
			Help:       "Number of seconds it took for Reaper to detect the pod was idle",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"funcname", "funcaddress"},
	)
)

func init() {
	// Register the function calls counter with Prometheus's default registry.
	prometheus.MustRegister(coldStarts)
	prometheus.MustRegister(funcRunningSummary)
	prometheus.MustRegister(funcAliveSummary)
	prometheus.MustRegister(funcIsAlive)
	prometheus.MustRegister(funcReapTime)
	prometheus.MustRegister(idleTime)
	prometheus.MustRegister(funcSpecialisePod)
	prometheus.MustRegister(funcChoosePod)
}

// IncreaseColdStarts increments the counter by 1.
func (fsc *FunctionServiceCache) IncreaseColdStarts(funcname, funcuid string, cached bool) {
	coldStarts.WithLabelValues(funcname, funcuid, strconv.FormatBool(cached)).Inc()
}

func (fsc *FunctionServiceCache) observeFuncRunningTime(funcname, funcuid string, running float64, cached bool) {
	funcRunningSummary.WithLabelValues(funcname, funcuid, strconv.FormatBool(cached)).Observe(running)
}

func (fsc *FunctionServiceCache) observeFuncAliveTime(funcname, funcuid string, alive float64, cached bool) {
	funcAliveSummary.WithLabelValues(funcname, funcuid, strconv.FormatBool(cached)).Observe(alive)
}

// ObserveChoosePodTime observes the time taken to choose a pod
func (fsc *FunctionServiceCache) ObserveChoosePodTime(envName, envuid string, duration time.Duration) {
	funcChoosePod.WithLabelValues(envName, envuid).Observe(float64(duration.Nanoseconds()) / 1e9)
}

// ObserveSpecialisePodTime observes the time taken to choose a pod
func (fsc *FunctionServiceCache) ObserveSpecialisePodTime(envName, envuid string, err bool, duration time.Duration) {
	funcSpecialisePod.WithLabelValues(envName, envuid, strconv.FormatBool(err)).Observe(float64(duration.Nanoseconds()) / 1e9)
}
func (fsc *FunctionServiceCache) setFuncAlive(funcname, funcuid string, isAlive bool, cached bool) {
	count := 0
	if isAlive {
		count = 1
	}
	funcIsAlive.WithLabelValues(funcname, funcuid, strconv.FormatBool(cached)).Set(float64(count))
}

// ReapTime is the amount of time taken to reap a pod
func (fsc *FunctionServiceCache) ReapTime(funcName, funcAddress string, time float64) {
	funcReapTime.WithLabelValues(funcName, funcAddress).Observe(float64(time))
}

// IdleTime is the amount of time it took Reaper to find out the pod was idle
func (fsc *FunctionServiceCache) IdleTime(funcName, funcAddress string, time float64) {
	idleTime.WithLabelValues(funcName, funcAddress).Observe(time)
}
