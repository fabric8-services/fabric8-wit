package configuration

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

// String returns the current configuration as a string
func String() string {
	allSettings := viper.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		panic(fmt.Errorf("Failed to marshall config to string: %s", err.Error()))
	}
	return fmt.Sprintf("%s\n", y)
}

// Setup sets up defaults for viper configuration options and
// overrides these values with the values from the given configuration file
// if it is not empty. Those values again are overwritten by environment
// variables.
func Setup(configFilePath string) error {
	viper.Reset()

	// Expect environment variables to be prefix with "ALMIGHTY_".
	viper.SetEnvPrefix("ALMIGHTY")

	// Automatically map environment variables to viper values
	viper.AutomaticEnv()

	// To override nested variables through environment variables, we
	// need to make sure that we don't have to use dots (".") inside the
	// environment variable names.
	// To override foo.bar you need to set ALM_FOO_BAR
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetTypeByDefaultValue(true)
	setConfigDefaults()

	// Read the config
	// Explicitly specify which file to load config from
	if configFilePath != "" {
		viper.SetConfigFile(configFilePath)
		viper.SetConfigType("yaml")
		err := viper.ReadInConfig() // Find and read the config file
		if err != nil {             // Handle errors reading the config file
			return fmt.Errorf("Fatal error config file: %s \n", err)
		}
	}

	return nil
}

// Constants for viper variable names. Will be used to set
// default values as well as to get each value
const (
	varPostgresHost                 = "postgres.host"
	varPostgresPort                 = "postgres.port"
	varPostgresUser                 = "postgres.user"
	varPostgresPassword             = "postgres.password"
	varPostgresSSLMode              = "postgres.sslmode"
	varPostgresConnectionMaxRetries = "postgres.connection.maxretries"
	varPostgresConnectionRetrySleep = "postgres.connection.retrysleep"
	varHTTPAddress                  = "http.address"
	varDeveloperModeEnabled         = "developer.mode.enabled"
)

func setConfigDefaults() {
	//---------
	// Postgres
	//---------
	viper.SetTypeByDefaultValue(true)
	viper.SetDefault(varPostgresHost, "localhost")
	viper.SetDefault(varPostgresPort, 5432)
	viper.SetDefault(varPostgresUser, "postgres")
	viper.SetDefault(varPostgresPassword, "mysecretpassword")
	viper.SetDefault(varPostgresSSLMode, "disable")
	// The number of times alm server will attempt to open a connection to the database before it gives up
	viper.SetDefault(varPostgresConnectionMaxRetries, 50)
	// Number of seconds to wait before trying to connect again
	viper.SetDefault(varPostgresConnectionRetrySleep, time.Duration(time.Second))

	//-----
	// HTTP
	//-----
	viper.SetDefault(varHTTPAddress, "0.0.0.0:8080")

	// Enable development related features, e.g. token generation endpoint
	viper.SetDefault(varDeveloperModeEnabled, false)
}

// GetPostgresHost returns the postgres host as set via config file or environment variable
func GetPostgresHost() string {
	return viper.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via config file or environment variable
func GetPostgresPort() int64 {
	return viper.GetInt64(varPostgresPort)
}

// GetPostgresUser returns the postgres user as set via config file or environment variable
func GetPostgresUser() string {
	return viper.GetString(varPostgresUser)
}

// GetPostgresPassword returns the postgres password as set via config file or environment variable
func GetPostgresPassword() string {
	return viper.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via config file or environment variable
func GetPostgresSSLMode() string {
	return viper.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionMaxRetries returns the number of times (as set via config file or
// environment variable) alm server will attempt to open a connection to the database before it gives up
func GetPostgresConnectionMaxRetries() int {
	return viper.GetInt(varPostgresConnectionMaxRetries)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via config file or
// environment variable) to wait before trying to connect again
func GetPostgresConnectionRetrySleep() time.Duration {
	return viper.GetDuration(varPostgresConnectionRetrySleep)
}

// GetHTTPAddress returns the HTTP address (as set via config file or environment variable)
// that the alm server binds to (e.g. "0.0.0.0:8080")
func GetHTTPAddress() string {
	return viper.GetString(varHTTPAddress)
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via config file or
// environment variable, e.g. token generation endpoint are enabled
func IsPostgresDeveloperModeEnabled() bool {
	return viper.GetBool(varDeveloperModeEnabled)
}
