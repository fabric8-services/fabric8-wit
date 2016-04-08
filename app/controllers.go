//************************************************************************//
// API "alm": Application Controllers
//
// Generated with goagen v0.0.1, command line:
// $ goagen
// --out=$(GOPATH)/src/github.com/almighty/almighty-design
// --design=github.com/almighty/almighty-design/design
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

// inited is true if initService has been called
var inited = false

// initService sets up the service encoders, decoders and mux.
func initService(service *goa.Service) {
	if inited {
		return
	}
	inited = true

	// Setup encoders and decoders
	service.Encoder(goa.NewJSONEncoder, "application/json")
	service.Decoder(goa.NewJSONDecoder, "application/json")

	// Setup default encoder and decoder
	service.Encoder(goa.NewJSONEncoder, "*/*")
	service.Decoder(goa.NewJSONDecoder, "*/*")
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
	service.Mux.Handle("GET", "/api/version", ctrl.MuxHandler("Show", h, nil))
	service.LogInfo("mount", "ctrl", "Version", "action", "Show", "route", "GET /api/version")
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
			ctx = goa.LogWith(ctx, "origin", origin)
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
