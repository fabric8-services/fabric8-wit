package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/app"
	rest "k8s.io/client-go/rest"
)

// OSIOClient contains configuration and methods for interacting with OSIO API
type OSIOClient struct {
	config *rest.Config
}

// NewOsioClient creates an openshift IO client given an http request context
func NewOSIOClient(authToken string, witURL string) *OSIOClient {

	config := rest.Config{
		Host:        witURL,
		BearerToken: authToken,
	}

	client := new(OSIOClient)
	client.config = &config

	return client
}

// GetResource - generic JSON resource fetch
func (osioclient *OSIOClient) getResource(url string, allowMissing bool) (map[string]interface{}, error) {
	var body []byte
	fullURL := strings.TrimSuffix(osioclient.config.Host, "/") + url
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+osioclient.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status == 404 && allowMissing {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET url " + fullURL + " due to status code " + string(status))
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType map[string]interface{}
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType, nil
}

// GetNamespaceByType finds a namespace by type (user, che, stage, etc)
// if userService is nil, will fetch the user services under the hood
func (osioclient *OSIOClient) GetNamespaceByType(userService *app.UserService, namespaceType string) (*app.NamespaceAttributes, error) {
	if userService == nil {
		us, err := osioclient.GetUserServices()
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
func (osioclient *OSIOClient) GetUserServices() (*app.UserService, error) {
	var body []byte
	fullURL := strings.TrimSuffix(osioclient.config.Host, "/") + "/api/user/services"
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+osioclient.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status == 404 {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET url " + fullURL + " due to status code " + string(status))
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType app.UserServiceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType.Data, nil
}

// GetSpaceByName - fetch space given username and spacename
func (osioclient *OSIOClient) GetSpaceByName(username string, spaceName string, allowMissing bool) (*app.Space, error) {
	fullURL := strings.TrimSuffix(osioclient.config.Host, "/") + "/api/namedspaces/" + username + "/" + spaceName
	return osioclient.getSpace(fullURL, allowMissing)
}

// GetSpaceByID - fetch space given UUID
func (osioclient *OSIOClient) GetSpaceByID(spaceID string, allowMissing bool) (*app.Space, error) {
	fullURL := strings.TrimSuffix(osioclient.config.Host, "/") + "/api/spaces/" + spaceID
	return osioclient.getSpace(fullURL, allowMissing)
}

func (osioclient *OSIOClient) getSpace(fullURL string, allowMissing bool) (*app.Space, error) {
	var body []byte
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+osioclient.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status == 404 && allowMissing {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET url " + fullURL + " due to status code " + string(status))
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType app.SpaceSingle
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType.Data, nil
}
