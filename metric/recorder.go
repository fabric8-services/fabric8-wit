package metric

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
)

// Recorder record metrics
func Recorder() goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			startTime := time.Now()
			err := h(ctx, rw, req)
			recordReqCnt(ctx, req)
			if err == nil {
				recordReqCompleted(ctx, req, startTime)
			}
			return err
		}
	}

}

func recordReqCnt(ctx context.Context, req *http.Request) {
	method := req.Method
	ctrl := goa.ContextController(ctx)
	entity := ""
	if strings.HasSuffix(ctrl, "Controller") {
		entity = strings.ToLower(strings.TrimSuffix(ctrl, "Controller"))
	}
	log.Debug(ctx, nil, "ctrl=%s, entity=%s, method=%s", ctrl, entity, method)

	if entity != "" {
		reportRequest(method, entity)
	}
}

func recordReqCompleted(ctx context.Context, req *http.Request, startTime time.Time) {
	method := req.Method
	reportRequestCompleted(method, startTime)
}
