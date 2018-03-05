package metric

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	namespace = ""
	subsystem = "wit"
)

var (
	reqCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "Counter of requests received into the system.",
	}, []string{"method", "entity"})

	reqSuccessDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_successful_duration_seconds",
		Help:      "Bucketed histogram of processing time (s) of successfully completed requests, by method (GET/PUT etc.).",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 7),
	}, []string{"method"})
)

func init() {
	reqCnt = register(reqCnt, "requests_total").(*prometheus.CounterVec)
	reqSuccessDuration = register(reqSuccessDuration, "request_successful_duration_seconds").(*prometheus.HistogramVec)
}

func register(c prometheus.Collector, name string) prometheus.Collector {
	err := prometheus.Register(c)
	if err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector
		}
		log.Panic(nil, map[string]interface{}{
			"metric_name": prometheus.BuildFQName(namespace, subsystem, name),
			"err":         err,
		}, "failed to register the prometheus metric")
	}
	return c
}

func reportRequest(method, entity string) {
	if entity != "" && method != "" {
		reqCnt.WithLabelValues(method, entity).Inc()
	}
}

func reportRequestCompleted(method string, startTime time.Time) {
	if method != "" && !startTime.IsZero() {
		reqSuccessDuration.WithLabelValues(method).Observe(time.Since(startTime).Seconds())
	}
}
