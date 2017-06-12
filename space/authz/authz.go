// Package authz contains the code that authorizes space operations
package authz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/auth"
	errs "github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	tokencontext "github.com/almighty/almighty-core/login/tokencontext"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/token"

	contx "context"
	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
)

// TokenPayload represents an rpt token
type TokenPayload struct {
	jwt.StandardClaims
	Authorization *AuthorizationPayload `json:"authorization"`
}

// AuthorizationPayload represents an authz payload in the rpt token
type AuthorizationPayload struct {
	Permissions []Permissions `json:"permissions"`
}

// Permissions represents an permissions and the AuthorizationPayload
type Permissions struct {
	ResourceSetName *string `json:"resource_set_name"`
	ResourceSetID   *string `json:"resource_set_id"`
}

// AuthzService represents a space authorization service
type AuthzService interface {
	Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error)
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

// KeycloakAuthzServiceManager is a keyaloak implementation of a space autharizarion service
type KeycloakAuthzServiceManager struct {
	Service             AuthzService
	entitlementEndpoint string
}

// AuthzService returns a space autharizarion service
func (m *KeycloakAuthzServiceManager) AuthzService() AuthzService {
	return m.Service
}

// EntitlementEndpoint returns a keyclaok entitlement endpoint URL
func (m *KeycloakAuthzServiceManager) EntitlementEndpoint() string {
	return m.entitlementEndpoint
}

// KeycloakAuthzService implements AuthzService interface
type KeycloakAuthzService struct {
	config AuthzConfiguration
	db     application.DB
}

// NewAuthzService constructs a new KeycloakAuthzService
func NewAuthzService(config AuthzConfiguration, db application.DB) *KeycloakAuthzService {
	return &KeycloakAuthzService{config: config, db: db}
}

// Configuration returns authz service configuration
func (s *KeycloakAuthzService) Configuration() AuthzConfiguration {
	return s.config
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func (s *KeycloakAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error) {
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return false, errs.NewUnauthorizedError("missing token")
	}
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")
		return false, errs.NewInternalError("Missing token manager")
	}
	tokenWithClaims, err := jwt.ParseWithClaims(jwttoken.Raw, &TokenPayload{}, func(t *jwt.Token) (interface{}, error) {
		return tm.(token.Manager).PublicKey(), nil
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space-id": spaceID,
			"err":      err,
		}, "unable to parse the rpt token")
		return false, errs.NewInternalError(fmt.Sprintf("unable to parse the rpt token: %s", err.Error()))
	}
	claims := tokenWithClaims.Claims.(*TokenPayload)

	if claims.Authorization == nil {
		// No authorization in the token. This is not a RPT token. This is an access token.
		// We need to obtain an PRT token.
		log.Warn(ctx, map[string]interface{}{
			"space-id": spaceID,
		}, "no authorization found in the token; this is an access token (not a RPT token)")
		return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
	}

	// Check if the token was issued before the space resouces changed the last time.
	// If so, we need to re-fetch the rpt token for that space/resource and check permissions.
	outdated, err := s.isTokenOutdated(ctx, *claims, entitlementEndpoint, spaceID)
	if err != nil {
		return false, err
	}
	if outdated {
		return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
	}

	permissions := claims.Authorization.Permissions
	if permissions == nil {
		return false, nil
	}
	for _, permission := range permissions {
		name := permission.ResourceSetName
		if name != nil && spaceID == *name {
			return true, nil
		}
	}
	return false, nil
}

func (s *KeycloakAuthzService) checkEntitlementForSpace(ctx context.Context, token jwt.Token, entitlementEndpoint string, spaceID string) (bool, error) {
	resource := auth.EntitlementResource{
		Permissions: []auth.ResourceSet{{Name: spaceID}},
	}
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &resource, token.Raw)
	if err != nil {
		return false, err
	}
	return ent != nil, nil
}

func (s *KeycloakAuthzService) isTokenOutdated(ctx context.Context, token TokenPayload, entitlementEndpoint string, spaceID string) (bool, error) {
	spaceUUID, err := uuid.FromString(spaceID)
	if err != nil {
		return false, errs.NewInternalError(err.Error())
	}
	var spaceResource *space.Resource
	err = application.Transactional(s.db, func(appl application.Application) error {
		spaceResource, err = appl.SpaceResources().LoadBySpace(ctx, &spaceUUID)
		return err
	})
	if err != nil {
		return false, err
	}
	if token.IssuedAt == 0 {
		return false, errs.NewInternalError("iat claim is not found in the token")
	}
	tokenIssued := time.Unix(token.IssuedAt, 0)
	return tokenIssued.Before(spaceResource.UpdatedAt), nil
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
			"space-id": spaceID,
		}, "Missing space authz service")

		return false, errors.New("missing space authz service")
	}
	manager := srv.(AuthzServiceManager)
	return manager.AuthzService().Authorize(ctx, manager.EntitlementEndpoint(), spaceID)
}
