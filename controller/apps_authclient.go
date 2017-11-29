package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
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
func NewAuthClient(ctx context.Context, authURL string) (*AuthClient, error) {

	config := rest.Config{
		Host:        authURL,
		BearerToken: goajwt.ContextJWT(ctx).Raw,
	}

	// TODO - remove before production
	if os.Getenv("OSIO_TOKEN") != "" {
		config.BearerToken = os.Getenv("OSIO_TOKEN")
	}

	client := new(AuthClient)
	client.config = &config

	return client, nil
}

func (authclient *AuthClient) getAuthToken(forHost string) (*AuthToken, error) {
	var body []byte

	fullURL := strings.TrimSuffix(authclient.config.Host, "/") + "/token?for=" + forHost

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

func (authclient *AuthClient) ccgetAuthUser(username string) (*AuthUser, error) {
	//path := "/users?filter[username]=" + username
	path := "/user"
	json, err := authclient.GetResource(path, false)
	if err != nil {
		return nil, err
	}
	fmt.Println("returned user json=" + tostring(json))
	return nil, nil
}

func (authclient *AuthClient) getAuthUser() (*AuthUser, error) {
	return authclient.getAuthUserByName(nil)
}

func (authclient *AuthClient) getAuthUserByName(username *string) (*AuthUser, error) {
	var body []byte

	path := "/user"
	if username != nil {
		path = "/users?filter[username]=" + *username
	}
	fullURL := strings.TrimSuffix(authclient.config.Host, "/") + path
	fmt.Println("AUTH URL= " + fullURL)
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
	fmt.Println("full URL=", fullURL)
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
