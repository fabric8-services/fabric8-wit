package auth

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// ResourceManager represents a space resource manager
type ResourceManager interface {
	CreateSpace(ctx context.Context, request *http.Request, spaceID string) (*authservice.SpaceResource, error)
	DeleteSpace(ctx context.Context, request *http.Request, spaceID string) error
}

// AuthzResourceManager implements ResourceManager interface
type AuthzResourceManager struct {
	configuration AuthServiceConfiguration
}

// AuthServiceConfiguration represents auth service configuration
type AuthServiceConfiguration interface {
	GetAuthEndpointSpaces(*http.Request) (string, error)
	IsAuthorizationEnabled() bool
}

// NewAuthzResourceManager constructs AuthzResourceManager
func NewAuthzResourceManager(config AuthServiceConfiguration) *AuthzResourceManager {
	return &AuthzResourceManager{config}
}

// CreateSpace calls auth service to create a keycloak resource associated with the space
func (m *AuthzResourceManager) CreateSpace(ctx context.Context, request *http.Request, spaceID string) (*authservice.SpaceResource, error) {
	if !m.configuration.IsAuthorizationEnabled() {
		// Keycloak authorization is disabled by default in Developer Mode
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Authorization is disabled. Keycloak space resource won't be created")
		return &authservice.SpaceResource{Data: &authservice.SpaceResourceData{
			ResourceID:   uuid.NewV4().String(),
			PermissionID: uuid.NewV4().String(),
			PolicyID:     uuid.NewV4().String(),
		}}, nil
	}

	c, err := m.createClient(ctx, request)
	if err != nil {
		return nil, err
	}
	sUD, err := goauuid.FromString(spaceID)
	if err != nil {
		return nil, err
	}
	res, err := c.CreateSpace(goasupport.ForwardContextRequestID(ctx), authservice.CreateSpacePath(sUD))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err.Error(),
		}, "unable to create a space resource via auth service")
		return nil, errs.Wrap(err, "unable to create a space resource via auth service")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to create a space resource via auth service")
		return nil, errs.Errorf("unable to create a space resource via auth service. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body))
	}

	resource, err := c.DecodeSpaceResource(res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id":      spaceID,
			"response_body": rest.ReadBody(res.Body),
		}, "unable to decode the create space resource request result")

		return nil, errs.Wrapf(err, "unable to decode the create space resource request result %s ", rest.ReadBody(res.Body))
	}

	log.Debug(ctx, map[string]interface{}{
		"space_id":    spaceID,
		"resource_id": resource.Data.ResourceID,
	}, "Space resource created")

	return resource, nil
}

// DeleteSpace calls auth service to delete the keycloak resource associated with the space
func (m *AuthzResourceManager) DeleteSpace(ctx context.Context, request *http.Request, spaceID string) error {
	if !m.configuration.IsAuthorizationEnabled() {
		// Keycloak authorization is disabled by default in Developer Mode
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Authorization is disabled. Keycloak space resource won't be deleted")
		return nil
	}
	c, err := m.createClient(ctx, request)
	if err != nil {
		return err
	}
	sUD, err := goauuid.FromString(spaceID)
	if err != nil {
		return err
	}
	res, err := c.DeleteSpace(goasupport.ForwardContextRequestID(ctx), authservice.CreateSpacePath(sUD))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err.Error(),
		}, "unable to delete a space resource via auth service")
		return errs.Wrap(err, "unable to delete a space resource via auth service")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to delete a space resource via auth service")
		return errs.Errorf("unable to delete a space resource via auth service. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body))
	}

	log.Debug(ctx, map[string]interface{}{
		"space_id": spaceID,
	}, "Space resource deleted")

	return nil
}

func (m *AuthzResourceManager) createClient(ctx context.Context, request *http.Request) (*authservice.Client, error) {
	authSpacesEndpoint, err := m.configuration.GetAuthEndpointSpaces(request)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(authSpacesEndpoint)
	if err != nil {
		return nil, err
	}
	c := authservice.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c, nil
}
