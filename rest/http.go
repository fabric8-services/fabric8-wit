package rest

import (
	"context"
	"net/http"

	"github.com/goadesign/goa/client"
)

// Doer is a wrapper interface for goa client Doer
type HttpDoer interface {
	client.Doer
}

// HttpClient defines the Do method of the http client.
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HttpClientDoer implements HttpDoer
type HttpClientDoer struct {
	HttpClient HttpClient
}

// DefaultHttpDoer creates a new HttpDoer with default http client
func DefaultHttpDoer() HttpDoer {
	return &HttpClientDoer{HttpClient: http.DefaultClient}
}

// Do overrides Do method of the default goa client Doer. It's needed for mocking http clients in tests.
func (d *HttpClientDoer) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return d.HttpClient.Do(req)
}
