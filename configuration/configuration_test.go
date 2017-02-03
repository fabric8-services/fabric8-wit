package configuration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"strings"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tDefaultConfigurationFile           = "config.yaml"
	tEnvironmentVariableNameConfigFile  = "ALMIGHTY_CONFIG_FILE_PATH"
	tEnvironmentVariableValueConfigFile = "../config.yaml"
	tInvalidConfigFilePath              = "../invalid_config.yaml"
	tTestConfigFilePath                 = "../test/data/multiple_config.yaml"

	// The following will be used to set env variables for testing.
	tTestEnvString         = "ALMIGHTY_CONFIG_VALUE"
	tTestEnvInt64    int64 = 12345
	tTestEnvInt      int   = 12
	tTestEnvDuration       = "1s"
	tTestEnvBool           = false
)

func TestGetDefaultConfigurationFile(t *testing.T) {
	t.Parallel()
	assert.Equal(t, configuration.GetDefaultConfigurationFile(), tDefaultConfigurationFile)
}

func TestGetConfigurationDataSucess(t *testing.T) {
	t.Parallel()
	// this, in reality gets set in main
	os.Setenv(tEnvironmentVariableNameConfigFile, tTestConfigFilePath)
	cd, err := configuration.GetConfigurationData()
	assert.Nil(t, err)
	assert.NotNil(t, cd)
	assert.Equal(t, "mysecretpassword2", cd.GetPostgresPassword())
}

func TestNewConfigurationDataSuccess(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	cd, err := configuration.NewConfigurationData(tTestConfigFilePath)
	assert.Nil(t, err)
	assert.NotNil(t, cd)
	assert.Equal(t, "mysecretpassword2", cd.GetPostgresPassword())

}

func TestNewConfigurationDataFail(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	configFilePath := tInvalidConfigFilePath
	_, err := configuration.NewConfigurationData(configFilePath)
	assert.NotNil(t, err)

}

func TestGetKeycloakEndpointToken(t *testing.T) {
	t.Parallel()

	cd := getConfigurationDataHandler()
	require.NotNil(t, cd)

	viperValue := cd.GetKeycloakEndpointToken()

	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.KeycloakEndpointToken

	assert.Equal(t, expectedValue, viperValue)

}

func TestGetPostgresHost(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresHost)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresHost()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap(tEnvironmentVariableValueConfigFile)
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresHost

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresHost()
	assert.Equal(t, tTestEnvString, viperValue)

}

func TestGetPostgresPort(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresPort)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresPort()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresPort

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, strconv.FormatInt(tTestEnvInt64, 10))

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresPort()
	assert.Equal(t, tTestEnvInt64, viperValue)
}

func TestGetPostgresUser(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresUser)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresUser()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresUser

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresUser()
	assert.Equal(t, tTestEnvString, viperValue)

}

func TestGetPostgresDatabase(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresDatabase)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresDatabase()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresDatabase

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresDatabase()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetPostgresPassword(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresPassword)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresPassword()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresPassword

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresPassword()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetPostgresSSLMode(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresSSLMode)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresSSLMode()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresSSLMode

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresSSLMode()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetPostgresConnectionMaxRetries(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresConnectionMaxRetries)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresConnectionMaxRetries()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresConnectionMaxRetries

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, strconv.Itoa(tTestEnvInt))

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresConnectionMaxRetries()
	assert.Equal(t, tTestEnvInt, viperValue)
}

func TestGetPostgresConnectionRetrySleep(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPostgresConnectionRetrySleep)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPostgresConnectionRetrySleep()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PostgresConnectionRetrySleep

	assert.NotNil(t, viperValue)
	assert.Equal(t, cast.ToDuration(expectedValue), viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPostgresConnectionRetrySleep()
	assert.Equal(t, cast.ToDuration(tTestEnvString), viperValue)
}

func TestGetPostgresConfigString(t *testing.T) {
	t.Parallel()

	configurationData := getConfigurationDataHandler()
	// This is a derviced config parameter, not present as is, in the config file.
	assert.NotNil(t, configurationData.GetPostgresConfigString())
}

func TestGetPopulateCommonTypes(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varPopulateCommonTypes)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetPopulateCommonTypes()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.PopulateCommonTypes

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, strconv.FormatBool(tTestEnvBool))

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetPopulateCommonTypes()
	assert.Equal(t, tTestEnvBool, viperValue)
}

func TestGetHTTPAddress(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varHTTPAddress)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetHTTPAddress()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.HTTPAddress

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetHTTPAddress()
	assert.Equal(t, tTestEnvString, viperValue)
}

func IsPostgresDeveloperModeEnabled(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varDeveloperModeEnabled)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.IsPostgresDeveloperModeEnabled()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.DeveloperModeEnabled

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.IsPostgresDeveloperModeEnabled()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetTokenPrivateKey(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varTokenPrivateKey)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetTokenPrivateKey()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.TokenPrivateKey

	assert.NotNil(t, viperValue)
	assert.Equal(t, []byte(expectedValue), viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetTokenPrivateKey()
	assert.Equal(t, []byte(tTestEnvString), viperValue)
}

func TestGetTokenPublicKey(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varTokenPublicKey)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetTokenPublicKey()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.TokenPublicKey

	assert.NotNil(t, viperValue)
	assert.Equal(t, []byte(expectedValue), viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetTokenPublicKey()
	assert.Equal(t, []byte(tTestEnvString), viperValue)
}

