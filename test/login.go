package test

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// WithIdentity fills the context with token
// Token is filled using input Identity object
func WithIdentity(ctx context.Context, ident account.Identity) context.Context {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["fullName"] = ident.User.FullName
	token.Claims.(jwt.MapClaims)["imageURL"] = ident.User.ImageURL
	return goajwt.WithJWT(ctx, token)
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, tm token.Manager, u account.Identity) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = WithIdentity(svc.Context, u)
	svc.Context = login.ContextWithTokenManager(svc.Context, tm)
	return svc
}
