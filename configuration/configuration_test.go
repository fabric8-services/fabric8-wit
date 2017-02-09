package configuration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
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
	resource.Require(t, resource.UnitTest)

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

/*
TestMultipleConfigurations creates configuration objects using 2 different config files
and checks whether one doesn't override the other.append
*/
func TestMultipleConfigurations(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

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

// getTestConfigMapValue accomplishes the same thing that testConfig.<PropertyName> does.
func (tc testConfig) getTestConfigMapValue(key string) interface{} {
	r := reflect.ValueOf(tc)
	fmt.Println(key)
	key = strings.Replace(strings.Title((strings.Replace(key, ".", " ", -1))), " ", "", -1)
	fmt.Println(key)

	f := reflect.Indirect(r).FieldByName(key)
	return f.Interface()
}

type testConfig struct {
	PostgresHost                 string `yaml:"postgres.host"`
	PostgresPort                 int64  `yaml:"postgres.port"`
	PostgresUser                 string `yaml:"postgres.user"`
	PostgresDatabase             string `yaml:"postgres.database"`
	PostgresPassword             string `yaml:"postgres.password"`
	PostgresSslmode              string `yaml:"postgres.sslmode"`
	PostgresConnectionMaxretries int    `yaml:"postgres.connection.maxretries"`
	PostgresConnectionRetrysleep string `yaml:"postgres.connection.retrysleep"`
	PopulateCommontypes          bool   `yaml:"populate.commontypes"`
	HTTPAddress                  string `yaml:"http.address"`
	DeveloperModeEnabled         bool   `yaml:"developer.mode.enabled"`
	KeycloakSecret               string `yaml:"keycloak.secret"`
	KeycloakClientId             string `yaml:"keycloak.client.id"`
	KeycloakEndpointAuth         string `yaml:"keycloak.endpoint.auth"`
	KeycloakEndpointToken        string `yaml:"keycloak.endpoint.token"`
	KeycloakEndpointUserinfo     string `yaml:"keycloak.endpoint.userinfo"`
	TokenPublickey               string `yaml:"token.publickey"`
	TokenPrivatekey              string `yaml:"token.privatekey"`
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
	varPostgresConnectionMaxretries = "postgres.connection.maxretries"
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

// getConfigFileMap decodes config.yaml into a go struct
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

	return config, nil
}

// configTestDescriptor.ConfigTestFunc is an interface which was assigned a method receiver.
// This function types casts the interface to an actual callable method w.r.t to a ConfigurationData object
func getConfigurationDataFunction(configurationDataFunc interface{}, configurationData *configuration.ConfigurationData) reflect.Value {
	fn := reflect.ValueOf(configurationDataFunc)
	function := fn.Call([]reflect.Value{reflect.ValueOf(configurationData)})
	actualConfigurationDataFunction := function[0]
	return actualConfigurationDataFunction
}

// Scenario 1: env variable NOT set, so we check with config.yaml's value

func testSingleConfigReadFromConfigFile(t *testing.T, c configTestDescriptor) {
	envKey := generateEnvKey(c.ConfigKey)

	realEnvValue := os.Getenv(envKey) // could be "" as well.

	os.Unsetenv(envKey)
	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	/*

		If  ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresPassword
		was passed in configTestDescriptor , then the following 2 lines
		make the following functional call

		configurationData.GetPostgresPassword()

	*/
	configurationDataFunction := getConfigurationDataFunction(c.ConfigValueFunc, configurationData)
	viperValue := configurationDataFunction.Interface()

	// now check with the config yaml file.
	testConfigFileMap, err := getConfigFileMap("")
	require.Nil(t, err)
	expectedValue := testConfigFileMap.getTestConfigMapValue(c.ConfigKey)

	// PostgresConnectionRetrySleep  was passed as a string in the config file,
	// let's parse it to the approrpriate type.
	if c.ConfigKey == varPostgresConnectionRetrySleep {
		expectedValue = cast.ToDuration(fmt.Sprintf("%v", expectedValue))
		viperValue = cast.ToDuration(fmt.Sprintf("%v", viperValue))
	}
	assert.Equal(t, reflect.TypeOf(expectedValue), reflect.TypeOf(viperValue), fmt.Sprintf("Type mismatches for %s", c.ConfigKey))
	assert.Equal(t, expectedValue, viperValue, fmt.Sprintf("Mismatch for %s", c.ConfigKey))

}

// Scenario 2 : env variable will now be SET, so now we assert with env variable and NOT With config.yaml

func testSingleConfigReadFromEnvVariable(t *testing.T, c configTestDescriptor) {
	envKey := generateEnvKey(c.ConfigKey)

	realEnvValue := os.Getenv(envKey) // could be "" as well.
	os.Unsetenv(envKey)

	defer os.Setenv(envKey, realEnvValue) // set it back before we leave.

	var testEnvValue interface{}
	if c.ConfigValueDataType == reflect.String {
		testEnvValue = tTestEnvString
	} else if c.ConfigValueDataType == reflect.Int64 {
		testEnvValue = tTestEnvInt64
	} else if c.ConfigValueDataType == reflect.Int {
		testEnvValue = tTestEnvInt
	} else if c.ConfigValueDataType == reflect.Bool {
		testEnvValue = tTestEnvBool
	}
	os.Setenv(envKey, fmt.Sprintf("%v", testEnvValue))

	/*

		If  ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresPassword
		was passed in configTestDescriptor , then the following 2 lines
		make the following functional call

		configurationData.GetPostgresPassword()

	*/
	configurationData := getConfigurationDataHandler()
	require.NotNil(t, configurationData)

	configurationDataFunction := getConfigurationDataFunction(c.ConfigValueFunc, configurationData)
	viperValue := configurationDataFunction.Interface()

	// PostgresConnectionRetrySleep  was passed as a string in the config file,
	// let's parse it to the approrpriate type.
	if c.ConfigKey == varPostgresConnectionRetrySleep {
		testEnvValue = cast.ToDuration(fmt.Sprintf("%v", testEnvValue))
		viperValue = cast.ToDuration(fmt.Sprintf("%v", viperValue))
	}

	assert.Equal(t, testEnvValue, viperValue, fmt.Sprintf("Mismatch for %s", c.ConfigKey))

}

type configTestDescriptor struct {
	ConfigKey           string       // the key(s) present in config.yaml
	ConfigValueDataType reflect.Kind // will be used to decide the test string to be used for setting env variable.
	TestValue           string       // to be set in the env variable

	// ConfigValueFunc is to be invoked on ConfigurationData object.
	// This is of type "interface{}" because the return types of every method differ.
	// the method getConfigurationDataFunction(..) is used to convert the following
	// into a callable method.
	ConfigValueFunc interface{}
}

func TestAllConfigs(t *testing.T) {

	testData := []configTestDescriptor{
		configTestDescriptor{ConfigKey: varPostgresHost, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresHost},
		configTestDescriptor{ConfigKey: varPostgresPort, ConfigValueDataType: reflect.Int64, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresPort},
		configTestDescriptor{ConfigKey: varPostgresUser, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresUser},
		configTestDescriptor{ConfigKey: varPostgresPassword, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresPassword},
		configTestDescriptor{ConfigKey: varPostgresSSLMode, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresSSLMode},
		configTestDescriptor{ConfigKey: varPostgresConnectionMaxretries, ConfigValueDataType: reflect.Int, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresConnectionMaxRetries},
		configTestDescriptor{ConfigKey: varPostgresConnectionRetrySleep, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetPostgresConnectionRetrySleep},
		configTestDescriptor{ConfigKey: varDeveloperModeEnabled, ConfigValueDataType: reflect.Bool, ConfigValueFunc: (*configuration.ConfigurationData).IsPostgresDeveloperModeEnabled},
		configTestDescriptor{ConfigKey: varPopulateCommonTypes, ConfigValueDataType: reflect.Bool, ConfigValueFunc: (*configuration.ConfigurationData).GetPopulateCommonTypes},
		configTestDescriptor{ConfigKey: varKeycloakClientID, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetKeycloakClientID},
		configTestDescriptor{ConfigKey: varKeycloakEndpointAuth, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetKeycloakEndpointAuth},
		configTestDescriptor{ConfigKey: varKeycloakEndpointToken, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetKeycloakEndpointToken},
		configTestDescriptor{ConfigKey: varKeycloakEndpointUserinfo, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetKeycloakEndpointUserinfo},
		configTestDescriptor{ConfigKey: varKeycloakSecret, ConfigValueDataType: reflect.String, ConfigValueFunc: (*configuration.ConfigurationData).GetKeycloakSecret},
	}
	for _, c := range testData {
		testSingleConfigReadFromConfigFile(t, c)
		testSingleConfigReadFromEnvVariable(t, c)
	}

}
