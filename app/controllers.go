//************************************************************************//
// API "alm": Application Controllers
//
// Generated with goagen v0.0.1, command line:
// $ goagen
// --out=$(GOPATH)/src/github.com/almighty/almighty-core
// --design=github.com/almighty/almighty-core/design
// --pkg=app
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import (
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/cors"
	"golang.org/x/net/context"
	"net/http"
)

// initService sets up the service encoders, decoders and mux.
func initService(service *goa.Service) {
	// Setup encoders and decoders
	service.Encoder(goa.NewJSONEncoder, "application/json")
	service.Decoder(goa.NewJSONDecoder, "application/json")

	// Setup default encoder and decoder
	service.Encoder(goa.NewJSONEncoder, "*/*")
	service.Decoder(goa.NewJSONDecoder, "*/*")
}

// LoginController is the controller interface for the Login actions.
type LoginController interface {
	goa.Muxer
	Authorize(*AuthorizeLoginContext) error
}

// MountLoginController "mounts" a Login resource controller on the given service.
func MountLoginController(service *goa.Service, ctrl LoginController) {
	initService(service)
	var h goa.Handler
	service.Mux.Handle("OPTIONS", "/api/login/authorize", cors.HandlePreflight(service.Context, handleLoginOrigin))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		rctx, err := NewAuthorizeLoginContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Authorize(rctx)
	}
	h = handleLoginOrigin(h)
	service.Mux.Handle("GET", "/api/login/authorize", ctrl.MuxHandler("Authorize", h, nil))
	service.LogInfo("mount", "ctrl", "Login", "action", "Authorize", "route", "GET /api/login/authorize")
}

// handleLoginOrigin applies the CORS response headers corresponding to the origin.
func handleLoginOrigin(h goa.Handler) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			return h(ctx, rw, req)
		}
		if cors.MatchOrigin(origin, "*.almighty.io") {
			ctx = goa.WithLog(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*.almighty.io")
			rw.Header().Set("Vary", "Origin")
			rw.Header().Set("Access-Control-Max-Age", "600")
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
			if acrm := req.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			}
			return h(ctx, rw, req)
		}
		return h(ctx, rw, req)
	}
}

// VersionController is the controller interface for the Version actions.
type VersionController interface {
	goa.Muxer
	Show(*ShowVersionContext) error
}

// MountVersionController "mounts" a Version resource controller on the given service.
func MountVersionController(service *goa.Service, ctrl VersionController) {
	initService(service)
	var h goa.Handler
	service.Mux.Handle("OPTIONS", "/api/version", cors.HandlePreflight(service.Context, handleVersionOrigin))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		rctx, err := NewShowVersionContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Show(rctx)
	}
	h = handleVersionOrigin(h)
	h = handleSecurity("jwt", h, "system")
	service.Mux.Handle("GET", "/api/version", ctrl.MuxHandler("Show", h, nil))
	service.LogInfo("mount", "ctrl", "Version", "action", "Show", "route", "GET /api/version", "security", "jwt")
}

// handleVersionOrigin applies the CORS response headers corresponding to the origin.
func handleVersionOrigin(h goa.Handler) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			return h(ctx, rw, req)
		}
		if cors.MatchOrigin(origin, "*.almighty.io") {
			ctx = goa.WithLog(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*.almighty.io")
			rw.Header().Set("Vary", "Origin")
			rw.Header().Set("Access-Control-Max-Age", "600")
			rw.Header().Set("Access-Control-Allow-Credentials", "true")
			if acrm := req.Header.Get("Access-Control-Request-Method"); acrm != "" {
				// We are handling a preflight request
				rw.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			}
			return h(ctx, rw, req)
		}
		return h(ctx, rw, req)
	}
}
