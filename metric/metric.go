package metric

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/log"
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
		Buckets:   prometheus.ExponentialBuckets(0.05, 2, 8),
	}, reqLabels)

	resSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "response_size_bytes",
		Help:      "Bucketed histogram of the HTTP response sizes in bytes.",
		Buckets:   []float64{1000, 5000, 10000, 20000, 30000, 40000, 50000},
	}, reqLabels)

	reqSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "request_size_bytes",
		Help:      "Bucketed histogram of the HTTP request sizes in bytes.",
		Buckets:   []float64{1000, 5000, 10000, 20000, 30000, 40000, 50000},
	}, reqLabels)
)

func registerMetrics() {
	reqCnt = register(reqCnt, "requests_total").(*prometheus.CounterVec)
	reqDuration = register(reqDuration, "request_duration_seconds").(*prometheus.HistogramVec)
	resSize = register(resSize, "response_size_bytes").(*prometheus.HistogramVec)
	reqSize = register(reqSize, "request_size_bytes").(*prometheus.HistogramVec)
	log.Info(nil, nil, "metrics registered successfully")
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
	log.Debug(nil, map[string]interface{}{
		"metric_name": prometheus.BuildFQName(namespace, subsystem, name),
	}, "metric registered successfully")
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

func reportResponseSize(method, entity, code string, size int) {
	if method != "" && entity != "" && code != "" && size > 0 {
		resSize.WithLabelValues(method, entity, code).Observe(float64(size))
	}
}

func reportRequestSize(method, entity, code string, size int64) {
	if method != "" && entity != "" && code != "" && size > 0 {
		reqSize.WithLabelValues(method, entity, code).Observe(float64(size))
	}
}
