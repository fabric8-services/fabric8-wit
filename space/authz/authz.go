// Package authz contains the code that authorizes space operations
package authz

import (
	"context"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// AuthzService represents a space authorization service
type AuthzService interface {
	Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error)
	Configuration() AuthzConfiguration
}

// AuthzConfiguration represents a Keycloak entitlement endpoint configuration
type AuthzConfiguration interface {
	GetKeycloakEndpointEntitlement(*http.Request) (string, error)
	IsAuthorizationEnabled() bool
}

// AuthzServiceManager represents a space autharizarion service
type AuthzServiceManager interface {
	AuthzService() AuthzService
	EntitlementEndpoint() string
}

// KeycloakAuthzServiceManager is a keyaloak implementation of a space autharizarion service
type KeycloakAuthzServiceManager struct {
	Service             AuthzService
	entitlementEndpoint string
}

// AuthzService returns a space autharizarion service
func (m *KeycloakAuthzServiceManager) AuthzService() AuthzService {
	return m.Service
}

// EntitlementEndpoint returns a keycloak entitlement endpoint URL
func (m *KeycloakAuthzServiceManager) EntitlementEndpoint() string {
	return m.entitlementEndpoint
}

// KeycloakAuthzService implements AuthzService interface
type KeycloakAuthzService struct {
	config AuthzConfiguration
}

// NewAuthzService constructs a new KeycloakAuthzService
func NewAuthzService(config AuthzConfiguration) *KeycloakAuthzService {
	return &KeycloakAuthzService{config: config}
}

// Configuration returns authz service configuration
func (s *KeycloakAuthzService) Configuration() AuthzConfiguration {
	return s.config
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func (s *KeycloakAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error) {
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return false, errors.NewUnauthorizedError("missing token")
	}
	return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
}

func (s *KeycloakAuthzService) checkEntitlementForSpace(ctx context.Context, token jwt.Token, entitlementEndpoint string, spaceID string) (bool, error) {
	if !s.config.IsAuthorizationEnabled() {
		// Keycloak authorization is disabled by default in Developer Mode
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Authorization is disabled. All users are allowed to operate the space")
		return true, nil
	}
	resource := auth.EntitlementResource{
		Permissions: []auth.ResourceSet{{Name: spaceID}},
	}
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &resource, token.Raw)
	if err != nil {
		return false, err
	}
	return ent != nil, nil
}

// InjectAuthzService is a middleware responsible for setting up AuthzService in the context for every request.
func InjectAuthzService(service AuthzService) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			config := service.Configuration()
			var endpoint string
			if config != nil {
				var err error
				endpoint, err = config.GetKeycloakEndpointEntitlement(req)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"err": err,
					}, "unable to get entitlement endpoint")
					return err
				}
			}
			ctxWithAuthzServ := tokencontext.ContextWithSpaceAuthzService(ctx, &KeycloakAuthzServiceManager{Service: service, entitlementEndpoint: endpoint})
			return h(ctxWithAuthzServ, rw, req)
		}
	}
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func Authorize(ctx context.Context, spaceID string) (bool, error) {
	srv := tokencontext.ReadSpaceAuthzServiceFromContext(ctx)
	if srv == nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Missing space authz service")

		return false, errs.New("missing space authz service")
	}
	manager := srv.(AuthzServiceManager)
	return manager.AuthzService().Authorize(ctx, manager.EntitlementEndpoint(), spaceID)
}
