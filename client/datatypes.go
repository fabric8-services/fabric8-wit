//************************************************************************//
// User Types
//
// Generated with goagen v0.2.dev, command line:
// $ goagen.exe
// --design=github.com/almighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\almighty\almighty-core
// --version=v0.2.dev
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package client

import "net/http"

// Field user type.
type Field struct {
	Name string `json:"name" xml:"name" form:"name"`
	Type string `json:"type" xml:"type" form:"type"`
}

// JWT Token
type AuthToken struct {
	// JWT Token
	Token string `json:"token" xml:"token" form:"token"`
}

// DecodeAuthToken decodes the AuthToken instance encoded in resp body.
func (c *Client) DecodeAuthToken(resp *http.Response) (*AuthToken, error) {
	var decoded AuthToken
	err := c.Decoder.Decode(&decoded, resp.Body, resp.Header.Get("Content-Type"))
	return &decoded, err
}

// AuthTokenCollection media type is a collection of AuthToken.
type AuthTokenCollection []*AuthToken

// DecodeAuthTokenCollection decodes the AuthTokenCollection instance encoded in resp body.
func (c *Client) DecodeAuthTokenCollection(resp *http.Response) (AuthTokenCollection, error) {
	var decoded AuthTokenCollection
	err := c.Decoder.Decode(&decoded, resp.Body, resp.Header.Get("Content-Type"))
	return decoded, err
}

// The current running version
type Version struct {
	// The date when build
	BuildTime string `json:"build_time" xml:"build_time" form:"build_time"`
	// Commit SHA this build is based on
	Commit string `json:"commit" xml:"commit" form:"commit"`
}

// DecodeVersion decodes the Version instance encoded in resp body.
func (c *Client) DecodeVersion(resp *http.Response) (*Version, error) {
	var decoded Version
	err := c.Decoder.Decode(&decoded, resp.Body, resp.Header.Get("Content-Type"))
	return &decoded, err
}

// ALM Work Item
type WorkItem struct {
	Fields map[string]interface{} `json:"fields" xml:"fields" form:"fields"`
	// unique id per installation
	ID string `json:"id" xml:"id" form:"id"`
	// User Readable Name of this item
	Name string `json:"name" xml:"name" form:"name"`
	// Id of the type of this work item
	Type string `json:"type" xml:"type" form:"type"`
	// Version for optimistic concurrency control
	Version int `json:"version" xml:"version" form:"version"`
}

// DecodeWorkItem decodes the WorkItem instance encoded in resp body.
func (c *Client) DecodeWorkItem(resp *http.Response) (*WorkItem, error) {
	var decoded WorkItem
	err := c.Decoder.Decode(&decoded, resp.Body, resp.Header.Get("Content-Type"))
	return &decoded, err
}

// ALM Work Item Type
type WorkItemType struct {
	Fields []*Field `json:"fields" xml:"fields" form:"fields"`
	// unique id per installation
	ID string `json:"id" xml:"id" form:"id"`
	// User Readable Name of this item
	Name string `json:"name" xml:"name" form:"name"`
	// Version for optimistic concurrency control
	Version int `json:"version" xml:"version" form:"version"`
}

// DecodeWorkItemType decodes the WorkItemType instance encoded in resp body.
func (c *Client) DecodeWorkItemType(resp *http.Response) (*WorkItemType, error) {
	var decoded WorkItemType
	err := c.Decoder.Decode(&decoded, resp.Body, resp.Header.Get("Content-Type"))
	return &decoded, err
}
