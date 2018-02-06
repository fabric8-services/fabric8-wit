package controller

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	witclient "github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/log"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// OSIOClientV1 contains configuration and methods for interacting with OSIO API
type OSIOClientV1 struct {
	wc *witclient.Client
}

// NewOSIOClientV1 creates an openshift IO client given an http request context
func NewOSIOClientV1(ctx context.Context, scheme string, host string) *OSIOClientV1 {

	client := new(OSIOClientV1)
	httpClient := newHTTPClient()
	client.wc = witclient.New(goaclient.HTTPClientDoer(httpClient))
	client.wc.Host = host
	client.wc.Scheme = scheme
	client.wc.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return client
}

// newHTTPClient returns the HTTP client used by the API client to make requests to the service.
func newHTTPClient() *http.Client {
	// TODO change timeout
	return http.DefaultClient
}

// GetNamespaceByType finds a namespace by type (user, che, stage, etc)
// if userService is nil, will fetch the user services under the hood
func (osioclient *OSIOClientV1) GetNamespaceByType(ctx context.Context, userService *app.UserService, namespaceType string) (*app.NamespaceAttributes, error) {
	if userService == nil {
		us, err := osioclient.GetUserServices(ctx)
		if err != nil {
			return nil, errs.Wrapf(err, "could not retrieve user services")
		}
		userService = us
	}
	nameSpaces := userService.Attributes.Namespaces
	for _, ns := range nameSpaces {
		if *ns.Type == namespaceType {
			if ns.Name == nil {
				return nil, errs.Errorf("namespace with type %s found, but has no name", namespaceType)
			}
			return ns, nil
		}
	}
	return nil, nil
}

// GetUserServices - fetch array of user services
// In the future, consider calling the tenant service (as /api/user/services implementation does)
func (osioclient *OSIOClientV1) GetUserServices(ctx context.Context) (*app.UserService, error) {
	resp, err := osioclient.wc.ShowUserService(ctx, witclient.ShowUserServicePath())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"show_user_sevice_path": witclient.ShowUserServicePath(),
			"err": err,
		}, "could not retrieve uses services from ", witclient.ShowUserServicePath())
		return nil, errs.Wrapf(err, "could not retrieve uses services")
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		// 404 Not Found is either a bad URL path or an invalid space.
		return nil, nil
	} else if status != http.StatusOK {
		return nil, errs.Errorf("failed to GET %s due to status code %d", witclient.ShowUserServicePath(), status)
	}

	var respType app.UserServiceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"json": respBody,
			"err":  err,
		}, "could not unmarshal UserServiceSingle JSON")
		return nil, errs.Wrapf(err, "could not unmarshal UserServiceSingle JSON")
	}
	return respType.Data, nil
}

// GetSpaceByID - fetch space given UUID
func (osioclient *OSIOClientV1) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {

	urlpath := witclient.ShowSpacePath(goauuid.UUID(spaceID))

	resp, err := osioclient.wc.ShowSpace(ctx, urlpath, nil, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"show_space_path": witclient.ShowSpacePath(goauuid.UUID(spaceID)),
			"err":             err,
		}, "could not retrieve space from %s", witclient.ShowSpacePath(goauuid.UUID(spaceID)))
		return nil, errs.Wrapf(err, "could not connect to %s", urlpath)
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		return nil, nil
	} else if status != http.StatusOK {
		return nil, errs.Errorf("failed to GET %s due to status code %d", urlpath, status)
	}

	var respType app.SpaceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"json": respBody,
			"err":  err,
		}, "could not unmarshal SpaceSingle JSON")
		return nil, errs.Wrapf(err, "could not unmarshal SpaceSingle JSON")
	}
	return respType.Data, nil
}
