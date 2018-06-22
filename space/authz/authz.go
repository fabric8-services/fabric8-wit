// Package authz contains the code that authorizes space operations
package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/rest"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// AuthzService represents a space authorization service
type AuthzService interface {
	Authorize(ctx context.Context, spaceID string) (bool, error)
	Configuration() auth.ServiceConfiguration
}

// AuthzServiceManager represents a space autharizarion service
type AuthzServiceManager interface {
	AuthzService() AuthzService
}

// KeycloakAuthzServiceManager is a keyaloak implementation of a space autharizarion service
type KeycloakAuthzServiceManager struct {
	Service AuthzService
	Config  auth.ServiceConfiguration
}

// AuthzService returns a space autharizarion service
func (m *KeycloakAuthzServiceManager) AuthzService() AuthzService {
	return m.Service
}

// KeycloakAuthzService implements AuthzService interface
type KeycloakAuthzService struct {
	config auth.ServiceConfiguration
	doer   rest.HttpDoer
}

// NewAuthzService constructs a new KeycloakAuthzService
func NewAuthzService(config auth.ServiceConfiguration) *KeycloakAuthzService {
	return &KeycloakAuthzService{config: config, doer: rest.DefaultHttpDoer()}
}

// Configuration returns authz service configuration
func (s *KeycloakAuthzService) Configuration() auth.ServiceConfiguration {
	return s.config
}

// Authorize returns true if the current user is among the space collaborators
func (s *KeycloakAuthzService) Authorize(ctx context.Context, spaceID string) (bool, error) {
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return false, errors.NewUnauthorizedError("missing token")
	}
	return s.checkRole(ctx, *jwttoken, spaceID)
}

type Roles struct {
	Data []Role `json:"data"`
}

type Role struct {
	RoleName   string `json:"role_name"`
	AssigneeID string `json:"assignee_id"`
}

func (s *KeycloakAuthzService) checkRole(ctx context.Context, token jwt.Token, spaceID string) (bool, error) {
	if !s.config.IsAuthorizationEnabled() {
		// Keycloak authorization is disabled by default in Developer Mode
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Authorization is disabled. All users are allowed to operate the space")
		return true, nil
	}
	currentIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/resources/%s/roles", s.config.GetAuthServiceURL(), spaceID), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token.Raw)
	res, err := s.doer.Do(ctx, req)
	if err != nil {
		return false, errors.NewInternalError(ctx, err)
	}
	defer rest.CloseResponse(res)
	bodyString := rest.ReadBody(res.Body)
	if res.StatusCode != http.StatusOK {
		return false, errors.NewInternalError(ctx, errs.New("unable to get space roles. Response status: "+res.Status+". Response body: "+bodyString))
	}

	var roles Roles
	err = json.Unmarshal([]byte(bodyString), &roles)
	if err != nil {
		return false, errors.NewInternalError(ctx, err)
	}

	id := currentIdentityID.String()
	for _, r := range roles.Data {
		if r.AssigneeID == id && (r.RoleName == "admin" || r.RoleName == "contributor") {
			return true, nil
		}
	}
	return false, nil
}

// InjectAuthzService is a middleware responsible for setting up AuthzService in the context for every request.
func InjectAuthzService(service AuthzService) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctxWithAuthzServ := tokencontext.ContextWithSpaceAuthzService(ctx, &KeycloakAuthzServiceManager{Service: service, Config: service.Configuration()})
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
	return manager.AuthzService().Authorize(ctx, spaceID)
}
