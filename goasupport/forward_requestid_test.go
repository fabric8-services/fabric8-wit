package goasupport

import (
	"context"
	"io"
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForwardRequest(t *testing.T) {
	reqID := uuid.NewV4().String()

	service := goa.New("test")
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/goo", nil)
	req.Header.Set(middleware.RequestIDHeader, reqID)
	service.Context = goa.NewContext(nil, rw, req, nil)
	service.Encoder.Register(func(w io.Writer) goa.Encoder { return goa.NewJSONEncoder(w) }, "*/*")
	service.Decoder.Register(func(r io.Reader) goa.Decoder { return goa.NewJSONDecoder(r) }, "*/*")

	var newCtx context.Context
	h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		newCtx = ctx
		return service.Send(ctx, 200, "ok")
	}
	rg := middleware.RequestID()(h)
	err := rg(service.Context, rw, req)
	require.NoError(t, err)

	assert.Equal(t, middleware.ContextRequestID(newCtx), reqID)

	clientCtx := ForwardContextRequestID(newCtx)
	assert.Equal(t, client.ContextRequestID(clientCtx), reqID)
}
