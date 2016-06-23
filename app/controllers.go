//************************************************************************//
// API "alm": Application Controllers
//
// Generated with goagen v0.2.dev, command line:
// $ goagen.exe
// --design=github.com/almighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\almighty\almighty-core
// --version=v0.2.dev
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
	service.Encoder.Register(goa.NewJSONEncoder, "application/json")
	service.Decoder.Register(goa.NewJSONDecoder, "application/json")

	// Setup default encoder and decoder
	service.Encoder.Register(goa.NewJSONEncoder, "*/*")
	service.Decoder.Register(goa.NewJSONDecoder, "*/*")
}

// LoginController is the controller interface for the Login actions.
type LoginController interface {
	goa.Muxer
	Authorize(*AuthorizeLoginContext) error
	Generate(*GenerateLoginContext) error
}

// MountLoginController "mounts" a Login resource controller on the given service.
func MountLoginController(service *goa.Service, ctrl LoginController) {
	initService(service)
	var h goa.Handler
	service.Mux.Handle("OPTIONS", "/api/login/authorize", ctrl.MuxHandler("preflight", handleLoginOrigin(cors.HandlePreflight()), nil))
	service.Mux.Handle("OPTIONS", "/api/login/generate", ctrl.MuxHandler("preflight", handleLoginOrigin(cors.HandlePreflight()), nil))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		// Check if there was an error loading the request
		if err := goa.ContextError(ctx); err != nil {
			return err
		}
		// Build the context
		rctx, err := NewAuthorizeLoginContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Authorize(rctx)
	}
	h = handleLoginOrigin(h)
	service.Mux.Handle("GET", "/api/login/authorize", ctrl.MuxHandler("Authorize", h, nil))
	service.LogInfo("mount", "ctrl", "Login", "action", "Authorize", "route", "GET /api/login/authorize")

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		// Check if there was an error loading the request
		if err := goa.ContextError(ctx); err != nil {
			return err
		}
		// Build the context
		rctx, err := NewGenerateLoginContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Generate(rctx)
	}
	h = handleLoginOrigin(h)
	service.Mux.Handle("GET", "/api/login/generate", ctrl.MuxHandler("Generate", h, nil))
	service.LogInfo("mount", "ctrl", "Login", "action", "Generate", "route", "GET /api/login/generate")
}

// handleLoginOrigin applies the CORS response headers corresponding to the origin.
func handleLoginOrigin(h goa.Handler) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			return h(ctx, rw, req)
		}
		if cors.MatchOrigin(origin, "*") {
			ctx = goa.WithLogContext(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*")
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
	service.Mux.Handle("OPTIONS", "/api/version", ctrl.MuxHandler("preflight", handleVersionOrigin(cors.HandlePreflight()), nil))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		// Check if there was an error loading the request
		if err := goa.ContextError(ctx); err != nil {
			return err
		}
		// Build the context
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
		if cors.MatchOrigin(origin, "*") {
			ctx = goa.WithLogContext(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*")
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

// WorkitemController is the controller interface for the Workitem actions.
type WorkitemController interface {
	goa.Muxer
	Show(*ShowWorkitemContext) error
}

// MountWorkitemController "mounts" a Workitem resource controller on the given service.
func MountWorkitemController(service *goa.Service, ctrl WorkitemController) {
	initService(service)
	var h goa.Handler
	service.Mux.Handle("OPTIONS", "/api/workitem/:id", ctrl.MuxHandler("preflight", handleWorkitemOrigin(cors.HandlePreflight()), nil))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		// Check if there was an error loading the request
		if err := goa.ContextError(ctx); err != nil {
			return err
		}
		// Build the context
		rctx, err := NewShowWorkitemContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Show(rctx)
	}
	h = handleWorkitemOrigin(h)
	service.Mux.Handle("GET", "/api/workitem/:id", ctrl.MuxHandler("Show", h, nil))
	service.LogInfo("mount", "ctrl", "Workitem", "action", "Show", "route", "GET /api/workitem/:id")
}

// handleWorkitemOrigin applies the CORS response headers corresponding to the origin.
func handleWorkitemOrigin(h goa.Handler) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			return h(ctx, rw, req)
		}
		if cors.MatchOrigin(origin, "*") {
			ctx = goa.WithLogContext(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*")
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

// WorkitemtypeController is the controller interface for the Workitemtype actions.
type WorkitemtypeController interface {
	goa.Muxer
	Show(*ShowWorkitemtypeContext) error
}

// MountWorkitemtypeController "mounts" a Workitemtype resource controller on the given service.
func MountWorkitemtypeController(service *goa.Service, ctrl WorkitemtypeController) {
	initService(service)
	var h goa.Handler
	service.Mux.Handle("OPTIONS", "/api/workitemtype/:id", ctrl.MuxHandler("preflight", handleWorkitemtypeOrigin(cors.HandlePreflight()), nil))

	h = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		// Check if there was an error loading the request
		if err := goa.ContextError(ctx); err != nil {
			return err
		}
		// Build the context
		rctx, err := NewShowWorkitemtypeContext(ctx, service)
		if err != nil {
			return err
		}
		return ctrl.Show(rctx)
	}
	h = handleWorkitemtypeOrigin(h)
	service.Mux.Handle("GET", "/api/workitemtype/:id", ctrl.MuxHandler("Show", h, nil))
	service.LogInfo("mount", "ctrl", "Workitemtype", "action", "Show", "route", "GET /api/workitemtype/:id")
}

// handleWorkitemtypeOrigin applies the CORS response headers corresponding to the origin.
func handleWorkitemtypeOrigin(h goa.Handler) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		origin := req.Header.Get("Origin")
		if origin == "" {
			// Not a CORS request
			return h(ctx, rw, req)
		}
		if cors.MatchOrigin(origin, "*") {
			ctx = goa.WithLogContext(ctx, "origin", origin)
			rw.Header().Set("Access-Control-Allow-Origin", "*")
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
