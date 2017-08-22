package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// ResourceManager represents a space resource manager
type ResourceManager interface {
	CreateSpace(ctx context.Context, request *goa.RequestData, spaceID string) (*SpaceResource, error)
	DeleteSpace(ctx context.Context, request *goa.RequestData, spaceID string) error
}

// AuthzResourceManager implements ResourceManager interface
type AuthzResourceManager struct {
	configuration AuthServiceConfiguration
}

type SpaceResource struct {
	Data SpaceResourceData `form:"data" json:"data" xml:"data"`
}

type SpaceResourceData struct {
	PermissionID string `form:"permissionID" json:"permissionID" xml:"permissionID"`
	PolicyID     string `form:"policyID" json:"policyID" xml:"policyID"`
	ResourceID   string `form:"resourceID" json:"resourceID" xml:"resourceID"`
}

// AuthServiceConfiguration represents auth service configuration
type AuthServiceConfiguration interface {
	GetAuthEndpointSpaces(*goa.RequestData) (string, error)
}

// NewAuthzResourceManager constructs AuthzResourceManager
func NewAuthzResourceManager(config AuthServiceConfiguration) *AuthzResourceManager {
	return &AuthzResourceManager{config}
}

// CreateSpace calls auth service to create a keycloak resource associated with the space
func (m *AuthzResourceManager) CreateSpace(ctx context.Context, request *goa.RequestData, spaceID string) (*SpaceResource, error) {
	authSpacesEndpoint, err := m.configuration.GetAuthEndpointSpaces(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", authSpacesEndpoint, spaceID), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return nil, errs.Wrap(err, "unable to create http request")
	}
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return nil, errors.NewUnauthorizedError("missing token")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwttoken.Raw))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err.Error(),
		}, "unable to create a sapace resource via auth service")
		return nil, errs.Wrap(err, "unable to create a sapace resource via auth service")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to create a sapace resource via auth service")
		return nil, errs.Errorf("unable to create a sapace resource via auth service. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body))
	}
	jsonString := rest.ReadBody(res.Body)

	var resource SpaceResource
	err = json.Unmarshal([]byte(jsonString), &resource)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id":    spaceID,
			"json_string": jsonString,
		}, "unable to unmarshal json with the create space resource request result")

		return nil, errs.Wrapf(err, "unable to unmarshal json with the create space resource request result %s ", jsonString)
	}

	log.Debug(ctx, map[string]interface{}{
		"space_id":    spaceID,
		"resource_id": resource.Data.ResourceID,
	}, "Space resource created")

	return &resource, nil
}

// DeleteSpace calls auth service to delete the keycloak resource associated with the space
func (m *AuthzResourceManager) DeleteSpace(ctx context.Context, request *goa.RequestData, spaceID string) error {
	authSpacesEndpoint, err := m.configuration.GetAuthEndpointSpaces(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/%s", authSpacesEndpoint, spaceID), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return errs.Wrap(err, "unable to create http request")
	}
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return errors.NewUnauthorizedError("missing token")
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwttoken.Raw))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err.Error(),
		}, "unable to delete a sapace resource via auth service")
		return errs.Wrap(err, "unable to delete a sapace resource via auth service")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		log.Error(ctx, map[string]interface{}{
			"space_id":        spaceID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to delete a sapace resource via auth service")
		return errs.Errorf("unable to delete a sapace resource via auth service. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body))
	}

	log.Debug(ctx, map[string]interface{}{
		"space_id": spaceID,
	}, "Space resource deleted")

	return nil
}
