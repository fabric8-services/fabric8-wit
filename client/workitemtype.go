package client

import (
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

// ShowWorkitemtypePath computes a request path to the show action of workitemtype.
func ShowWorkitemtypePath(id string) string {
	return fmt.Sprintf("/api/workitemtype/%v", id)
}

// Retrieve work item type with given id.
func (c *Client) ShowWorkitemtype(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewShowWorkitemtypeRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewShowWorkitemtypeRequest create the request corresponding to the show action endpoint of the workitemtype resource.
func (c *Client) NewShowWorkitemtypeRequest(ctx context.Context, path string) (*http.Request, error) {
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
