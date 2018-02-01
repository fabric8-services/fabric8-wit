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

// Allows for reading from responses
type ResponseReader interface {
	ReadResponse(*http.Response) ([]byte, error)
}

type IOResponseReader struct {
}

func (r *IOResponseReader) ReadResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// Interface for mocking the witclient.Client
type WitClient interface {
	ShowSpace(ctx context.Context, path string, ifModifiedSince *string, ifNoneMatch *string) (*http.Response, error)
	ShowUserService(ctx context.Context, path string) (*http.Response, error)
}

// Interface for mocking OSIOClient
type OpenshiftIOClient interface {
	GetNamespaceByType(ctx context.Context, userService *app.UserService, namespaceType string) (*app.NamespaceAttributes, error)
	GetUserServices(ctx context.Context) (*app.UserService, error)
	GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error)
}

// OSIOClient contains configuration and methods for interacting with OSIO API
type OSIOClient struct {
	wc             WitClient
	responseReader ResponseReader
}

// NewOSIOClient creates an openshift IO client given an http request context
func NewOSIOClient(ctx context.Context, scheme string, host string) *OSIOClient {
	wc := witclient.New(goaclient.HTTPClientDoer(http.DefaultClient))
	wc.Host = host
	wc.Scheme = scheme
	wc.SetJWTSigner(goasupport.NewForwardSigner(ctx))
	return CreateOSIOClient(wc, &IOResponseReader{})
}

func CreateOSIOClient(witclient WitClient, responseReader ResponseReader) *OSIOClient {
	client := new(OSIOClient)
	client.wc = witclient
	client.responseReader = responseReader
	return client
}

// GetNamespaceByType finds a namespace by type (user, che, stage, etc)
// if userService is nil, will fetch the user services under the hood
func (osioclient *OSIOClient) GetNamespaceByType(ctx context.Context, userService *app.UserService, namespaceType string) (*app.NamespaceAttributes, error) {
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
			return ns, nil
		}
	}
	return nil, nil
}

// GetUserServices - fetch array of user services
// In the future, consider calling the tenant service (as /api/user/services implementation does)
func (osioclient *OSIOClient) GetUserServices(ctx context.Context) (*app.UserService, error) {
	resp, err := osioclient.wc.ShowUserService(ctx, witclient.ShowUserServicePath())
	if err != nil {
		return nil, errs.Wrapf(err, "could not retrieve uses services")
	}

	respBody, err := osioclient.responseReader.ReadResponse(resp)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		return nil, nil
	} else if status != http.StatusOK {
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
func (osioclient *OSIOClient) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	guid := goauuid.UUID(spaceID)
	urlpath := witclient.ShowSpacePath(guid)
	resp, err := osioclient.wc.ShowSpace(ctx, urlpath, nil, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "could not connect to %s", urlpath)
	}

	respBody, err := osioclient.responseReader.ReadResponse(resp)

	status := resp.StatusCode
	if status == http.StatusNotFound {
		return nil, nil
	} else if status != http.StatusOK {
		return nil, errs.Errorf("failed to GET %s due to status code %d", urlpath, status)
	}

	var respType app.SpaceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, errs.Wrapf(err, "could not unmarshal SpaceSingle JSON")
	}
	return respType.Data, nil
}
