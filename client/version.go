package client

import (
	"golang.org/x/net/context"
	"io"
	"net/http"
	"net/url"
)

// Show current running version
func (c *Client) ShowVersion(ctx context.Context, path string) (*http.Response, error) {
	var body io.Reader
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("GET", u.String(), body)
	if err != nil {
		return nil, err
	}
	header := req.Header
	header.Set("Content-Type", "application/json")
	c.SignerJWT.Sign(ctx, req)
	return c.Client.Do(ctx, req)
}
