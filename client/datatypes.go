//************************************************************************//
// User Types
//
// Generated with goagen v0.0.1, command line:
// $ goagen.exe
// --design=github.com/ALMighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\ALMighty\almighty-core
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package client

import "net/http"

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
