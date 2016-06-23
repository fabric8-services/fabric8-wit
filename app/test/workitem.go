package test

import (
	"bytes"
	"fmt"
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// ShowWorkitemNotFound test setup
func ShowWorkitemNotFound(t *testing.T, ctrl app.WorkitemController, id string) {
	ShowWorkitemNotFoundCtx(t, context.Background(), ctrl, id)
}

// ShowWorkitemNotFoundCtx test setup
func ShowWorkitemNotFoundCtx(t *testing.T, ctx context.Context, ctrl app.WorkitemController, id string) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/workitem/%v", id), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["id"] = []string{fmt.Sprintf("%v", id)}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "WorkitemTest"), rw, req, prms)
	showCtx, err := app.NewShowWorkitemContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	if rw.Code != 404 {
		t.Errorf("invalid response status code: got %+v, expected 404", rw.Code)
	}

}

// ShowWorkitemOK test setup
func ShowWorkitemOK(t *testing.T, ctrl app.WorkitemController, id string) *app.WorkItem {
	return ShowWorkitemOKCtx(t, context.Background(), ctrl, id)
}

// ShowWorkitemOKCtx test setup
func ShowWorkitemOKCtx(t *testing.T, ctx context.Context, ctrl app.WorkitemController, id string) *app.WorkItem {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/workitem/%v", id), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["id"] = []string{fmt.Sprintf("%v", id)}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "WorkitemTest"), rw, req, prms)
	showCtx, err := app.NewShowWorkitemContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	a, ok := resp.(*app.WorkItem)
	if !ok {
		t.Errorf("invalid response media: got %+v, expected instance of app.WorkItem", resp)
	}

	if rw.Code != 200 {
		t.Errorf("invalid response status code: got %+v, expected 200", rw.Code)
	}

	err = a.Validate()
	if err != nil {
		t.Errorf("invalid response payload: got %v", err)
	}
	return a

}
