package metric

import (
	"context"
	"net/http"
	"strconv"
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
			recordReqsTotal(ctx, req)
			recordReqDuration(ctx, req, startTime)
			return err
		}
	}
}

func recordReqsTotal(ctx context.Context, req *http.Request) {
	reportRequestsTotal(labelsVal(ctx, req))
}

func recordReqDuration(ctx context.Context, req *http.Request, startTime time.Time) {
	method, entity, code := labelsVal(ctx, req)
	reportRequestDuration(method, entity, code, startTime)
}

func labelsVal(ctx context.Context, req *http.Request) (method, entity, code string) {
	method = methodVal(req.Method)
	ctrl := goa.ContextController(ctx)
	entity = entityVal(ctrl)
	status := goa.ContextResponse(ctx).Status
	code = codeVal(status)
	log.Debug(ctx, nil, "method=%s, ctrl=%s, entity=%s, status=%s, code=%s",
		method, ctrl, entity, status, code)
	return method, entity, code
}

func methodVal(method string) string {
	return strings.ToLower(method)
}

// ctrl=SpaceController -> entity=space
func entityVal(ctrl string) string {
	entity := ""
	if strings.HasSuffix(ctrl, "Controller") {
		entity = strings.ToLower(strings.TrimSuffix(ctrl, "Controller"))
	}
	return entity
}

// Group HTTP status code in the form of 2xx, 3xx etc.
func codeVal(status int) string {
	code := (status - (status % 100)) / 100
	return strconv.Itoa(code) + "xx"
}
