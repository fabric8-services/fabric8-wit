package auth

import (
	"context"
	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	errs "github.com/pkg/errors"
)

// ResourceManager represents a space resource manager
type ResourceManager interface {
	CreateSpace(ctx context.Context, request *http.Request, spaceID string) error
	DeleteSpace(ctx context.Context, request *http.Request, spaceID string) error
}

// AuthzResourceManager implements ResourceManager interface
type AuthzResourceManager struct {
	configuration ServiceConfiguration
}

// ServiceConfiguration represents auth service configuration
type ServiceConfiguration interface {
	GetAuthServiceURL() string
	GetAuthShortServiceHostName() string
	IsAuthorizationEnabled() bool
}

// NewAuthzResourceManager constructs AuthzResourceManager
func NewAuthzResourceManager(config ServiceConfiguration) *AuthzResourceManager {
	return &AuthzResourceManager{config}
}

// CreateSpace calls auth service to create a keycloak resource associated with the space
func (m *AuthzResourceManager) CreateSpace(ctx context.Context, request *http.Request, spaceID string) error {
	if !m.configuration.IsAuthorizationEnabled() {
		// Keycloak authorization is disabled by default in Developer Mode
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Authorization is disabled. Keycloak space resource won't be created")
		return nil
	}

	c, err := CreateClient(ctx, m.configuration)
	if err != nil {
		return err
	}
	sUD, err := goauuid.FromString(spaceID)
	if err != nil {
		return err
	}
	res, err := c.CreateSpace(goasupport.ForwardContextRequestID(ctx), authservice.CreateSpacePath(sUD))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err.Error(),
		}, "unable to create a space resource via auth service")
		return errs.Wrap(err, "unable to create a space resource via auth service")
	}
	defer rest.CloseResponse(res)

	if res.StatusCode != http.StatusOK {
		responseBody := rest.ReadBody(res.Body)
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   responseBody,
		}, "unable to create a space resource via auth service")
		// Proxy-back back the response as is -
		// WIT acts as a gateway to Auth, who would send the appropriate response.
		return proxy.ConvertHTTPErrorCode(res.StatusCode, responseBody)
	}

	return nil
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
	c, err := CreateClient(ctx, m.configuration)
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
	defer rest.CloseResponse(res)

	responseBody := rest.ReadBody(res.Body)
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   responseBody,
		}, "unable to delete a space resource via auth service")
		// Proxy-back back the response as in -
		// WIT acts as a gateway to Auth, who would send the appropriate response.
		return proxy.ConvertHTTPErrorCode(res.StatusCode, responseBody)
	}

	log.Debug(ctx, map[string]interface{}{
		"space_id": spaceID,
	}, "Space resource deleted")

	return nil
}

func CreateClient(ctx context.Context, config ServiceConfiguration) (*authservice.Client, error) {
	u, err := url.Parse(config.GetAuthServiceURL())
	if err != nil {
		return nil, err
	}
	c := authservice.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return c, nil
}
