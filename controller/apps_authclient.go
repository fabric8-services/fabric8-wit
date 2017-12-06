package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	rest "k8s.io/client-go/rest"
)

// AuthClient contains configuration and methods for interacting with OSIO API
type AuthClient struct {
	config *rest.Config
}

// AuthToken is an authorization token returned from the OSIO Auth API
// currently, github and openshift
type AuthToken struct {
	AccessToken *string `json:"access_token,omitempty"`
	Scope       *string `json:"scope,omitempty"`
	TokenType   *string `json:"token_type,omitempty"`
}

// AuthUser is a user record returned from the OSIO Auth API
// this is only a subset of the record
type AuthUser struct {
	Data struct {
		Attributes struct {
			Cluster  *string
			UserID   *string
			Username *string
		}
	}
}

// NewAuthClient creates an openshift IO client given an http request context
func NewAuthClient(authToken string, authURL string) *AuthClient {

	config := rest.Config{
		Host:        authURL,
		BearerToken: authToken,
	}

	client := new(AuthClient)
	client.config = &config

	return client
}

func (authclient *AuthClient) getAuthToken(forHost string) (*AuthToken, error) {
	var body []byte

	fullURL := strings.TrimSuffix(authclient.config.Host, "/") + "/api/token?for=" + forHost

	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+authclient.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status < 200 || status > 300 {
		return nil, fmt.Errorf("Failed to GET url %s due to status code %d", fullURL, status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType AuthToken
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

func (authclient *AuthClient) getAuthUser() (*AuthUser, error) {
	return authclient.getAuthUserByName(nil)
}

func (authclient *AuthClient) getAuthUserByName(username *string) (*AuthUser, error) {
	var body []byte

	path := "/api/user"
	if username != nil {
		path = "/api/users?filter[username]=" + *username
	}
	fullURL := strings.TrimSuffix(authclient.config.Host, "/") + path
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+authclient.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status < 200 || status > 300 {
		return nil, fmt.Errorf("Failed to GET url %s due to status code %d", fullURL, status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType AuthUser
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

// GetResource - generic JSON resource fetch
func (authclient *AuthClient) GetResource(url string, allowMissing bool) (map[string]interface{}, error) {
	var body []byte
	fullURL := strings.TrimSuffix(authclient.config.Host, "/") + url
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", "Bearer "+authclient.config.BearerToken)

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
		return nil, fmt.Errorf("Failed to GET url %s due to status code %d", fullURL, status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType map[string]interface{}
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return respType, nil
}
