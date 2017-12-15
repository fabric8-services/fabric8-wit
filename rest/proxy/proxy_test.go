package proxy

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"

	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	go startServer()
	waitForServer(t)

	// GET with custom header and 201 response
	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.NoError(t, err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx, err := app.NewShowStatusContext(goaCtx, req, goa.New("StatusService"))
	require.NoError(t, err)
	statusCtx.Request.Header.Del("Accept-Encoding")

	err = RouteHTTP(statusCtx, "http://localhost:8889")
	require.NoError(t, err)

	assert.Equal(t, 201, rw.Code)
	assert.Equal(t, "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body := rest.ReadBody(rw.Result().Body)
	assert.Equal(t, veryLongBody, body)

	// POST, gzipped, changed target path
	rw = httptest.NewRecorder()
	req, err = http.NewRequest("POST", u.String(), nil)
	require.NoError(t, err)

	ctx = context.Background()
	goaCtx = goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx, err = app.NewShowStatusContext(goaCtx, req, goa.New("StatusService"))
	require.NoError(t, err)
	statusCtx.Request.Header.Set("Accept-Encoding", "gzip")

	err = RouteHTTPToPath(statusCtx, "http://localhost:8889", "/api")
	require.NoError(t, err)

	assert.Equal(t, 201, rw.Code)
	assert.Equal(t, "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body = rest.ReadBody(rw.Result().Body)
	assert.Equal(t, veryLongBody, body)
}

func startServer() {
	http.HandleFunc("/api", handlerGzip)
	http.ListenAndServe(":8889", nil)
}

func waitForServer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8889/api", nil)
	require.NoError(t, err)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		client := &http.Client{Timeout: time.Duration(500 * time.Millisecond)}
		res, err := client.Do(req)
		if err == nil && res.StatusCode == 201 {
			return
		}
	}
	assert.Fail(t, "Failed to start server")
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Custom-Test-Header", "proxyTest")
	w.WriteHeader(201)
	fmt.Fprint(w, veryLongBody)
}

func handlerGzip(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		handler(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
	handler(gzr, r)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) WriteHeader(code int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

var veryLongBody = generateLongBody()

func generateLongBody() string {
	body := uuid.NewV4().String()
	for i := 0; i < 100; i++ {
		body = body + uuid.NewV4().String()
	}
	return body
}
