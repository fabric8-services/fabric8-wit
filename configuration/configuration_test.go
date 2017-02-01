package configuration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestGetConfigurationDataSucess(t *testing.T) {

	// this, in reality gets set in main
	os.Setenv("ALMIGHTY_CONFIG_FILE_PATH", "../config.yaml")
	cd, err := configuration.GetConfigurationData()
	assert.Nil(t, err)
	assert.NotNil(t, cd)
}

func TestNewConfigurationDataSuccess(t *testing.T) {

	resource.Require(t, resource.UnitTest)
	configFilePath := "../config.yaml"
	cd, err := configuration.NewConfigurationData(configFilePath)
	assert.Nil(t, err)
	assert.NotNil(t, cd)

}

func TestNewConfigurationDataFail(t *testing.T) {

	resource.Require(t, resource.UnitTest)
	configFilePath := "../invalid_config.yaml"
	_, err := configuration.NewConfigurationData(configFilePath)
	assert.NotNil(t, err)

}

func TestGetKeycloakEndpointToken(t *testing.T) {

	cd := getConfigurationDataHandler()

	t.Log(cd.GetKeycloakEndpointToken())
	assert.NotNil(t, cd.GetKeycloakEndpointToken())

}

func TestGetPostgresHost(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresHost())
}

func TestGetPostgresPort(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresPort())
}

func TestGetPostgresUser(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresUser())
}

func TestGetPostgresDatabase(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresDatabase())
}

func TestGetPostgresPassword(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresPassword())
}

func TestGetPostgresSSLMode(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresSSLMode())
}

func TestGetPostgresConnectionMaxRetries(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.Equal(t, configurationData.GetPostgresConnectionMaxRetries(), 50)
}

func TestGetPostgresConnectionRetrySleep(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresConnectionRetrySleep())
}

func TestGetPostgresConfigString(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPostgresConfigString())
}

func TestGetPopulateCommonTypes(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetPopulateCommonTypes())
}

func TestGetHTTPAddress(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetHTTPAddress())
}

func IsPostgresDeveloperModeEnabled(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.IsPostgresDeveloperModeEnabled())
}

func TestGetTokenPrivateKey(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetTokenPrivateKey())
	fmt.Println(configurationData.GetTokenPrivateKey())
}

func TestGetTokenPublicKey(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetTokenPublicKey())
}

func TestGetGithubAuthToken(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetGithubAuthToken())
}

func TestGetKeycloakSecret(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetKeycloakSecret())
}

func TestGetKeycloakClientID(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetKeycloakClientID())
}

func TestGetKeycloakEndpointAuth(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetKeycloakEndpointAuth())
}

func TestGetKeycloakEndpointUserinfo(t *testing.T) {
	configurationData := getConfigurationDataHandler()
	assert.NotNil(t, configurationData.GetKeycloakEndpointUserinfo())
}

func getConfigurationDataHandler() *configuration.ConfigurationData {
	configFilePath := "../config.yaml"
	cd, err := configuration.NewConfigurationData(configFilePath)
	if err == nil {
		return cd
	}
	return nil
}
