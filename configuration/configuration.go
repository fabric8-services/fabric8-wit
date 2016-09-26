package configuration

import (
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

const (
	// DefaultConfigFilePath is the path to the configuration file that is used if no other
	// file is specified via the -config switch or via the ALMIGHTY_CONFIG_FILE_PATH environment
	// variable.
	DefaultConfigFilePath = "config.yaml"
)

// GetConfiguration returns the current configuration as a string
func GetConfiguration() string {
	allSettings := viper.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		panic(fmt.Errorf("Failed to marshall config to string: %s", err.Error()))
	}
	return fmt.Sprintf("%s\n", y)
}

// SetupConfiguration sets up defaults for viper configuration options and
// overrides these values with the values from the given configuration file
// if it is not empty. Those values again are overwritten by environment
// variables.
func SetupConfiguration(configFilePath string) error {
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

func setConfigDefaults() {
	//---------
	// Postgres
	//---------
	viper.SetTypeByDefaultValue(true)
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "mysecretpassword")
	viper.SetDefault("postgres.sslmode", "disable")
	// The number of times alm server will attempt to open a connection to the database before it gives up
	viper.SetDefault("postgres.connection.maxretries", 50)
	// Number of seconds to wait before trying to connect again
	viper.SetDefault("postgres.connection.retrysleep", time.Duration(time.Second))

	//-----
	// HTTP
	//-----
	viper.SetDefault("http.address", "0.0.0.0:8080")

	// Enable development related features, e.g. token generation endpoint
	viper.SetDefault("developer.mode.enabled", false)
}
