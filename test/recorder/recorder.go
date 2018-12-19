package recorder

import (
	"fmt"
	"net/http"
	"os"

	jwt "github.com/dgrijalva/jwt-go"
	jwtrequest "github.com/dgrijalva/jwt-go/request"
	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/fabric8-services/fabric8-wit/log"
	testjwt "github.com/fabric8-services/fabric8-wit/test/jwt"
	errs "github.com/pkg/errors"
)

// Option an option to customize the recorder to create
type Option func(*recorder.Recorder)

// WithMatcher an option to specify a custom matcher for the recorder
func WithMatcher(matcher cassette.Matcher) Option {
	return func(r *recorder.Recorder) {
		r.SetMatcher(matcher)
	}
}

// WithJWTMatcher an option to specify the JWT matcher for the recorder
func WithJWTMatcher(publicKey string) Option {
	return func(r *recorder.Recorder) {
		r.SetMatcher(newJWTMatcher(publicKey))
	}
}

// New creates a new recorder
func New(cassetteName string, options ...Option) (*recorder.Recorder, error) {
	_, err := os.Stat(fmt.Sprintf("%s.yaml", cassetteName))
	if err != nil {
		return nil, errs.Wrapf(err, "unable to find file '%s.yaml'", cassetteName)
	}
	r, err := recorder.New(cassetteName)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to create recorder from file '%s.yaml'", cassetteName)
	}
	// custom cassette matcher that will compare the HTTP requests' token subject with the `sub` header of the recorded data (the yaml file)
	for _, opt := range options {
		opt(r)
	}
	return r, nil
}

// newJWTMatcher a cassette matcher that verifies the request method/URL and the subject of the token in the "Authorization" header.
func newJWTMatcher(publicKey string) cassette.Matcher {
	return func(httpRequest *http.Request, cassetteRequest cassette.Request) bool {
		// check the request URI and method
		if httpRequest.Method != cassetteRequest.Method ||
			(httpRequest.URL != nil && httpRequest.URL.String() != cassetteRequest.URL) {
			return false
		}
		// look-up the JWT's "sub" claim and compare with the request
		token, err := jwtrequest.ParseFromRequest(httpRequest, jwtrequest.AuthorizationHeaderExtractor, func(*jwt.Token) (interface{}, error) {
			return testjwt.PublicKey(publicKey)
		})
		//
		if err == jwtrequest.ErrNoTokenInRequest {
			// no token in request, but that may be expected, after all
			if _, found := cassetteRequest.Headers["sub"]; !found {
				return true
			}
			return false
		} else if err != nil {
			log.Error(nil, map[string]interface{}{
				"error":                err.Error(),
				"request_method":       cassetteRequest.Method,
				"request_url":          cassetteRequest.URL,
				"authorization_header": httpRequest.Header["Authorization"]},
				"failed to parse token from request")
			return false
		}
		claims := token.Claims.(jwt.MapClaims)
		sub, found := cassetteRequest.Headers["sub"]
		if found && len(sub) > 0 && sub[0] == claims["sub"] {
			log.Debug(nil, map[string]interface{}{
				"method": cassetteRequest.Method,
				"url":    cassetteRequest.URL,
				"sub":    sub[0]}, "found interaction")
			return true
		}
		log.Debug(nil, map[string]interface{}{
			"method":              cassetteRequest.Method,
			"url":                 cassetteRequest.URL,
			"cassetteRequest_sub": sub,
			"http_request_sub":    claims["sub"],
		}, "Authorization header's 'sub' claim doesn't match with the current request")

		return false
	}
}
