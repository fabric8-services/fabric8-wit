package space

import (
	"context"
	"errors"
	"net/http"

	"github.com/almighty/almighty-core/auth"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	tokencontext "github.com/almighty/almighty-core/login/token_context"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	contx "golang.org/x/net/context"
)

// AuthzService represents a space authorization service
type AuthzService interface {
	Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (*string, bool, error)
	Configuration() AuthzConfiguration
}

// AuthzConfiguration represents a Keycloak entitlement endpoint configuration
type AuthzConfiguration interface {
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
}

// AuthzServiceManager represents a space autharizarion service
type AuthzServiceManager interface {
	AuthzService() AuthzService
	EntitlementEndpoint() string
}

// KeyclaokAuthzServiceManager is a keyaloak implementation of a space autharizarion service
type KeyclaokAuthzServiceManager struct {
	Service             AuthzService
	entitlementEndpoint string
}

// AuthzService returns a space autharizarion service
func (m *KeyclaokAuthzServiceManager) AuthzService() AuthzService {
	return m.Service
}

// EntitlementEndpoint returns a keyclaok entitlement endpoint URL
func (m *KeyclaokAuthzServiceManager) EntitlementEndpoint() string {
	return m.entitlementEndpoint
}

// KeyclaokAuthzService implements AuthzService interface
type KeyclaokAuthzService struct {
	config AuthzConfiguration
}

// NewAuthzService constructs a new KeyclaokAuthzService
func NewAuthzService(config AuthzConfiguration) *KeyclaokAuthzService {
	return &KeyclaokAuthzService{config: config}
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func (s *KeyclaokAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (*string, bool, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return nil, false, errs.NewUnauthorizedError("missing token")
	}

	resource := auth.EntitlementResource{
		Permissions: []auth.ResourceSet{{Name: spaceID}},
	}
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, resource, token.Raw)
	if err != nil {
		return nil, false, err
	}
	return nil, ent != nil, nil
}

// Configuration returns authz service configuration
func (s *KeyclaokAuthzService) Configuration() AuthzConfiguration {
	return s.config
}

// InjectAuthzService is a middleware responsible for setting up AuthzService in the context for every request.
func InjectAuthzService(service AuthzService) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx contx.Context, rw http.ResponseWriter, req *http.Request) error {
			config := service.Configuration()
			var endpoint string
			if config != nil {
				var err error
				endpoint, err = config.GetKeycloakEndpointEntitlement(&goa.RequestData{Request: req})
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"err": err,
					}, "unable to get entitlement endpoint")
					return err
				}
			}
			ctxWithAuthzServ := tokencontext.ContextWithSpaceAuthzService(ctx, &KeyclaokAuthzServiceManager{Service: service, entitlementEndpoint: endpoint})
			return h(ctxWithAuthzServ, rw, req)
		}
	}
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func Authorize(ctx context.Context, spaceID string) (*string, bool, error) {
	srv := tokencontext.ReadSpaceAuthzServiceFromContext(ctx)
	if srv == nil {
		log.Error(ctx, map[string]interface{}{
			"space-id": spaceID,
		}, "Missing space authz service")

		return nil, false, errors.New("missing space authz service")
	}
	manager := srv.(AuthzServiceManager)
	return manager.AuthzService().Authorize(ctx, manager.EntitlementEndpoint(), spaceID)
}
