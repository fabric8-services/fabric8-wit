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

// ShowVersionOK Show runs the method Show of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func ShowVersionOK(t *testing.T, ctrl app.VersionController) (http.ResponseWriter, *app.Version) {
	return ShowVersionOKWithContext(t, context.Background(), ctrl)
}

// ShowVersionOKWithContext Show runs the method Show of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func ShowVersionOKWithContext(t *testing.T, ctx context.Context, ctrl app.VersionController) (http.ResponseWriter, *app.Version) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/version"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "VersionTest"), rw, req, prms)
	showCtx, err := app.NewShowVersionContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Show(showCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}
	if rw.Code != 200 {
		t.Errorf("invalid response status code: got %+v, expected 200", rw.Code)
	}
	var mt *app.Version
	if resp != nil {
		var ok bool
		mt, ok = resp.(*app.Version)
		if !ok {
			t.Errorf("invalid response media: got %+v, expected instance of app.Version", resp)
		}
		err = mt.Validate()
		if err != nil {
			t.Errorf("invalid response media type: %s", err)
		}
	}

	return rw, mt
}
