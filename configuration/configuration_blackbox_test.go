package configuration_test

import (
	"fmt"
	"os"
	"testing"

	"net/http"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *configuration.ConfigurationData

func TestMain(m *testing.M) {
	resetConfiguration()

	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	os.Exit(m.Run())
}

func resetConfiguration() {
	var err error
	config, err = configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetKeycloakEndpointAuthDevModeOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := config.GetKeycloakEndpointAuth(reqLong)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/auth", url)

	url, err = config.GetKeycloakEndpointAuth(reqShort)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/auth", url)
}

func TestGetKeycloakEndpointAuthSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("ALMIGHTY_KEYCLOAK_ENDPOINT_AUTH")
	defer func() {
		os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_AUTH", env)
		resetConfiguration()
	}()

	os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_AUTH", "authEndpoint")
	resetConfiguration()

	url, err := config.GetKeycloakEndpointAuth(reqLong)
	assert.Nil(t, err)
	assert.Equal(t, "authEndpoint", url)

	url, err = config.GetKeycloakEndpointAuth(reqShort)
	assert.Nil(t, err)
	assert.Equal(t, "authEndpoint", url)
}

func TestGetKeycloakEndpointTokenOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := config.GetKeycloakEndpointToken(reqLong)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/token", url)

	url, err = config.GetKeycloakEndpointToken(reqShort)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/token", url)
}

func TestGetKeycloakEndpointTokenSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("ALMIGHTY_KEYCLOAK_ENDPOINT_TOKEN")
	defer func() {
		os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_TOKEN", env)
		resetConfiguration()
	}()

	os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_TOKEN", "tokenEndpoint")
	resetConfiguration()

	url, err := config.GetKeycloakEndpointToken(reqLong)
	assert.Nil(t, err)
	assert.Equal(t, "tokenEndpoint", url)

	url, err = config.GetKeycloakEndpointToken(reqShort)
	assert.Nil(t, err)
	assert.Equal(t, "tokenEndpoint", url)
}

func TestGetKeycloakEndpointUserInfoOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := config.GetKeycloakEndpointUserInfo(reqLong)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/userinfo", url)

	url, err = config.GetKeycloakEndpointUserInfo(reqShort)
	assert.Nil(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, "http://sso.demo.almighty.io/auth/realms/fabric8/protocol/openid-connect/userinfo", url)
}

func TestGetKeycloakEndpointUserInfoSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("ALMIGHTY_KEYCLOAK_ENDPOINT_USERINFO")
	defer func() {
		os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_USERINFO", env)
		resetConfiguration()
	}()

	os.Setenv("ALMIGHTY_KEYCLOAK_ENDPOINT_USERINFO", "userinfoEndpoint")
	resetConfiguration()

	url, err := config.GetKeycloakEndpointUserInfo(reqLong)
	assert.Nil(t, err)
	assert.Equal(t, "userinfoEndpoint", url)

	url, err = config.GetKeycloakEndpointUserInfo(reqShort)
	assert.Nil(t, err)
	assert.Equal(t, "userinfoEndpoint", url)
}
