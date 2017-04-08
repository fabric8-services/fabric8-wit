package test

import (
	"context"

	"github.com/almighty/almighty-core/account"
	tokencontext "github.com/almighty/almighty-core/login/token_context"
	"github.com/almighty/almighty-core/token"

	"github.com/almighty/almighty-core/space"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

type dummySpaceAuthzService struct {
}

func (s *dummySpaceAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (*string, bool, error) {
	token := ""
	return &token, true, nil
}

func (s *dummySpaceAuthzService) Configuration() space.AuthzConfiguration {
	return nil
}

// WithIdentity fills the context with token
// Token is filled using input Identity object
func WithIdentity(ctx context.Context, ident account.Identity) context.Context {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["sub"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["uuid"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["fullName"] = ident.User.FullName
	token.Claims.(jwt.MapClaims)["imageURL"] = ident.User.ImageURL
	return goajwt.WithJWT(ctx, token)
}

func service(serviceName string, tm token.Manager, u account.Identity) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = WithIdentity(svc.Context, u)
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, tm)
	return svc
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, tm token.Manager, u account.Identity) *goa.Service {
	svc := service(serviceName, tm, u)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &space.KeyclaokAuthzServiceManager{Service: &dummySpaceAuthzService{}})
	return svc
}

// ServiceAsSpaceUser creates a new service and fill the context with input Identity and space authz service
func ServiceAsSpaceUser(serviceName string, tm token.Manager, u account.Identity, authz space.AuthzService) *goa.Service {
	svc := service(serviceName, tm, u)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &space.KeyclaokAuthzServiceManager{Service: authz})
	return svc
}
