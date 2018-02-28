package sentry

import (
	"context"
	"os"
	"sync"

	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// client encapsulates client to Sentry service
// also has mutex which controls access to the client
type client struct {
	c   *raven.Client
	mux sync.Mutex
}

var (
	sentryClient *client
)

// Sentry returns client declared inside package
func Sentry() *client {
	return sentryClient
}

// InitializeSentryClient initializes sentry client
func InitializeSentryClient(options ...func(*client)) error {
	c, err := raven.New(os.Getenv("SENTRY_DSN"))
	if err != nil {
		return err
	}
	sentryClient = &client{
		c: c,
	}
	// set all options passed by user
	for _, opt := range options {
		opt(sentryClient)
	}

	return nil
}

// WithRelease helps you set release/commit of currently running
// code while initializing sentry client using function InitializeSentryClient
func WithRelease(release string) func(*client) {
	return func(c *client) {
		c.c.SetRelease(release)
	}
}

// WithEnvironment helps you set environment the deployed code is
// running in while initializing sentry client using function
// InitializeSentryClient
func WithEnvironment(env string) func(*client) {
	return func(c *client) {
		c.c.SetEnvironment(env)
	}
}

// CaptureError sends error 'err' to Sentry, meanwhile also sets user
// information by extracting user information from the context provided
func (c *client) CaptureError(ctx context.Context, err error) {
	// Extract user information. Ignoring error here but then before using the
	// object user make sure to check if it wasn't nil.
	user, _ := extractUserInfo(ctx)

	c.mux.Lock()
	if user != nil {
		c.c.SetUserContext(user)
	}
	c.c.CaptureError(err, nil)
	c.c.ClearContext()
	c.mux.Unlock()
}

// extractUserInfo reads the context and returns sentry understandable
// user object's reference and error
func extractUserInfo(ctx context.Context) (*raven.User, error) {
	m, err := token.ReadManagerFromContext(ctx)
	if err != nil {
		return nil, err
	}

	q := *m
	token := goajwt.ContextJWT(ctx)
	t, err := q.ParseToken(ctx, token.Raw)
	if err != nil {
		return nil, err
	}

	return &raven.User{
		Username: t.Username,
		Email:    t.Email,
		ID:       t.Id,
	}, nil
}
