package sentry

import (
	"context"
	"os"
	"sync"

	"github.com/fabric8-services/fabric8-wit/token"

	"github.com/getsentry/raven-go"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// Client encapsulates client to Sentry service
// also has mutex which controls access to the client
type Client struct {
	c   *raven.Client
	mux sync.Mutex
}

var (
	sentryClient *Client
)

// Sentry returns client declared inside package
func Sentry() *Client {
	return sentryClient
}

// InitializeSentryClient initializes sentry client
func InitializeSentryClient(release, environment string) error {
	c, err := raven.New(os.Getenv("SENTRY_DSN"))
	if err != nil {
		return err
	}

	c.SetRelease(release)
	c.SetEnvironment(environment)
	sentryClient = &Client{
		c: c,
	}

	return nil
}

// CaptureError sends error 'err' to Sentry, meanwhile also sets user
// information by extracting user information from the context provided
func (c *Client) CaptureError(ctx context.Context, err error) error {
	// extract user information
	user, errLocal := extractUserInfo(ctx)
	if errLocal != nil {
		return errLocal
	}

	c.mux.Lock()
	c.c.SetUserContext(user)
	c.c.CaptureError(err, nil)
	c.c.ClearContext()
	c.mux.Unlock()

	return nil
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
