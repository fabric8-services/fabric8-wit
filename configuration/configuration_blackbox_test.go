package configuration_test

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"net/http"

	"time"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultConfigFilePath       = "../config.yaml"
	defaultValuesConfigFilePath = "" // when the code defaults are to be used, the path to config file is ""
)

var reqLong *http.Request
var reqShort *http.Request
var config *configuration.Registry

func TestMain(m *testing.M) {
	resetConfiguration(defaultConfigFilePath)

	reqLong = &http.Request{Host: "api.service.domain.org"}
	reqShort = &http.Request{Host: "api.domain.org"}
	os.Exit(m.Run())
}

func resetConfiguration(configPath string) {
	var err error
	config, err = configuration.New(configPath)
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestGetAuthURLSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("F8_AUTH_URL")
	defer func() {
		os.Setenv("F8_AUTH_URL", env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv("F8_AUTH_URL", "https://auth.xyz.io")
	resetConfiguration(defaultValuesConfigFilePath)

	url := config.GetAuthServiceURL()
	require.Equal(t, "https://auth.xyz.io", url)
}

func TestGetKeycloakEndpointSetByUrlEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	env := os.Getenv("F8_KEYCLOAK_URL")
	defer func() {
		os.Setenv("F8_KEYCLOAK_URL", env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv("F8_KEYCLOAK_URL", "http://xyz.io")
	resetConfiguration(defaultValuesConfigFilePath)

	url, err := config.GetKeycloakEndpointAuth(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/protocol/openid-connect/auth", url)

	url, err = config.GetKeycloakEndpointLogout(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/protocol/openid-connect/logout", url)

	url, err = config.GetKeycloakEndpointToken(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/protocol/openid-connect/token", url)

	url, err = config.GetKeycloakEndpointUserInfo(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/protocol/openid-connect/userinfo", url)

	url, err = config.GetKeycloakEndpointAuthzResourceset(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/authz/protection/resource_set", url)

	url, err = config.GetKeycloakEndpointClients(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/admin/realms/"+config.GetKeycloakRealm()+"/clients", url)

	url, err = config.GetKeycloakEndpointEntitlement(reqLong)
	require.NoError(t, err)
	require.Equal(t, "http://xyz.io/auth/realms/"+config.GetKeycloakRealm()+"/authz/entitlement/fabric8-online-platform", url)
}

func TestGetKeycloakEndpointAuthzResourcesetDevModeOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/authz/protection/resource_set", config.GetKeycloakEndpointAuthzResourceset)
}

func TestGetKeycloakEndpointAuthDevModeOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/protocol/openid-connect/auth", config.GetKeycloakEndpointAuth)
}

func TestGetKeycloakEndpointLogoutDevModeOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/protocol/openid-connect/logout", config.GetKeycloakEndpointLogout)
}

func TestGetKeycloakEndpointTokenOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/protocol/openid-connect/token", config.GetKeycloakEndpointToken)
}

func TestGetKeycloakEndpointUserInfoOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/protocol/openid-connect/userinfo", config.GetKeycloakEndpointUserInfo)
}

func TestGetKeycloakEndpointEntitlementOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/authz/entitlement/fabric8-online-platform", config.GetKeycloakEndpointEntitlement)
}

func TestGetKeycloakEndpointBrokerOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/broker", config.GetKeycloakEndpointBroker)
}

func TestGetKeycloakUserInfoEndpointOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	checkGetServiceEndpointOK(t, config.GetKeycloakDevModeURL()+"/account", config.GetKeycloakAccountEndpoint)
}

func checkGetServiceEndpointOK(t *testing.T, expectedEndpoint string, getEndpoint func(req *http.Request) (string, error)) {
	url, err := getEndpoint(reqLong)
	assert.NoError(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, expectedEndpoint, url)

	url, err = getEndpoint(reqShort)
	assert.NoError(t, err)
	// In dev mode it's always the defualt value regardless of the request
	assert.Equal(t, expectedEndpoint, url)
}

func TestGetMaxHeaderSizeUsingDefaults(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	viperValue := config.GetHeaderMaxLength()
	require.NotNil(t, viperValue)
	assert.Equal(t, int64(5000), viperValue)
}

func TestGetMaxHeaderSizeSetByEnvVaribaleOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	envName := "F8_HEADER_MAXLENGTH"
	envValue := time.Now().Unix()
	env := os.Getenv(envName)
	defer func() {
		os.Setenv(envName, env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv(envName, strconv.FormatInt(envValue, 10))
	resetConfiguration(defaultValuesConfigFilePath)

	viperValue := config.GetHeaderMaxLength()
	require.NotNil(t, viperValue)
	assert.Equal(t, envValue, viperValue)
}

func generateEnvKey(yamlKey string) string {
	return "F8_" + strings.ToUpper(strings.Replace(yamlKey, ".", "_", -1))
}

func checkGetKeycloakEndpointSetByEnvVaribaleOK(t *testing.T, envName string, getEndpoint func(req *http.Request) (string, error)) {
	envValue := uuid.NewV4().String()
	env := os.Getenv(envName)
	defer func() {
		os.Setenv(envName, env)
		resetConfiguration(defaultValuesConfigFilePath)
	}()

	os.Setenv(envName, envValue)
	resetConfiguration(defaultValuesConfigFilePath)

	url, err := getEndpoint(reqLong)
	require.NoError(t, err)
	require.Equal(t, envValue, url)

	url, err = getEndpoint(reqShort)
	require.NoError(t, err)
	require.Equal(t, envValue, url)
}
