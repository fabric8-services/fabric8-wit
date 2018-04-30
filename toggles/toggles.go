package toggles

import (
	"context"
	"os"
	"time"

	unleash "github.com/Unleash/unleash-client-go"
	ucontext "github.com/Unleash/unleash-client-go/context"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/log"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

var ready = false

// Init toggle client lib
func Init(serviceName, hostURL string) {
	unleash.Initialize(
		unleash.WithAppName(serviceName),
		unleash.WithInstanceId(os.Getenv("HOSTNAME")),
		unleash.WithUrl(hostURL),
		unleash.WithMetricsInterval(1*time.Minute),
		unleash.WithRefreshInterval(10*time.Second),
		unleash.WithListener(&listener{}),
	)
}

// WithContext creates a Token based contex
func WithContext(ctx context.Context) unleash.FeatureOption {
	uctx := ucontext.Context{}
	token := goajwt.ContextJWT(ctx)
	if token != nil {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			uctx.UserId = claims["sub"].(string)
			uctx.SessionId = claims["session_state"].(string)
		}
	}
	return unleash.WithContext(uctx)
}

// IsEnabled wraps unleash for a simpler API
func IsEnabled(ctx context.Context, feature string, fallback bool) bool {
	if !ready {
		return fallback
	}
	return unleash.IsEnabled(feature, WithContext(ctx), unleash.WithFallback(fallback))
}

type listener struct{}

// OnError prints out errors.
func (l listener) OnError(err error) {
	log.Error(nil, map[string]interface{}{
		"err": err.Error(),
	}, "toggles error")
}

// OnWarning prints out warning.
func (l listener) OnWarning(warning error) {
	log.Warn(nil, map[string]interface{}{
		"err": warning.Error(),
	}, "toggles warning")
}

// OnReady prints to the console when the repository is ready.
func (l listener) OnReady() {
	ready = true
	log.Info(nil, map[string]interface{}{}, "toggles ready")
}

// OnCount prints to the console when the feature is queried.
func (l listener) OnCount(name string, enabled bool) {
	log.Info(nil, map[string]interface{}{
		"name":    name,
		"enabled": enabled,
	}, "toggles count")
}

// OnSent prints to the console when the server has uploaded metrics.
func (l listener) OnSent(payload unleash.MetricsData) {
	log.Info(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles sent")
}

// OnRegistered prints to the console when the client has registered.
func (l listener) OnRegistered(payload unleash.ClientData) {
	log.Info(nil, map[string]interface{}{
		"payload": payload,
	}, "toggles registered")
}
