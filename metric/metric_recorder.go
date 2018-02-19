package metric

import (
	"context"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/prometheus/client_golang/prometheus"
)

var reqLabels = []string{"entity", "action"}

// Recorder record metrics
func Recorder() goa.Middleware {
	reqCnt := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "wit_requests_total",
		Help: "Total number of WIT requests.",
	}, reqLabels)
	// TODO VN: Check err, need prometheus version change.
	prometheus.Register(reqCnt)

	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			err := h(ctx, rw, req)

			action := goa.ContextAction(ctx)
			ctrl := goa.ContextController(ctx)
			entity := ""
			if strings.HasSuffix(ctrl, "Controller") {
				entity = strings.TrimSuffix(ctrl, "Controller")
			}
			log.Debug(ctx, nil, "ctrl=%s, entity=%s, action=%s", ctrl, entity, action)

			if entity != "" {
				reqCnt.WithLabelValues(entity, action).Inc()
			}

			return err
		}
	}

}
