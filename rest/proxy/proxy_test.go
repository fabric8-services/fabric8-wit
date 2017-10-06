package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	go startServer()
	waitForServer(t)

	rw := httptest.NewRecorder()
	u, err := url.Parse("http://domain.org/api")
	require.Nil(t, err)
	req, err := http.NewRequest("GET", u.String(), nil)
	require.Nil(t, err)

	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "ProxyTest"), rw, req, url.Values{})
	statusCtx, err := app.NewShowStatusContext(goaCtx, req, goa.New("StatusService"))
	require.Nil(t, err)

	err = RouteHTTP(statusCtx, "http://localhost:8889")
	require.Nil(t, err)

	assert.Equal(t, 201, rw.Code)
	assert.Equal(t, "proxyTest", rw.Header().Get("Custom-Test-Header"))
	body := rest.ReadBody(rw.Result().Body)
	assert.Equal(t, "Hi there!", body)
}

func waitForServer(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:8889/api", nil)
	require.Nil(t, err)
	attempt := 1
	for ; attempt < 30; attempt++ {
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
	fmt.Fprint(w, "Hi there!")
}

func startServer() {
	http.HandleFunc("/api", handler)
	http.ListenAndServe(":8889", nil)
}