func TestGetKeycloakSecret(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varKeycloakSecret)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetKeycloakSecret()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.KeycloakSecret

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetKeycloakSecret()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetKeycloakClientID(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varKeycloakClientID)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetKeycloakClientID()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.KeycloakClientID

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetKeycloakClientID()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetKeycloakEndpointAuth(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varKeycloakEndpointAuth)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetKeycloakEndpointAuth()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.KeycloakEndpointAuth

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetKeycloakEndpointAuth()
	assert.Equal(t, tTestEnvString, viperValue)
}

func TestGetKeycloakEndpointUserinfo(t *testing.T) {
	t.Parallel()

	envKey := generateEnvKey(varKeycloakEndpointUserinfo)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	// env variable NOT set, so we check with config.yaml's value

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	viperValue := configurationData.GetKeycloakEndpointUserinfo()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.KeycloakEndpointUserinfo

	assert.NotNil(t, viperValue)
	assert.Equal(t, expectedValue, viperValue)

	// env variable will now be SET, so now we check with env variable and NOT With config.yaml
	os.Setenv(envKey, tTestEnvString)

	configurationData = getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	require.Nil(t, err)

	viperValue = configurationData.GetKeycloakEndpointUserinfo()
	assert.Equal(t, tTestEnvString, viperValue)
}

/*
TestMultipleConfigurations creates configuration objects using 2 different config files
and checks whether one doesn't override the other.append
*/

func TestMultipleConfigurations(t *testing.T) {
	t.Parallel()

	configurationData1, err := configuration.NewConfigurationData(tEnvironmentVariableValueConfigFile)
	require.Nil(t, err)
	require.NotNil(t, configurationData1)

	configurationData2, err := configuration.NewConfigurationData(tTestConfigFilePath)
	require.Nil(t, err)
	require.NotNil(t, configurationData2)

	assert.NotEqual(t, configurationData1.GetPostgresHost, configurationData2.GetPostgresHost)
	assert.NotEqual(t, configurationData1.GetPostgresPort, configurationData2.GetPostgresPort)

}

func getConfigurationDataHandler() *configuration.ConfigurationData {

	configFilePath := tEnvironmentVariableValueConfigFile
	cd, err := configuration.NewConfigurationData(configFilePath)
	if err == nil {
		return cd
	}
	return nil
}

type testConfig struct {
	PostgresHost                 string `yaml:"postgres.host"`
	PostgresPort                 int64  `yaml:"postgres.port"`
	PostgresUser                 string `yaml:"postgres.user"`
	PostgresDatabase             string `yaml:"postgres.database"`
	PostgresPassword             string `yaml:"postgres.password"`
	PostgresSSLMode              string `yaml:"postgres.sslmode"`
	PostgresConnectionMaxRetries int    `yaml:"postgres.connection.maxretries"`
	PostgresConnectionRetrySleep string `yaml:"postgres.connection.retrysleep"`
	PopulateCommonTypes          bool   `yaml:"populate.commontypes"`
	HTTPAddress                  string `yaml:"http.address"`
	DeveloperModeEnabled         bool   `yaml:"developer.mode.enabled"`
	KeycloakSecret               string `yaml:"keycloak.secret"`
	KeycloakClientID             string `yaml:"keycloak.client.id"`
	KeycloakEndpointAuth         string `yaml:"keycloak.endpoint.auth"`
	KeycloakEndpointToken        string `yaml:"keycloak.endpoint.token"`
	KeycloakEndpointUserinfo     string `yaml:"keycloak.endpoint.userinfo"`
	TokenPublicKey               string `yaml:"token.publickey"`
	TokenPrivateKey              string `yaml:"token.privatekey"`
}

// Copy-pasted this from configuration package because they are inaccesible from
// configuration_test package. They will be consumed to determine what the
// environment variable would be
const (
	varPostgresHost                 = "postgres.host"
	varPostgresPort                 = "postgres.port"
	varPostgresUser                 = "postgres.user"
	varPostgresDatabase             = "postgres.database"
	varPostgresPassword             = "postgres.password"
	varPostgresSSLMode              = "postgres.sslmode"
	varPostgresConnectionMaxRetries = "postgres.connection.maxretries"
	varPostgresConnectionRetrySleep = "postgres.connection.retrysleep"
	varPopulateCommonTypes          = "populate.commontypes"
	varHTTPAddress                  = "http.address"
	varDeveloperModeEnabled         = "developer.mode.enabled"
	varGithubAuthToken              = "github.auth.token"
	varKeycloakSecret               = "keycloak.secret"
	varKeycloakClientID             = "keycloak.client.id"
	varKeycloakEndpointAuth         = "keycloak.endpoint.auth"
	varKeycloakEndpointToken        = "keycloak.endpoint.token"
	varKeycloakEndpointUserinfo     = "keycloak.endpoint.userinfo"
	varTokenPublicKey               = "token.publickey"
	varTokenPrivateKey              = "token.privatekey"
	defaultConfigFile               = "config.yaml"
)

func generateEnvKey(yamlKey string) string {
	return "ALMIGHTY_" + strings.ToUpper(strings.Replace(yamlKey, ".", "_", -1))
}

func TestGenerateEnvKey(t *testing.T) {
	assert.Equal(t, "ALMIGHTY_POSTGRES_HOST", generateEnvKey("postgres.host"))
}

func getConfigFileMap(configFilePath string) (testConfig, error) {
	if configFilePath == "" {
		//use default
		configFilePath = tEnvironmentVariableValueConfigFile
	}
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		// its okay to return defaults, they would eventually fail in the assert statememts
		fmt.Print(err)
		return testConfig{}, err
	}

	var config testConfig
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println(err)
		return testConfig{}, err
	}
	//fmt.Printf("Value: %#v\n", config)
	//spew.Dump(config)
	return config, nil
}
