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

// ShowWorkitemtypeNotFound test setup
func ShowWorkitemtypeNotFound(t *testing.T, ctrl app.WorkitemtypeController, id string) {
	ShowWorkitemtypeNotFoundCtx(t, context.Background(), ctrl, id)
}

// ShowWorkitemtypeNotFoundCtx test setup
func ShowWorkitemtypeNotFoundCtx(t *testing.T, ctx context.Context, ctrl app.WorkitemtypeController, id string) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/workitemtype/%v", id), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["id"] = []string{fmt.Sprintf("%v", id)}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "WorkitemtypeTest"), rw, req, prms)
	showCtx, err := app.NewShowWorkitemtypeContext(goaCtx, service)
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

// ShowWorkitemtypeOK test setup
func ShowWorkitemtypeOK(t *testing.T, ctrl app.WorkitemtypeController, id string) *app.WorkItemType {
	return ShowWorkitemtypeOKCtx(t, context.Background(), ctrl, id)
}

// ShowWorkitemtypeOKCtx test setup
func ShowWorkitemtypeOKCtx(t *testing.T, ctx context.Context, ctrl app.WorkitemtypeController, id string) *app.WorkItemType {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/workitemtype/%v", id), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["id"] = []string{fmt.Sprintf("%v", id)}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "WorkitemtypeTest"), rw, req, prms)
	showCtx, err := app.NewShowWorkitemtypeContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	a, ok := resp.(*app.WorkItemType)
	if !ok {
		t.Errorf("invalid response media: got %+v, expected instance of app.WorkItemType", resp)
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

// ShowWorkitemtypeOKLink test setup
func ShowWorkitemtypeOKLink(t *testing.T, ctrl app.WorkitemtypeController, id string) *app.WorkItemTypeLink {
	return ShowWorkitemtypeOKLinkCtx(t, context.Background(), ctrl, id)
}

// ShowWorkitemtypeOKLinkCtx test setup
func ShowWorkitemtypeOKLinkCtx(t *testing.T, ctx context.Context, ctrl app.WorkitemtypeController, id string) *app.WorkItemTypeLink {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/workitemtype/%v", id), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}
	prms["id"] = []string{fmt.Sprintf("%v", id)}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "WorkitemtypeTest"), rw, req, prms)
	showCtx, err := app.NewShowWorkitemtypeContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	a, ok := resp.(*app.WorkItemTypeLink)
	if !ok {
		t.Errorf("invalid response media: got %+v, expected instance of app.WorkItemTypeLink", resp)
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
