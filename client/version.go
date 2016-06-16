package client

import (
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

// ShowVersionPath computes a request path to the show action of version.
func ShowVersionPath() string {
	return fmt.Sprintf("/api/version")
}

// Show current running version
func (c *Client) ShowVersion(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewShowVersionRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewShowVersionRequest create the request corresponding to the show action endpoint of the version resource.
func (c *Client) NewShowVersionRequest(ctx context.Context, path string) (*http.Request, error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.JWTSigner != nil {
		c.JWTSigner.Sign(req)
	}
	return req, nil
}
