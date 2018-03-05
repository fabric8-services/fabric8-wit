package metric

import (
	"net/http"
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

func TestReqCntMetric(t *testing.T) {
	svc := goa.New("metric")
	params := url.Values{}

	// for dummy entity, POST=2 and GET=1
	ctrl := svc.NewController("DummyController")

	req := &http.Request{Host: "localhost", Method: postMethod}
	ctx := goa.NewContext(ctrl.Context, nil, req, params)
	recordReqCnt(ctx, req)

	req = &http.Request{Host: "localhost", Method: getMethod}
	ctx = goa.NewContext(ctrl.Context, nil, req, params)
	recordReqCnt(ctx, req)

	req = &http.Request{Host: "localhost", Method: postMethod}
	ctx = goa.NewContext(ctrl.Context, nil, req, params)
	recordReqCnt(ctx, req)

	// for test entity, POST=1 and GET=1
	ctrl = svc.NewController("TestController")

	req = &http.Request{Host: "localhost", Method: postMethod}
	ctx = goa.NewContext(ctrl.Context, nil, req, params)
	recordReqCnt(ctx, req)

	req = &http.Request{Host: "localhost", Method: getMethod}
	ctx = goa.NewContext(ctrl.Context, nil, req, params)
	recordReqCnt(ctx, req)

	check(t, postMethod, dummyEntity, 2)
	check(t, getMethod, dummyEntity, 1)
	check(t, postMethod, testEntity, 1)
	check(t, getMethod, testEntity, 1)
}

func TestReqSuccessDurationMetric(t *testing.T) {
	reqTimes := []time.Duration{101, 201, 401, 801, 1601, 3201, 6401}
	expectedBound := []float64{0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4}
	expectedCnt := []uint64{0, 1, 2, 3, 4, 5, 6}

	svc := goa.New("metric")
	params := url.Values{}

	// add create action
	ctrl := svc.NewController("DummyController")
	req := &http.Request{Host: "localhost", Method: postMethod}
	ctx := goa.NewContext(ctrl.Context, nil, req, params)
	for _, reqTime := range reqTimes {
		startTime := time.Now().Add(time.Millisecond * -reqTime)
		recordReqCompleted(ctx, req, startTime)
	}

	// add list action to make sure that this should be filtered out
	ctrl = svc.NewController("DummyController")
	req = &http.Request{Host: "localhost", Method: getMethod}
	recordReqCompleted(ctx, req, time.Now())

	// validate
	reqMetric, _ := reqSuccessDuration.GetMetricWithLabelValues(postMethod)
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

func check(t *testing.T, method, entity string, expected int64) {
	reqMetric, _ := reqCnt.GetMetricWithLabelValues(method, entity)
	m := &dto.Metric{}
	reqMetric.Write(m)
	actual := int64(m.Counter.GetValue())
	if actual != expected {
		t.Errorf("metric(\"%s\", \"%s\"), want: %d, got: %d", entity, method, expected, actual)
	}
}
