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

// AuthorizeLoginOK Authorize runs the method Authorize of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func AuthorizeLoginOK(t *testing.T, ctrl app.LoginController) (http.ResponseWriter, *app.AuthToken) {
	return AuthorizeLoginOKWithContext(t, context.Background(), ctrl)
}

// AuthorizeLoginOKWithContext Authorize runs the method Authorize of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func AuthorizeLoginOKWithContext(t *testing.T, ctx context.Context, ctrl app.LoginController) (http.ResponseWriter, *app.AuthToken) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/authorize"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Authorize(authorizeCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}
	if rw.Code != 200 {
		t.Errorf("invalid response status code: got %+v, expected 200", rw.Code)
	}
	var mt *app.AuthToken
	if resp != nil {
		var ok bool
		mt, ok = resp.(*app.AuthToken)
		if !ok {
			t.Errorf("invalid response media: got %+v, expected instance of app.AuthToken", resp)
		}
		err = mt.Validate()
		if err != nil {
			t.Errorf("invalid response media type: %s", err)
		}
	}

	return rw, mt
}

// AuthorizeLoginUnauthorized Authorize runs the method Authorize of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers.
func AuthorizeLoginUnauthorized(t *testing.T, ctrl app.LoginController) http.ResponseWriter {
	return AuthorizeLoginUnauthorizedWithContext(t, context.Background(), ctrl)
}

// AuthorizeLoginUnauthorizedWithContext Authorize runs the method Authorize of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers.
func AuthorizeLoginUnauthorizedWithContext(t *testing.T, ctx context.Context, ctrl app.LoginController) http.ResponseWriter {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/authorize"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Authorize(authorizeCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}
	if rw.Code != 401 {
		t.Errorf("invalid response status code: got %+v, expected 401", rw.Code)
	}

	return rw
}

// GenerateLoginOK Generate runs the method Generate of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func GenerateLoginOK(t *testing.T, ctrl app.LoginController) (http.ResponseWriter, *app.AuthTokenCollection) {
	return GenerateLoginOKWithContext(t, context.Background(), ctrl)
}

// GenerateLoginOKWithContext Generate runs the method Generate of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers and the media type struct written to the response.
func GenerateLoginOKWithContext(t *testing.T, ctx context.Context, ctrl app.LoginController) (http.ResponseWriter, *app.AuthTokenCollection) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/generate"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	generateCtx, err := app.NewGenerateLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Generate(generateCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}
	if rw.Code != 200 {
		t.Errorf("invalid response status code: got %+v, expected 200", rw.Code)
	}
	var mt *app.AuthTokenCollection
	if resp != nil {
		var ok bool
		mt, ok = resp.(*app.AuthTokenCollection)
		if !ok {
			t.Errorf("invalid response media: got %+v, expected instance of app.AuthTokenCollection", resp)
		}
		err = mt.Validate()
		if err != nil {
			t.Errorf("invalid response media type: %s", err)
		}
	}

	return rw, mt
}

// GenerateLoginUnauthorized Generate runs the method Generate of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers.
func GenerateLoginUnauthorized(t *testing.T, ctrl app.LoginController) http.ResponseWriter {
	return GenerateLoginUnauthorizedWithContext(t, context.Background(), ctrl)
}

// GenerateLoginUnauthorizedWithContext Generate runs the method Generate of the given controller with the given parameters.
// It returns the response writer so it's possible to inspect the response headers.
func GenerateLoginUnauthorizedWithContext(t *testing.T, ctx context.Context, ctrl app.LoginController) http.ResponseWriter {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/generate"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	prms := url.Values{}

	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, prms)
	generateCtx, err := app.NewGenerateLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}

	err = ctrl.Generate(generateCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}
	if rw.Code != 401 {
		t.Errorf("invalid response status code: got %+v, expected 401", rw.Code)
	}

	return rw
}
