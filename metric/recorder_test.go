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
	dummyCtrl  = "DummyController"
	testCtrl   = "TestController"
	postMethod = "POST"
	getMethod  = "GET"
)

func TestReqsTotalMetric(t *testing.T) {
	svc := goa.New("metric")

	// for dummy entity, POST=3 and GET=1
	ctrl := svc.NewController(dummyCtrl)
	ctx, req := creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 409)
	recordReqsTotal(ctx, req)

	// for test entity, POST=1 and GET=1
	ctrl = svc.NewController(testCtrl)
	ctx, req = creaeCtxAndReq(ctrl, postMethod, 201)
	recordReqsTotal(ctx, req)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqsTotal(ctx, req)

	// validate
	checkCounter(t, postMethod, dummyCtrl, "2xx", 2)
	checkCounter(t, getMethod, dummyCtrl, "2xx", 1)
	checkCounter(t, postMethod, dummyCtrl, "4xx", 1)
	checkCounter(t, postMethod, testCtrl, "2xx", 1)
	checkCounter(t, getMethod, testCtrl, "2xx", 1)
}

func TestReqDurationMetric(t *testing.T) {
	reqTimes := []time.Duration{101, 201, 401, 801, 1601, 3201, 6401, 12801}
	expectedBound := []float64{0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8}
	expectedCnt := []uint64{0, 1, 2, 3, 4, 5, 6, 7}

	svc := goa.New("metric")

	// add post method
	ctrl := svc.NewController(dummyCtrl)
	ctx, req := creaeCtxAndReq(ctrl, postMethod, 201)
	for _, reqTime := range reqTimes {
		startTime := time.Now().Add(time.Millisecond * -reqTime)
		recordReqDuration(ctx, req, startTime)
	}

	// add get method to make sure that this should be filtered out
	ctrl = svc.NewController(dummyCtrl)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	recordReqDuration(ctx, req, time.Now())

	// validate
	reqMetric, _ := reqDuration.GetMetricWithLabelValues(methodVal(postMethod), entityVal(dummyCtrl), "2xx")
	m := &dto.Metric{}
	reqMetric.Write(m)
	checkHistogram(t, m, uint64(len(reqTimes)), expectedBound, expectedCnt)
}

func TestResSizeMetric(t *testing.T) {
	resSizes := []int{1001, 5001, 10001, 20001, 30001, 40001, 50001}
	expectedBound := []float64{1000, 5000, 10000, 20000, 30000, 40000, 50000}
	expectedCnt := []uint64{0, 1, 2, 3, 4, 5, 6}

	svc := goa.New("metric")

	// add get method for dummy entity
	ctrl := svc.NewController(dummyCtrl)
	ctx, req := creaeCtxAndReq(ctrl, getMethod, 200)
	for _, size := range resSizes {
		goa.ContextResponse(ctx).Length = size
		recordResSize(ctx, req)
	}

	// add get method for test entity to make sure that this should be filtered out
	ctrl = svc.NewController(testCtrl)
	ctx, req = creaeCtxAndReq(ctrl, getMethod, 200)
	goa.ContextResponse(ctx).Length = 1000
	recordResSize(ctx, req)

	// validate
	reqMetric, _ := resSize.GetMetricWithLabelValues(methodVal(getMethod), entityVal(dummyCtrl), "2xx")
	m := &dto.Metric{}
	reqMetric.Write(m)
	checkHistogram(t, m, uint64(len(resSizes)), expectedBound, expectedCnt)
}

func TestMethodVal(t *testing.T) {
	tables := []struct {
		in, out string
	}{
		{"GET", "get"},
		{"POST", "post"},
	}

	for _, table := range tables {
		actual := methodVal(table.in)
		if table.out != actual {
			t.Errorf("output was incorrect, want:%s, got:%s", table.out, actual)
		}
	}
}

func TestEntityVal(t *testing.T) {
	tables := []struct {
		in, out string
	}{
		{"TestController", "test"},
		{"Anycontroller", ""},
		{"Dummy", ""},
	}

	for _, table := range tables {
		actual := entityVal(table.in)
		if table.out != actual {
			t.Errorf("output was incorrect, want:%s, got:%s", table.out, actual)
		}
	}
}

func TestCodeVal(t *testing.T) {
	tables := []struct {
		in  int
		out string
	}{
		{100, "1xx"},
		{200, "2xx"},
		{201, "2xx"},
		{404, "4xx"},
	}

	for _, table := range tables {
		actual := codeVal(table.in)
		if table.out != actual {
			t.Errorf("output was incorrect, want:%s, got:%s", table.out, actual)
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

func checkCounter(t *testing.T, method, entity, code string, expected int64) {
	method = methodVal(method)
	entity = entityVal(entity)
	reqMetric, _ := reqCnt.GetMetricWithLabelValues(method, entity, code)
	m := &dto.Metric{}
	reqMetric.Write(m)
	actual := int64(m.Counter.GetValue())
	if actual != expected {
		t.Errorf("metric(\"%s\", \"%s\", \"%s\"), want: %d, got: %d", entity, method, code, expected, actual)
	}
}

func checkHistogram(t *testing.T, m *dto.Metric, expectedCount uint64, expectedBound []float64, expectedCnt []uint64) {
	if expectedCount != m.Histogram.GetSampleCount() {
		t.Errorf("Histogram count was incorrect, want: %d, got: %d",
			expectedCount, m.Histogram.GetSampleCount())
	}
	for ind, bucket := range m.Histogram.GetBucket() {
		if expectedBound[ind] != *bucket.UpperBound {
			t.Errorf("Bucket upper bound was incorrect, want: %f, got: %f\n",
				expectedBound[ind], *bucket.UpperBound)
		}
		if expectedCnt[ind] != *bucket.CumulativeCount {
			t.Errorf("Bucket cumulative count was incorrect, want: %d, got: %d\n",
				expectedCnt[ind], *bucket.CumulativeCount)
		}
	}
}
