package metric

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	namespace = ""
	subsystem = "service"
)

var (
	reqLabels = []string{"method", "entity", "code"}

	reqCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "Counter of requests received into the system.",
	}, reqLabels)

	reqDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help:      "Bucketed histogram of processing time (s) of requests.",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 8),
	}, reqLabels)
)

func init() {
	reqCnt = register(reqCnt, "requests_total").(*prometheus.CounterVec)
	reqDuration = register(reqDuration, "request_duration_seconds").(*prometheus.HistogramVec)
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

func reportRequestsTotal(method, entity, code string) {
	if method != "" && entity != "" && code != "" {
		reqCnt.WithLabelValues(method, entity, code).Inc()
	}
}

func reportRequestDuration(method, entity, code string, startTime time.Time) {
	if method != "" && entity != "" && code != "" && !startTime.IsZero() {
		reqDuration.WithLabelValues(method, entity, code).Observe(time.Since(startTime).Seconds())
	}
}
