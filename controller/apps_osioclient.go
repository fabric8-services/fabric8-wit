package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	witclient "github.com/fabric8-services/fabric8-wit/client"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	goaclient "github.com/goadesign/goa/client"
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
			return nil, err
		}
		userService = us
	}
	nameSpaces := userService.Attributes.Namespaces
	for _, ns := range nameSpaces {
		if *ns.Type == namespaceType {
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
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == 404 {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET " + witclient.ShowUserServicePath() + " due to status code " + string(status))
	}

	var respType app.UserServiceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType.Data, nil
}

// GetSpaceByID - fetch space given UUID
func (osioclient *OSIOClientV1) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	// there are two different uuid packages at play here:
	//   github.com/satori/go.uuid and goadesign/goa/uuid.
	// because of that, we fenerate our own URL to avoid issues for now.
	urlpath := fmt.Sprintf("/api/spaces/%s", spaceID.String())
	resp, err := osioclient.wc.ShowSpace(ctx, urlpath, nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status == 404 {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET " + urlpath + " due to status code " + string(status))
	}

	var respType app.SpaceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType.Data, nil
}
