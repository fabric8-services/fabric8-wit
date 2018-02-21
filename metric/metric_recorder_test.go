package metric

import (
	"testing"

	"github.com/goadesign/goa"
	dto "github.com/prometheus/client_model/go"
)

var (
	dummyEntity  = "dummy"
	testEntity   = "test"
	createAction = "create"
	listAction   = "list"
)

func TestMetricRecorder(t *testing.T) {
	svc := goa.New("metric")

	// for dummy entity, create=2 and list=1
	ctrl := svc.NewController("DummyController")
	ctx := goa.WithAction(ctrl.Context, createAction)
	recordMetric(ctx)
	ctx = goa.WithAction(ctrl.Context, listAction)
	recordMetric(ctx)
	ctx = goa.WithAction(ctrl.Context, createAction)
	recordMetric(ctx)

	// for test entity, create=1 and list=1
	ctrl = svc.NewController("TestController")
	ctx = goa.WithAction(ctrl.Context, createAction)
	recordMetric(ctx)
	ctx = goa.WithAction(ctrl.Context, listAction)
	recordMetric(ctx)

	check(t, dummyEntity, createAction, 2)
	check(t, dummyEntity, listAction, 1)
	check(t, testEntity, createAction, 1)
	check(t, testEntity, listAction, 1)
}

func check(t *testing.T, entity, action string, expected int64) {
	reqMetric, _ := reqCnt.GetMetricWithLabelValues(entity, action)
	metric := &dto.Metric{}
	reqMetric.Write(metric)
	actual := int64(metric.Counter.GetValue())
	if actual != expected {
		t.Errorf("metric(\"%s\", \"%s\"), want: %d, got: %d", entity, action, expected, actual)
	}
}
