package configuration

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	"github.com/spf13/viper"
)

// String returns the current configuration as a string
func String() string {
	allSettings := viper.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"settings": allSettings,
			"err":      err,
		}).Panicln("Failed to marshall config to string")
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
	varPostgresDatabase             = "postgres.database"
	varPostgresPassword             = "postgres.password"
	varPostgresSSLMode              = "postgres.sslmode"
	varPostgresConnectionTimeout    = "postgres.connection.timeout"
	varPostgresConnectionRetrySleep = "postgres.connection.retrysleep"
	varPopulateCommonTypes          = "populate.commontypes"
	varHTTPAddress                  = "http.address"
	varDeveloperModeEnabled         = "developer.mode.enabled"
	varGithubAuthToken              = "github.auth.token"
	varKeycloakSecret               = "keycloak.secret"
	varKeycloakClientID             = "keycloak.client.id"
	varKeycloakEndpointAuth         = "keycloak.endpoint.auth"
	varKeycloakEndpointToken        = "keycloak.endpoint.token"
	varKeycloakDomainPrefix         = "keycloak.domain.prefix"
	varKeycloakRealm                = "keycloak.realm"
	varKeycloakEndpointUserinfo     = "keycloak.endpoint.userinfo"
	varKeycloakTesUserName          = "keycloak.testuser.name"
	varKeycloakTesUserSecret        = "keycloak.testuser.secret"
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
	viper.SetDefault(varPostgresConnectionTimeout, 5)

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
	viper.SetDefault(varKeycloakClientID, defaultKeycloakClientID)
	viper.SetDefault(varKeycloakSecret, defaultKeycloakSecret)
	viper.SetDefault(varGithubAuthToken, defaultActualToken)
	viper.SetDefault(varKeycloakDomainPrefix, defaultKeycloakDomainPrefix)
	viper.SetDefault(varKeycloakRealm, defaultKeycloakRealm)
	viper.SetDefault(varKeycloakTesUserName, defaultKeycloakTesUserName)
	viper.SetDefault(varKeycloakTesUserSecret, defaultKeycloakTesUserSecret)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func GetPostgresHost() string {
	return viper.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func GetPostgresPort() int64 {
	return viper.GetInt64(varPostgresPort)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func GetPostgresUser() string {
	return viper.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func GetPostgresDatabase() string {
	return viper.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func GetPostgresPassword() string {
	return viper.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func GetPostgresSSLMode() string {
	return viper.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func GetPostgresConnectionTimeout() int64 {
	return viper.GetInt64(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func GetPostgresConnectionRetrySleep() time.Duration {
	return viper.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		GetPostgresHost(),
		GetPostgresPort(),
		GetPostgresUser(),
		GetPostgresPassword(),
		GetPostgresDatabase(),
		GetPostgresSSLMode(),
		GetPostgresConnectionTimeout(),
	)
}

// GetPopulateCommonTypes returns true if the (as set via default, config file, or environment variable)
// the common work item types such as bug or feature shall be created.
func GetPopulateCommonTypes() bool {
	return viper.GetBool(varPopulateCommonTypes)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the alm server binds to (e.g. "0.0.0.0:8080")
func GetHTTPAddress() string {
	return viper.GetString(varHTTPAddress)
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func IsPostgresDeveloperModeEnabled() bool {
	return viper.GetBool(varDeveloperModeEnabled)
}

// GetTokenPrivateKey returns the private key (as set via config file or environment variable)
// that is used to sign the authentication token.
func GetTokenPrivateKey() []byte {
	return []byte(viper.GetString(varTokenPrivateKey))
}

// GetTokenPublicKey returns the public key (as set via config file or environment variable)
// that is used to decrypt the authentication token.
func GetTokenPublicKey() []byte {
	return []byte(viper.GetString(varTokenPublicKey))
}

// GetGithubAuthToken returns the actual Github OAuth Access Token
func GetGithubAuthToken() string {
	return viper.GetString(varGithubAuthToken)
}

// GetKeycloakSecret returns the keycloak client secret (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func GetKeycloakSecret() string {
	return viper.GetString(varKeycloakSecret)
}

// GetKeycloakClientID returns the keycloak client ID (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func GetKeycloakClientID() string {
	return viper.GetString(varKeycloakClientID)
}

// GetKeycloakDomainPrefix returns the domain prefix which should be used in all Keycloak requests
func GetKeycloakDomainPrefix() string {
	return viper.GetString(varKeycloakDomainPrefix)
}

// GetKeycloakRealm returns the keyclaok realm name
func GetKeycloakRealm() string {
	return viper.GetString(varKeycloakRealm)
}

// GetKeycloakTestUserName returns the keycloak test user name used to obtain a test token (as set via config file or environment variable)
func GetKeycloakTestUserName() string {
	return viper.GetString(varKeycloakTesUserName)
}

// GetKeycloakTestUserSecret returns the keycloak test user password used to obtain a test token (as set via config file or environment variable)
func GetKeycloakTestUserSecret() string {
	return viper.GetString(varKeycloakTesUserSecret)
}

// GetKeycloakEndpointAuth returns the keycloak auth endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
func GetKeycloakEndpointAuth(req *goa.RequestData) (string, error) {
	return getKeycloakEndpoing(req, varKeycloakEndpointAuth, devModeKeycloakEndpointAuth, "auth")
}

// GetKeycloakEndpointToken returns the keycloak token endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
func GetKeycloakEndpointToken(req *goa.RequestData) (string, error) {
	return getKeycloakEndpoing(req, varKeycloakEndpointToken, devModeKeycloakEndpointToken, "token")
}

// GetKeycloakEndpointUserInfo returns the keycloak userinfo endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
func GetKeycloakEndpointUserInfo(req *goa.RequestData) (string, error) {
	return getKeycloakEndpoing(req, varKeycloakEndpointUserinfo, devModeKeycloakEndpointUserinfo, "userinfo")
}

func getKeycloakEndpoing(req *goa.RequestData, endpointVarName string, devModeEndpoint string, pathSufix string) (string, error) {
	if viper.IsSet(endpointVarName) {
		return viper.GetString(endpointVarName), nil
	}
	if IsPostgresDeveloperModeEnabled() {
		return devModeEndpoint, nil
	}
	endpoint, err := getKeycloakURL(req, openIDConnectPath(pathSufix))
	if err != nil {
		return "", err
	}
	viper.Set(endpointVarName, endpoint) // Set the variable, so, we don't have to recalculate it again the next time
	return endpoint, nil
}

func openIDConnectPath(suffix string) string {
	return "auth/realms/" + GetKeycloakRealm() + "/protocol/openid-connect/" + suffix
}

func getKeycloakURL(req *goa.RequestData, path string) (string, error) {
	scheme := "http"
	if req.TLS != nil { // isHTTPS
		scheme = "https"
	}
	currentHost := req.Host
	var newHost string
	var err error
	if currentHost == "demo.api.almighty.io" {
		// demo.api.almighty.io doesn't follow the service name convention <serviceName>.<domain>
		// The correct name would be something like API.demo.almighty.io which is to be converted to SSO.demo.almighty.io
		// So, we need to treat it as an exception
		newHost = "sso.demo.almighty.io"
	} else {
		newHost, err = rest.ReplaceDomainPrefix(currentHost, GetKeycloakDomainPrefix())
		if err != nil {
			return "", err
		}
	}
	newURL := fmt.Sprintf("%s://%s/%s", scheme, newHost, path)

	return newURL, nil
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
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAiRd6pdNjiwQFH2xmNugn
TkVhkF+TdJw19Kpj3nRtsoUe4/6gIureVi7FWqcb+2t/E0dv8rAAs6vl+d7roz3R
SkAzBjPxVW5+hi5AJjUbAxtFX/aYJpZePVhK0Dv8StCPSv9GC3T6bUSF3q3E9R9n
G1SZFkN9m2DhL+45us4THzX2eau6s0bISjAUqEGNifPyYYUzKVmXmHS9fiZJR61h
6TulPwxv68DUSk+7iIJvJfQ3lH/XNWlxWNMMehetcmdy8EDR2IkJCCAbjx9yxgKV
JXdQ7zylRlpaLopock0FGiZrJhEaAh6BGuaoUWLiMEvqrLuyZnJYEg9f/vyxUJSD
JwIDAQAB
-----END PUBLIC KEY-----`

var defaultKeycloakClientID = "fabric8-online-platform"
var defaultKeycloakSecret = "08a8bcd1-f362-446a-9d2b-d34b8d464185"

var defaultKeycloakDomainPrefix = "sso"
var defaultKeycloakRealm = "demo"

// Github does not allow committing actual OAuth tokens no matter how less privilege the token has
var camouflagedAccessToken = "751e16a8b39c0985066-AccessToken-4871777f2c13b32be8550"

// ActualToken is actual OAuth access token of github
var defaultActualToken = strings.Split(camouflagedAccessToken, "-AccessToken-")[0] + strings.Split(camouflagedAccessToken, "-AccessToken-")[1]

var defaultKeycloakTesUserName = "testuser"
var defaultKeycloakTesUserSecret = "testuser"

// Keycloak URLs to be used in dev mode. Can be overridden by setting up keycloak.endpoint.*
var devModeKeycloakEndpointAuth = "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/auth"
var devModeKeycloakEndpointToken = "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/token"
var devModeKeycloakEndpointUserinfo = "http://sso.demo.almighty.io/auth/realms/demo/protocol/openid-connect/userinfo"
