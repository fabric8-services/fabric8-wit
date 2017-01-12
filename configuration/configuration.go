package configuration

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"
)

// String returns the current configuration as a string
func (c *ConfigurationData) String() string {
	allSettings := viper.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		panic(fmt.Errorf("Failed to marshall config to string: %s", err.Error()))
	}
	return fmt.Sprintf("%s\n", y)
}

var mapLock sync.RWMutex

// Setup sets up defaults for viper configuration options and
// overrides these values with the values from the given configuration file
// if it is not empty. Those values again are overwritten by environment
// variables.
func Setup(configFilePath string) (*ConfigurationData, error) {
	mapLock.Lock()
	defer mapLock.Unlock()

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
			return nil, fmt.Errorf("Fatal error config file: %s \n", err)
		}
	}

	configuration := NewConfigurationData()
	return &configuration, nil
}

// Constants for viper variable names. Will be used to set
// default values as well as to get each value
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
	varGithubSecret                 = "github.secret"
	varGithubClientID               = "github.client.id"
	varGithubAuthToken              = "github.auth.token"
	varTokenPublicKey               = "token.publickey"
	varTokenPrivateKey              = "token.privatekey"
)

func setConfigDefaults() {
	//---------
	// Postgres
	//---------
	viper.SetTypeByDefaultValue(true)
	viper.SetDefault(varPostgresHost, "localhost")
	viper.SetDefault(varPostgresPort, 5432)
	viper.SetDefault(varPostgresUser, "postgres")
	viper.SetDefault(varPostgresDatabase, "postgres")
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

	//-----
	// Misc
	//-----

	// Enable development related features, e.g. token generation endpoint
	viper.SetDefault(varDeveloperModeEnabled, false)

	viper.SetDefault(varPopulateCommonTypes, true)

	// Auth-related defaults
	viper.SetDefault(varTokenPublicKey, defaultTokenPublicKey)
	viper.SetDefault(varTokenPrivateKey, defaultTokenPrivateKey)
	viper.SetDefault(varGithubClientID, defaultGithubClientID)
	viper.SetDefault(varGithubSecret, defaultGithubSecret)
	viper.SetDefault(varGithubAuthToken, defaultActualToken)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func getPostgresHost() string {
	return viper.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func getPostgresPort() int64 {
	return viper.GetInt64(varPostgresPort)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func getPostgresUser() string {
	return viper.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func getPostgresDatabase() string {
	return viper.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func getPostgresPassword() string {
	return viper.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func getPostgresSSLMode() string {
	return viper.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionMaxRetries returns the number of times (as set via default, config file, or environment variable)
// alm server will attempt to open a connection to the database before it gives up
func getPostgresConnectionMaxRetries() int {
	return viper.GetInt(varPostgresConnectionMaxRetries)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func getPostgresConnectionRetrySleep() time.Duration {
	return viper.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPopulateCommonTypes returns true if the (as set via default, config file, or environment variable)
// the common work item types such as system.bug or system.feature shall be created.
func getPopulateCommonTypes() bool {
	return viper.GetBool(varPopulateCommonTypes)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the alm server binds to (e.g. "0.0.0.0:8080")
func getHTTPAddress() string {
	return viper.GetString(varHTTPAddress)
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func isPostgresDeveloperModeEnabled() bool {
	return viper.GetBool(varDeveloperModeEnabled)
}

// GetTokenPrivateKey returns the private key (as set via config file or environment variable)
// that is used to sign the authentication token.
func getTokenPrivateKey() []byte {
	return []byte(viper.GetString(varTokenPrivateKey))
}

// GetTokenPublicKey returns the public key (as set via config file or environment variable)
// that is used to decrypt the authentication token.
func getTokenPublicKey() []byte {
	return []byte(viper.GetString(varTokenPublicKey))
}

// GetGithubSecret returns the Github secret(as set via config file or environment variable)
// that is used to make authorized Github API Calls.
func getGithubSecret() string {
	return viper.GetString(varGithubSecret)
}

// GetGithubClientID returns the Github Client ID(as set via config file or environment variable)
// that is used to make authorized Github API Calls.
func getGithubClientID() string {
	return viper.GetString(varGithubClientID)
}

// GetGithubAuthToken returns the actual Github OAuth Access Token
func getGithubAuthToken() string {
	return viper.GetString(varGithubAuthToken)
}

// Auth-related defaults

// RSAPrivateKey for signing JWT Tokens
// ssh-keygen -f alm_rsa
var defaultTokenPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAnwrjH5iTSErw9xUptp6QSFoUfpHUXZ+PaslYSUrpLjw1q27O
DSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nwxBobS723okFaIkshRrf6qgtD6coT
HlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3bwTIFlLlVMiZf7XVE7P3yuOCpqkk
2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpjO5QnY+Exx98E30iEdPHZpsfNhsjh
9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbvenbR9Q59d88lbnEeHKgSMe2RQpFR
3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWmPwIDAQABAoIBAQCBCl5ZpnvprhRx
BVTA/Upnyd7TCxNZmzrME+10Gjmz79pD7DV25ejsu/taBYUxP6TZbliF3pggJOv6
UxomTB4znlMDUz0JgyjUpkyril7xVQ6XRAPbGrS1f1Def+54MepWAn3oGeqASb3Q
bAj0Yl12UFTf+AZmkhQpUKk/wUeN718EIY4GRHHQ6ykMSqCKvdnVbMyb9sIzbSTl
v+l1nQFnB/neyJq6P0Q7cxlhVj03IhYj/AxveNlKqZd2Ih3m/CJo0Abtwhx+qHZp
cCBrYj7VelEaGARTmfoIVoGxFGKZNCcNzn7R2ic7safxXqeEnxugsAYX/UmMoq1b
vMYLcaLRAoGBAMqMbbgejbD8Cy6wa5yg7XquqOP5gPdIYYS88TkQTp+razDqKPIU
hPKetnTDJ7PZleOLE6eJ+dQJ8gl6D/dtOsl4lVRy/BU74dk0fYMiEfiJMYEYuAU0
MCramo3HAeySTP8pxSLFYqJVhcTpL9+NQgbpJBUlx5bLDlJPl7auY077AoGBAMkD
UpJRIv/0gYSz5btVheEyDzcqzOMZUVsngabH7aoQ49VjKrfLzJ9WznzJS5gZF58P
vB7RLuIA8m8Y4FUwxOr4w9WOevzlFh0gyzgNY4gCwrzEryOZqYYqCN+8QLWfq/hL
+gYFYpEW5pJ/lAy2i8kPanC3DyoqiZCsUmlg6JKNAoGBAIdCkf6zgKGhHwKV07cs
DIqx2p0rQEFid6UB3ADkb+zWt2VZ6fAHXeT7shJ1RK0o75ydgomObWR5I8XKWqE7
s1dZjDdx9f9kFuVK1Upd1SxoycNRM4peGJB1nWJydEl8RajcRwZ6U+zeOc+OfWbH
WUFuLadlrEx5212CQ2k+OZlDAoGAdsH2w6kZ83xCFOOv41ioqx5HLQGlYLpxfVg+
2gkeWa523HglIcdPEghYIBNRDQAuG3RRYSeW+kEy+f4Jc2tHu8bS9FWkRcsWoIji
ZzBJ0G5JHPtaub6sEC6/ZWe0F1nJYP2KLop57FxKRt0G2+fxeA0ahpMwa2oMMiQM
4GM3pHUCgYEAj2ZjjsF2MXYA6kuPUG1vyY9pvj1n4fyEEoV/zxY1k56UKboVOtYr
BA/cKaLPqUF+08Tz/9MPBw51UH4GYfppA/x0ktc8998984FeIpfIFX6I2U9yUnoQ
OCCAgsB8g8yTB4qntAYyfofEoDiseKrngQT5DSdxd51A/jw7B8WyBK8=
-----END RSA PRIVATE KEY-----`

// RSAPublicKey for verifying JWT Tokens
// openssl rsa -in alm_rsa -pubout -out alm_rsa.pub
var defaultTokenPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnwrjH5iTSErw9xUptp6Q
SFoUfpHUXZ+PaslYSUrpLjw1q27ODSFwmhV4+dAaTMO5chFv/kM36H3ZOyA146nw
xBobS723okFaIkshRrf6qgtD6coTHlVUSBTAcwKEjNn4C9jtEpyOl+eSgxhMzRH3
bwTIFlLlVMiZf7XVE7P3yuOCpqkk2rdYVSpQWQWKU+ZRywJkYcLwjEYjc70AoNpj
O5QnY+Exx98E30iEdPHZpsfNhsjh9Z7IX5TrMYgz7zBTw8+niO/uq3RBaHyIhDbv
enbR9Q59d88lbnEeHKgSMe2RQpFR3rxFRkc/64Rn/bMuL/ptNowPqh1P+9GjYzWm
PwIDAQAB
-----END PUBLIC KEY-----`

var defaultGithubClientID = "875da0d2113ba0a6951d"
var defaultGithubSecret = "2fe6736e90a9283036a37059d75ac0c82f4f5288"

// Github doesnot allow commiting actual OAuth tokens no matter how less priviledge the token has
var camouflagedAccessToken = "751e16a8b39c0985066-AccessToken-4871777f2c13b32be8550"

// ActualToken is actual OAuth access token of github
var defaultActualToken = strings.Split(camouflagedAccessToken, "-AccessToken-")[0] + strings.Split(camouflagedAccessToken, "-AccessToken-")[1]

func GetConfigFilePath() string {
	path, _ := os.LookupEnv("ALMIGHTY_CONFIG_FILE_PATH")
	return path
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresHost() string {
	return c.postgresHost
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPort() int64 {
	return c.postgresPort
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresUser() string {
	return c.postgresUser
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresDatabase() string {
	return c.postgresDatabase
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPassword() string {
	return c.postgresPassword
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresSSLMode() string {
	return c.postgresSSLMode
}

// GetPostgresConnectionMaxRetries returns the number of times (as set via default, config file, or environment variable)
// alm server will attempt to open a connection to the database before it gives up
func (c *ConfigurationData) GetPostgresConnectionMaxRetries() int {
	return c.postgresConnectionMaxRetries
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *ConfigurationData) GetPostgresConnectionRetrySleep() time.Duration {
	return c.postgresConnectionRetrySleep
}

// GetPopulateCommonTypes returns true if the (as set via default, config file, or environment variable)
// the common work item types such as system.bug or system.feature shall be created.
func (c *ConfigurationData) GetPopulateCommonTypes() bool {
	return c.populateCommonTypes
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the alm server binds to (e.g. "0.0.0.0:8080")
func (c *ConfigurationData) GetHTTPAddress() string {
	return c.httpAddress
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *ConfigurationData) IsPostgresDeveloperModeEnabled() bool {
	return c.developerModeEnabled
}

// GetTokenPrivateKey returns the private key (as set via config file or environment variable)
// that is used to sign the authentication token.
func (c *ConfigurationData) GetTokenPrivateKey() []byte {
	return c.tokenPrivateKey
}

// GetTokenPublicKey returns the public key (as set via config file or environment variable)
// that is used to decrypt the authentication token.
func (c *ConfigurationData) GetTokenPublicKey() []byte {
	return c.tokenPublicKey
}

// GetGithubSecret returns the Github secret(as set via config file or environment variable)
// that is used to make authorized Github API Calls.
func (c *ConfigurationData) GetGithubSecret() string {
	return c.githubSecret
}

// GetGithubClientID returns the Github Client ID(as set via config file or environment variable)
// that is used to make authorized Github API Calls.
func (c *ConfigurationData) GetGithubClientID() string {
	return c.githubClientID
}

// GetGithubAuthToken returns the actual Github OAuth Access Token
func (c *ConfigurationData) GetGithubAuthToken() string {
	return c.githubAuthToken
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *ConfigurationData) GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s DB.name=%s sslmode=%s",
		getPostgresHost(),
		getPostgresPort(),
		getPostgresUser(),
		getPostgresPassword(),
		getPostgresDatabase(),
		getPostgresSSLMode(),
	)
}

type ConfigurationData struct {
	// List of configuration items
	postgresHost                 string
	postgresPort                 int64
	postgresUser                 string
	postgresDatabase             string
	postgresPassword             string
	postgresSSLMode              string
	postgresConnectionMaxRetries int
	postgresConnectionRetrySleep time.Duration
	populateCommonTypes          bool
	httpAddress                  string
	developerModeEnabled         bool
	githubSecret                 string
	githubClientID               string
	githubAuthToken              string
	tokenPublicKey               []byte
	tokenPrivateKey              []byte
}

// NewConfigurationData returns an instance of ConfigurationData which contains the configuration
// information at the time of invokation of this method.
func NewConfigurationData() ConfigurationData {
	return ConfigurationData{
		postgresHost:                 getPostgresHost(),
		postgresPort:                 getPostgresPort(),
		postgresUser:                 getPostgresUser(),
		postgresDatabase:             getPostgresDatabase(),
		postgresPassword:             getPostgresPassword(),
		postgresSSLMode:              getPostgresSSLMode(),
		postgresConnectionMaxRetries: getPostgresConnectionMaxRetries(),
		postgresConnectionRetrySleep: getPostgresConnectionRetrySleep(),
		populateCommonTypes:          getPopulateCommonTypes(),
		httpAddress:                  getHTTPAddress(),
		developerModeEnabled:         isPostgresDeveloperModeEnabled(),
		githubSecret:                 getGithubSecret(),
		githubClientID:               getGithubClientID(),
		githubAuthToken:              getGithubAuthToken(),
		tokenPublicKey:               getTokenPublicKey(),
		tokenPrivateKey:              getTokenPrivateKey(),
	}
}

// ConfigurationLookup defines what is needed for providing all the relevant configuration
type ConfigurationLookup interface {
	GetPostgresHost() string
	GetPostgresPort() string
	GetPostgresUser() string
	GetPostgresDatabase() string
	GetPostgresPassword() string
	GetPostgresSSLMode() string
	GetPostgresConfigString() string
	GetPostgresConnectionMaxRetries() int
	GetPostgresConnectionRetrySleep() time.Duration
	GetPopulateCommonTypes() bool
	GetHTTPAddress() string
	GetDeveloperModeEnabled() bool
	GetGithubSecret() string
	GetGithubClientID() string
	GetGithubAuthToken() string
	GetTokenPublicKey() []byte
	GetTokenPrivateKey() []byte
	String() string
}
