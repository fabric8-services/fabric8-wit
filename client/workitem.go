package client

import (
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

// ShowWorkitemPath computes a request path to the show action of workitem.
func ShowWorkitemPath(id string) string {
	return fmt.Sprintf("/api/workitem/%v", id)
}

// Retrieve work item with given id.
func (c *Client) ShowWorkitem(ctx context.Context, path string) (*http.Response, error) {
	req, err := c.NewShowWorkitemRequest(ctx, path)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewShowWorkitemRequest create the request corresponding to the show action endpoint of the workitem resource.
func (c *Client) NewShowWorkitemRequest(ctx context.Context, path string) (*http.Request, error) {
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
