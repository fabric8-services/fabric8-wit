//************************************************************************//
// API "alm": Application Security
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
	"golang.org/x/net/context"
	"net/http"
)

type securitySchemeKey string
type key int

const securityScopesKey key = 1

func ConfigureJWTSecurity(service *goa.Service, f goa.JWTSecurityConfigFunc) {
	def := &goa.JWTSecurity{

		In:       goa.LocHeader,
		Name:     "Authorization",
		TokenURL: "/api/login/authorize",
	}
	def.Description = "JWT Token Auth"

	fetchScopes := func(ctx context.Context) []string {
		scopes, _ := ctx.Value(securityScopesKey).([]string)
		return scopes
	}
	middleware := f(def, fetchScopes)

	service.Context = context.WithValue(service.Context, securitySchemeKey("jwt"), middleware)
}

func handleSecurity(schemeName string, h goa.Handler, scopes ...string) goa.Handler {
	return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
		scheme := ctx.Value(securitySchemeKey(schemeName))
		middleware, ok := scheme.(goa.Middleware)
		if !ok {
			return goa.NoSecurityScheme(schemeName)
		}

		if len(scopes) != 0 {
			ctx = context.WithValue(ctx, securityScopesKey, scopes)
		}

		return middleware(h)(ctx, rw, req)
	}
}
