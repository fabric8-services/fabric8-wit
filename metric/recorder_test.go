package metric

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/goadesign/goa"
	dto "github.com/prometheus/client_model/go"
)

var (
	dummyEntity = "dummy"
	testEntity  = "test"
	postMethod  = "POST"
	getMethod   = "GET"
)

func TestReqsTotalMetric(t *testing.T) {
	svc := goa.New("metric")

	// for dummy entity, POST=2 and GET=1
	ctrl := svc.NewController("DummyController")
	ctx, req := creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 409)
	recordReqsTotal(ctx, req)

	// for test entity, POST=1 and GET=1
	ctrl = svc.NewController("TestController")
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqsTotal(ctx, req)

	// validate
	check(t, postMethod, dummyEntity, "2xx", 2)
	check(t, getMethod, dummyEntity, "2xx", 1)
	check(t, postMethod, dummyEntity, "4xx", 1)
	check(t, postMethod, testEntity, "2xx", 1)
	check(t, getMethod, testEntity, "2xx", 1)
}

func TestReqDurationMetric(t *testing.T) {
	reqTimes := []time.Duration{101, 201, 401, 801, 1601, 3201, 6401, 12801}
	expectedBound := []float64{0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8}
	expectedCnt := []uint64{0, 1, 2, 3, 4, 5, 6, 7}

	svc := goa.New("metric")

	// add create action
	ctrl := svc.NewController("DummyController")
	ctx, req := creaeCtxAndReq(ctrl, postMethod, 201)
	for _, reqTime := range reqTimes {
		startTime := time.Now().Add(time.Millisecond * -reqTime)
		recordReqDuration(ctx, req, startTime)
	}

	// add list action to make sure that this should be filtered out
	ctrl = svc.NewController("DummyController")
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqDuration(ctx, req, time.Now())

	// validate
	reqMetric, _ := reqDuration.GetMetricWithLabelValues(methodVal(postMethod), dummyEntity, "2xx")
	m := &dto.Metric{}
	reqMetric.Write(m)
	if uint64(len(reqTimes)) != m.Histogram.GetSampleCount() {
		t.Errorf("Histogram count was incorrect, want: %d, got: %d",
			len(reqTimes), m.Histogram.GetSampleCount())
	}
	for ind, bucket := range m.Histogram.GetBucket() {
		if expectedBound[ind] != *bucket.UpperBound {
			t.Errorf("Bucket upper bound was incorrect, want: %f, got: %f\n",
				expectedBound[ind], *bucket.UpperBound)
		} else if expectedCnt[ind] != *bucket.CumulativeCount {
			t.Errorf("Bucket cumulative count was incorrect, want: %d, got: %d\n",
				expectedCnt[ind], *bucket.CumulativeCount)
		}
	}
}

func creaeCtxAndReq(ctrl *goa.Controller, method string, code int) (context.Context, *http.Request) {
	req := &http.Request{Host: "localhost", Method: method}
	rw := httptest.NewRecorder()
	rw.WriteHeader(code)
	ctx := goa.NewContext(ctrl.Context, rw, req, url.Values{})
	goa.ContextResponse(ctx).Status = code
	return ctx, req
}

func check(t *testing.T, method, entity, code string, expected int64) {
	method = methodVal(method)
	reqMetric, _ := reqCnt.GetMetricWithLabelValues(method, entity, code)
	m := &dto.Metric{}
	reqMetric.Write(m)
	actual := int64(m.Counter.GetValue())
	if actual != expected {
		t.Errorf("metric(\"%s\", \"%s\", \"%s\"), want: %d, got: %d", entity, method, code, expected, actual)
	}
}
