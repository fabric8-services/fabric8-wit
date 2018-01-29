package controller

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	witclient "github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	goaclient "github.com/goadesign/goa/client"
	goauuid "github.com/goadesign/goa/uuid"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// OSIOClient contains configuration and methods for interacting with OSIO API
type OSIOClientV1 struct {
	wc *witclient.Client
}

// NewOSIOClient creates an openshift IO client given an http request context
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
				return nil, errs.Errorf("Namespace with type %s found, but has no name", namespaceType)
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
		return nil, errs.Wrapf(err, "could not retrieve uses services")
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errs.Errorf("failed to GET %s due to status code %d", witclient.ShowUserServicePath(), status)
	}

	var respType app.UserServiceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, errs.Wrapf(err, "could not unmarshal user services JSON")
	}
	return respType.Data, nil
}

// GetSpaceByID - fetch space given UUID
func (osioclient *OSIOClientV1) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	// there are two different uuid packages at play here:
	// github.com/satori/go.uuid and goadesign/goa/uuid.
	// because of that, we generate our own URL to avoid issues for now.
	var guid goauuid.UUID
	for i, b := range spaceID {
		guid[i] = b
	}
	urlpath := witclient.ShowSpacePath(guid)
	resp, err := osioclient.wc.ShowSpace(ctx, urlpath, nil, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "could not connect to %s", urlpath)
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errs.Errorf("failed to GET %s due to status code %d", urlpath, status)
	}

	var respType app.SpaceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, errs.Wrapf(err, "could not unmarshal SpaceSingle JSON")
	}
	return respType.Data, nil
}
