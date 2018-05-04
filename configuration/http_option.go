package configuration

import (
	"net/http"
)

// HTTPClientOption options passed to the HTTP Client
type HTTPClientOption func(client *http.Client)

// WithRoundTripper configures the client's transport with the given round-tripper
func WithRoundTripper(r http.RoundTripper) HTTPClientOption {
	return func(client *http.Client) {
		client.Transport = r
	}
}
