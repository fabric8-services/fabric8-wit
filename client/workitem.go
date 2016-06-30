package client

import (
	"bytes"
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
)

// CreateWorkitemPayload is the workitem create action payload.
type CreateWorkitemPayload struct {
	// The field values, must conform to the type
	Fields map[string]interface{} `json:"fields,omitempty" xml:"fields,omitempty" form:"fields,omitempty"`
	// User Readable Name of this item
	Name *string `json:"name,omitempty" xml:"name,omitempty" form:"name,omitempty"`
	// The type of the newly created work item
	TypeID *string `json:"typeId,omitempty" xml:"typeId,omitempty" form:"typeId,omitempty"`
}

// CreateWorkitemPath computes a request path to the create action of workitem.
func CreateWorkitemPath() string {
	return fmt.Sprintf("/api/workitem")
}

// create work item with type and id.
func (c *Client) CreateWorkitem(ctx context.Context, path string, payload *CreateWorkitemPayload, contentType string) (*http.Response, error) {
	req, err := c.NewCreateWorkitemRequest(ctx, path, payload, contentType)
	if err != nil {
		return nil, err
	}
	return c.Client.Do(ctx, req)
}

// NewCreateWorkitemRequest create the request corresponding to the create action endpoint of the workitem resource.
func (c *Client) NewCreateWorkitemRequest(ctx context.Context, path string, payload *CreateWorkitemPayload, contentType string) (*http.Request, error) {
	var body bytes.Buffer
	if contentType == "" {
		contentType = "*/*" // Use default encoder
	}
	err := c.Encoder.Encode(payload, &body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to encode body: %s", err)
	}
	scheme := c.Scheme
	if scheme == "" {
		scheme = "http"
	}
	u := url.URL{Host: c.Host, Scheme: scheme, Path: path}
	req, err := http.NewRequest("POST", u.String(), &body)
	if err != nil {
		return nil, err
	}
	header := req.Header
	if contentType != "*/*" {
		header.Set("Content-Type", contentType)
	}
	return req, nil
}

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
