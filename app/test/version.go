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
	"testing"
)

// ShowVersionOK test setup
func ShowVersionOK(t *testing.T, ctrl app.VersionController) *app.Version {
	return ShowVersionOKCtx(t, context.Background(), ctrl)
}

// ShowVersionOKCtx test setup
func ShowVersionOKCtx(t *testing.T, ctx context.Context, ctrl app.VersionController) *app.Version {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/version"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	goaCtx := goa.NewContext(goa.WithAction(ctx, "VersionTest"), rw, req, nil)
	showCtx, err := app.NewShowVersionContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}
	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	a, ok := resp.(*app.Version)
	if !ok {
		t.Errorf("invalid response media: got %+v, expected instance of app.Version", resp)
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
