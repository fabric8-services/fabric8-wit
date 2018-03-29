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

// Recorder record prometheus metrics related to http request and response.
func Recorder() goa.Middleware {
	registerMetrics()

	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			startTime := time.Now()
			err := h(ctx, rw, req)

			// record metrics
			method, entity, code := labelsVal(ctx)
			recordReqsTotal(method, entity, code)
			recordReqSize(method, entity, code, req)
			recordResSize(method, entity, code, goa.ContextResponse(ctx))
			recordReqDuration(method, entity, code, startTime)

			return err
		}
	}
}

func recordReqsTotal(method, entity, code string) {
	reportRequestsTotal(method, entity, code)
}

func recordReqSize(method, entity, code string, req *http.Request) {
	size := computeApproximateRequestSize(req)
	reportRequestSize(method, entity, code, size)
}

func recordResSize(method, entity, code string, res *goa.ResponseData) {
	size := res.Length
	reportResponseSize(method, entity, code, size)
}

func recordReqDuration(method, entity, code string, startTime time.Time) {
	reportRequestDuration(method, entity, code, startTime)
}

func labelsVal(ctx context.Context) (method, entity, code string) {
	method = methodVal(goa.ContextRequest(ctx).Method)
	ctrl := goa.ContextController(ctx)
	entity = entityVal(ctrl)
	status := goa.ContextResponse(ctx).Status
	code = codeVal(status)
	log.Debug(ctx, nil, "method=%s, ctrl=%s, entity=%s, status=%d, code=%s",
		method, ctrl, entity, status, code)
	return method, entity, code
}

func methodVal(method string) string {
	return strings.ToLower(method)
}

// ctrl=SpaceController -> entity=space
func entityVal(ctrl string) (entity string) {
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

// TODO http2 request not supported
func computeApproximateRequestSize(r *http.Request) int64 {
	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}
	s += len(r.Method)
	s += len(r.Proto)
	s += len(r.Host)

	hs := 0
	for name, values := range r.Header {
		hs += len(name)
		for _, value := range values {
			hs += len(value)
		}
	}
	s += hs

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	var size int64
	size = int64(s)
	if r.ContentLength != -1 {
		size += r.ContentLength
	}
	log.Debug(nil, map[string]interface{}{
		"header_size": hs, "body_size": r.ContentLength, "req_size": size},
		"compute request size")
	return size
}
