package metric

import (
	"context"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reqLabels = []string{"entity", "action"}
	reqCnt    = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "wit_requests_total",
		Help: "Total number of WIT requests.",
	}, reqLabels)
)

func init() {
	err := prometheus.Register(reqCnt)
	if err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			reqCnt = are.ExistingCollector.(*prometheus.CounterVec)
		} else {
			log.Panic(nil, map[string]interface{}{
				"metric_name": "wit_requests_total",
				"err":         err,
			}, "failed to register the prometheus metric")
		}
	}
}

// Recorder record metrics
func Recorder() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			err := h(ctx, rw, req)
			recordMetric(ctx)
			return err
		}
	}

}

func recordMetric(ctx context.Context) {
	action := goa.ContextAction(ctx)
	ctrl := goa.ContextController(ctx)
	entity := ""
	if strings.HasSuffix(ctrl, "Controller") {
		entity = strings.ToLower(strings.TrimSuffix(ctrl, "Controller"))
	}
	log.Debug(ctx, nil, "ctrl=%s, entity=%s, action=%s", ctrl, entity, action)

	if entity != "" {
		reqCnt.WithLabelValues(entity, action).Inc()
	}
}
