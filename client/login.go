package client

import (
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

// AuthorizeLoginPath computes a request path to the authorize action of login.
func AuthorizeLoginPath() string {
	return fmt.Sprintf("/api/login/authorize")
}

// Authorize with the ALM
func (c *Client) AuthorizeLogin(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewAuthorizeLoginRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewAuthorizeLoginRequest create the request corresponding to the authorize action endpoint of the login resource.
func (c *Client) NewAuthorizeLoginRequest(ctx context.Context, path string) (*http.Request, error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}
