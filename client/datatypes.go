//************************************************************************//
// User Types
//
// Generated with goagen v0.0.1, command line:
// $ goagen
// --out=$(GOPATH)/src/github.com/almighty/almighty-core
// --design=github.com/almighty/almighty-core/design
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package client

import (
	"github.com/goadesign/goa"
	"io"
)

// JWT Token
type AuthToken struct {
	// JWT Token
	Token string `json:"token" xml:"token"`
}

// DecodeAuthToken decodes the AuthToken instance encoded in r.
func DecodeAuthToken(r io.Reader, decoderFn goa.DecoderFunc) (*AuthToken, error) {
	var decoded AuthToken
	err := decoderFn(r).Decode(&decoded)
	return &decoded, err
}

// The current running version
type Version struct {
	// The date when build
	BuildTime string `json:"build_time" xml:"build_time"`
	// Commit SHA this build is based on
	Commit string `json:"commit" xml:"commit"`
}

// DecodeVersion decodes the Version instance encoded in r.
func DecodeVersion(r io.Reader, decoderFn goa.DecoderFunc) (*Version, error) {
	var decoded Version
	err := decoderFn(r).Decode(&decoded)
	return &decoded, err
}
