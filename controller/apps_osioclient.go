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
	uuid "github.com/satori/go.uuid"
)

// Allows for reading from responses
type ResponseReader interface {
	ReadResponse(*http.Response) ([]byte, error)
}

type IOResponseReader struct {
}

func (r *IOResponseReader) ReadResponse(resp *http.Response) ([]byte, error) {
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
	wc WitClient
	responseReader ResponseReader
}

// NewOSIOClient creates an openshift IO client given a wit client
func NewOSIOClient(witclient WitClient, responseReader ResponseReader) *OSIOClient {
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
func (osioclient *OSIOClient) GetUserServices(ctx context.Context) (*app.UserService, error) {
	resp, err := osioclient.wc.ShowUserService(ctx, witclient.ShowUserServicePath())
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := osioclient.responseReader.ReadResponse(resp)

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
func (osioclient *OSIOClient) GetSpaceByID(ctx context.Context, spaceID uuid.UUID) (*app.Space, error) {
	// there are two different uuid packages at play here:
	//   github.com/satori/go.uuid and goadesign/goa/uuid.
	// because of that, we fenerate our own URL to avoid issues for now.
	urlpath := fmt.Sprintf("/api/spaces/%s", spaceID.String())
	resp, err := osioclient.wc.ShowSpace(ctx, urlpath, nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := osioclient.responseReader.ReadResponse(resp)

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
