package test

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	tokencontext "github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/fabric8-services/fabric8-wit/token"

	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

type dummySpaceAuthzService struct {
}

func (s *dummySpaceAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error) {
	return true, nil
}

func (s *dummySpaceAuthzService) Configuration() authz.AuthzConfiguration {
	return nil
}

// WithIdentity fills the context with token
// Token is filled using input Identity object
func WithIdentity(ctx context.Context, ident account.Identity) context.Context {
	token := fillClaimsWithIdentity(ident)
	return goajwt.WithJWT(ctx, token)
}

// WithAuthz fills the context with token
// Token is filled using input Identity object and resource authorization information
func WithAuthz(ctx context.Context, key interface{}, ident account.Identity, authz token.AuthorizationPayload) context.Context {
	token := fillClaimsWithIdentity(ident)
	token.Claims.(jwt.MapClaims)["authorization"] = authz
	token.Header["kid"] = "test-key"
	t, err := token.SignedString(key)
	if err != nil {
		panic(err.Error())
	}
	token.Raw = t
	return goajwt.WithJWT(ctx, token)
}

// WithServiceAccountAuthz fills the context with token
// Token is filled using input Identity object and resource authorization information
func WithServiceAccountAuthz(ctx context.Context, key interface{}, ident account.Identity) context.Context {
	token := fillClaimsWithIdentity(ident) // irrelavant for service account , but keeping it anyway.
	token.Claims.(jwt.MapClaims)["service_accountname"] = "fabric8-auth"
	token.Header["kid"] = "test-key"
	t, err := token.SignedString(key)
	if err != nil {
		panic(err.Error())
	}
	token.Raw = t
	return goajwt.WithJWT(ctx, token)
}

func fillClaimsWithIdentity(ident account.Identity) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["sub"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["uuid"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["fullName"] = ident.User.FullName
	token.Claims.(jwt.MapClaims)["imageURL"] = ident.User.ImageURL
	token.Claims.(jwt.MapClaims)["iat"] = time.Now().Unix()
	return token
}

func service(serviceName string, key interface{}, u account.Identity, authz *token.AuthorizationPayload) *goa.Service {
	svc := goa.New(serviceName)
	if authz == nil {
		svc.Context = WithIdentity(svc.Context, u)
	} else {
		svc.Context = WithAuthz(svc.Context, key, u, *authz)
	}
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, testtoken.TokenManager)
	return svc
}

// ServiceAsUserWithAuthz creates a new service and fill the context with input Identity and resource authorization information
func ServiceAsUserWithAuthz(serviceName string, key interface{}, u account.Identity, authorizationPayload token.AuthorizationPayload) *goa.Service {
	svc := service(serviceName, key, u, &authorizationPayload)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: &dummySpaceAuthzService{}})
	return svc
}

// ServiceAsServiceAccountUser generates the minimal service needed to satisfy the condition of being a service account.
func ServiceAsServiceAccountUser(serviceName string, u account.Identity) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = WithServiceAccountAuthz(svc.Context, testtoken.PrivateKey(), u)
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, testtoken.TokenManager)
	return svc
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, u account.Identity) *goa.Service {
	svc := service(serviceName, nil, u, nil)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: &dummySpaceAuthzService{}})
	return svc
}

// ServiceAsSpaceUser creates a new service and fill the context with input Identity and space authz service
func ServiceAsSpaceUser(serviceName string, u account.Identity, authzSrv authz.AuthzService) *goa.Service {
	svc := service(serviceName, nil, u, nil)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: authzSrv})
	return svc
}
