package configuration

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/spf13/viper"
)

// String returns the current configuration as a string
func (c *ConfigurationData) String() string {
	allSettings := c.v.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"settings": allSettings,
			"err":      err,
		}).Panicln("Failed to marshall config to string")
	}
	return fmt.Sprintf("%s\n", y)
}

const (
	// Constants for viper variable names. Will be used to set
	// default values as well as to get each value

	varPostgresHost                 = "postgres.host"
	varPostgresPort                 = "postgres.port"
	varPostgresUser                 = "postgres.user"
	varPostgresDatabase             = "postgres.database"
	varPostgresPassword             = "postgres.password"
	varPostgresSSLMode              = "postgres.sslmode"
	varPostgresConnectionTimeout    = "postgres.connection.timeout"
	varPostgresTransactionTimeout   = "postgres.transaction.timeout"
	varPostgresConnectionRetrySleep = "postgres.connection.retrysleep"
	varPostgresConnectionMaxIdle    = "postgres.connection.maxidle"
	varPostgresConnectionMaxOpen    = "postgres.connection.maxopen"
	varFeatureWorkitemRemote        = "feature.workitem.remote"
	varPopulateCommonTypes          = "populate.commontypes"
	varHTTPAddress                  = "http.address"
	varDeveloperModeEnabled         = "developer.mode.enabled"
	varAuthDomainPrefix             = "auth.domain.prefix"
	varAuthURL                      = "auth.url"
	varAuthorizationEnabled         = "authz.enabled"
	varGithubAuthToken              = "github.auth.token"
	varKeycloakSecret               = "keycloak.secret"
	varKeycloakClientID             = "keycloak.client.id"
	varKeycloakDomainPrefix         = "keycloak.domain.prefix"
	varKeycloakRealm                = "keycloak.realm"
	varKeycloakTesUserName          = "keycloak.testuser.name"
	varKeycloakTesUserSecret        = "keycloak.testuser.secret"
	varKeycloakTesUser2Name         = "keycloak.testuser2.name"
	varKeycloakTesUser2Secret       = "keycloak.testuser2.secret"
	varKeycloakURL                  = "keycloak.url"
	varAuthNotApprovedRedirect      = "auth.notapproved.redirect"
	varHeaderMaxLength              = "header.maxlength"

	// cache control settings for a list of resources
	varCacheControlWorkItems         = "cachecontrol.workitems"
	varCacheControlWorkItemTypes     = "cachecontrol.workitemtypes"
	varCacheControlWorkItemLinks     = "cachecontrol.workitemLinks"
	varCacheControlWorkItemLinkTypes = "cachecontrol.workitemlinktypes"
	varCacheControlSpaces            = "cachecontrol.spaces"
	varCacheControlIterations        = "cachecontrol.iterations"
	varCacheControlAreas             = "cachecontrol.areas"
	varCacheControlLabels            = "cachecontrol.labels"
	varCacheControlComments          = "cachecontrol.comments"
	varCacheControlFilters           = "cachecontrol.filters"
	varCacheControlUsers             = "cachecontrol.users"
	varCacheControlCollaborators     = "cachecontrol.collaborators"

	// cache control settings for a single resource
	varCacheControlUser             = "cachecontrol.user"
	varCacheControlWorkItem         = "cachecontrol.workitem"
	varCacheControlWorkItemType     = "cachecontrol.workitemtype"
	varCacheControlWorkItemLink     = "cachecontrol.workitemLink"
	varCacheControlWorkItemLinkType = "cachecontrol.workitemlinktype"
	varCacheControlSpace            = "cachecontrol.space"
	varCacheControlIteration        = "cachecontrol.iteration"
	varCacheControlArea             = "cachecontrol.area"
	varCacheControlLabel            = "cachecontrol.label"
	varCacheControlComment          = "cachecontrol.comment"

	defaultConfigFile           = "config.yaml"
	varOpenshiftTenantMasterURL = "openshift.tenant.masterurl"
	varCheStarterURL            = "chestarterurl"
	varValidRedirectURLs        = "redirect.valid"
	varLogLevel                 = "log.level"
	varLogJSON                  = "log.json"
	varTenantServiceURL         = "tenant.serviceurl"
	varNotificationServiceURL   = "notification.serviceurl"
)

// ConfigurationData encapsulates the Viper configuration object which stores the configuration data in-memory.
type ConfigurationData struct {
	v               *viper.Viper
	tokenPublicKey  *rsa.PublicKey
	tokenPrivateKey *rsa.PrivateKey
}

// NewConfigurationData creates a configuration reader object using a configurable configuration file path
func NewConfigurationData(configFilePath string) (*ConfigurationData, error) {
	c := ConfigurationData{
		v: viper.New(),
	}
	c.v.SetEnvPrefix("F8")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	if configFilePath != "" {
		c.v.SetConfigType("yaml")
		c.v.SetConfigFile(configFilePath)
		err := c.v.ReadInConfig() // Find and read the config file
		if err != nil {           // Handle errors reading the config file
			return nil, errors.Errorf("Fatal error config file: %s \n", err)
		}
	}
	return &c, nil
}

func getConfigFilePath() string {
	// This was either passed as a env var Or, set inside main.go from --config
	envConfigPath, ok := os.LookupEnv("F8_CONFIG_FILE_PATH")
	if !ok {
		return ""
	}
	return envConfigPath
}

// GetDefaultConfigurationFile returns the default configuration file.
func (c *ConfigurationData) GetDefaultConfigurationFile() string {
	return defaultConfigFile
}

// GetConfigurationData is a wrapper over NewConfigurationData which reads configuration file path
// from the environment variable.
func GetConfigurationData() (*ConfigurationData, error) {
	cd, err := NewConfigurationData(getConfigFilePath())
	return cd, err
}

func (c *ConfigurationData) setConfigDefaults() {
	//---------
	// Postgres
	//---------
	c.v.SetTypeByDefaultValue(true)
	c.v.SetDefault(varPostgresHost, "localhost")
	c.v.SetDefault(varPostgresPort, 5432)
	c.v.SetDefault(varPostgresUser, "postgres")
	c.v.SetDefault(varPostgresDatabase, "postgres")
	c.v.SetDefault(varPostgresPassword, "mysecretpassword")
	c.v.SetDefault(varPostgresSSLMode, "disable")
	c.v.SetDefault(varPostgresConnectionTimeout, 5)
	c.v.SetDefault(varPostgresConnectionMaxIdle, -1)
	c.v.SetDefault(varPostgresConnectionMaxOpen, -1)

	// Number of seconds to wait before trying to connect again
	c.v.SetDefault(varPostgresConnectionRetrySleep, time.Duration(time.Second))

	// Timeout of a transaction in minutes
	c.v.SetDefault(varPostgresTransactionTimeout, time.Duration(5*time.Minute))

	//-----
	// HTTP
	//-----
	c.v.SetDefault(varHTTPAddress, "0.0.0.0:8080")
	c.v.SetDefault(varHeaderMaxLength, defaultHeaderMaxLength)

	//-----
	// Misc
	//-----

	// Enable development related features, e.g. token generation endpoint
	c.v.SetDefault(varDeveloperModeEnabled, false)

	c.v.SetDefault(varLogLevel, defaultLogLevel)

	c.v.SetDefault(varPopulateCommonTypes, true)

	// Auth-related defaults
	c.v.SetDefault(varAuthURL, devModeAuthURL)
	c.v.SetDefault(varAuthDomainPrefix, "auth")
	c.v.SetDefault(varKeycloakClientID, defaultKeycloakClientID)
	c.v.SetDefault(varKeycloakSecret, defaultKeycloakSecret)
	c.v.SetDefault(varGithubAuthToken, defaultActualToken)
	c.v.SetDefault(varKeycloakDomainPrefix, defaultKeycloakDomainPrefix)
	c.v.SetDefault(varKeycloakTesUserName, defaultKeycloakTesUserName)
	c.v.SetDefault(varKeycloakTesUserSecret, defaultKeycloakTesUserSecret)

	// HTTP Cache-Control/max-age default for a list of resources
	c.v.SetDefault(varCacheControlWorkItems, "max-age=2") // very short life in cache, to allow for quick, repetitive updates.
	c.v.SetDefault(varCacheControlWorkItemTypes, "max-age=2")
	c.v.SetDefault(varCacheControlWorkItemLinks, "max-age=2")
	c.v.SetDefault(varCacheControlWorkItemLinkTypes, "max-age=2")
	c.v.SetDefault(varCacheControlSpaces, "max-age=2")
	c.v.SetDefault(varCacheControlIterations, "max-age=2")
	c.v.SetDefault(varCacheControlAreas, "max-age=2")
	c.v.SetDefault(varCacheControlComments, "max-age=2")
	c.v.SetDefault(varCacheControlFilters, "max-age=86400")
	c.v.SetDefault(varCacheControlUsers, "max-age=2")
	c.v.SetDefault(varCacheControlCollaborators, "max-age=2")

	// Cache control values for a single resource
	c.v.SetDefault(varCacheControlWorkItem, "private,max-age=2")
	c.v.SetDefault(varCacheControlWorkItemType, "private,max-age=120")
	c.v.SetDefault(varCacheControlWorkItemLink, "private,max-age=120")
	c.v.SetDefault(varCacheControlWorkItemLinkType, "private,max-age=120")
	c.v.SetDefault(varCacheControlSpace, "private,max-age=120")
	c.v.SetDefault(varCacheControlIteration, "private,max-age=2")
	c.v.SetDefault(varCacheControlArea, "private,max-age=120")
	c.v.SetDefault(varCacheControlComment, "private,max-age=120")
	// data returned from '/api/user' must not be cached by intermediate proxies,
	// but can only be kept in the client's local cache.
	c.v.SetDefault(varCacheControlUser, "private,max-age=120")

	// Features
	c.v.SetDefault(varFeatureWorkitemRemote, true)

	c.v.SetDefault(varKeycloakTesUser2Name, defaultKeycloakTesUser2Name)
	c.v.SetDefault(varKeycloakTesUser2Secret, defaultKeycloakTesUser2Secret)
	c.v.SetDefault(varOpenshiftTenantMasterURL, defaultOpenshiftTenantMasterURL)
	c.v.SetDefault(varCheStarterURL, defaultCheStarterURL)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresHost() string {
	return c.v.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPort() int64 {
	return c.v.GetInt64(varPostgresPort)
}

// GetFeatureWorkitemRemote returns true if remote Work Item feaute is enabled
func (c *ConfigurationData) GetFeatureWorkitemRemote() bool {
	return c.v.GetBool(varFeatureWorkitemRemote)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresUser() string {
	return c.v.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresDatabase() string {
	return c.v.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresPassword() string {
	return c.v.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresSSLMode() string {
	return c.v.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func (c *ConfigurationData) GetPostgresConnectionTimeout() int64 {
	return c.v.GetInt64(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *ConfigurationData) GetPostgresConnectionRetrySleep() time.Duration {
	return c.v.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresTransactionTimeout returns the number of minutes to timeout a transaction
func (c *ConfigurationData) GetPostgresTransactionTimeout() time.Duration {
	return c.v.GetDuration(varPostgresTransactionTimeout)
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior
func (c *ConfigurationData) GetPostgresConnectionMaxIdle() int {
	return c.v.GetInt(varPostgresConnectionMaxIdle)
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior
func (c *ConfigurationData) GetPostgresConnectionMaxOpen() int {
	return c.v.GetInt(varPostgresConnectionMaxOpen)
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *ConfigurationData) GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.GetPostgresHost(),
		c.GetPostgresPort(),
		c.GetPostgresUser(),
		c.GetPostgresPassword(),
		c.GetPostgresDatabase(),
		c.GetPostgresSSLMode(),
		c.GetPostgresConnectionTimeout(),
	)
}

// GetPopulateCommonTypes returns true if the (as set via default, config file, or environment variable)
// the common work item types such as bug or feature shall be created.
func (c *ConfigurationData) GetPopulateCommonTypes() bool {
	return c.v.GetBool(varPopulateCommonTypes)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the wit server binds to (e.g. "0.0.0.0:8080")
func (c *ConfigurationData) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// GetHeaderMaxLength returns the max length of HTTP headers allowed in the system
// For example it can be used to limit the size of bearer tokens returned by the api service
func (c *ConfigurationData) GetHeaderMaxLength() int64 {
	return c.v.GetInt64(varHeaderMaxLength)
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *ConfigurationData) IsPostgresDeveloperModeEnabled() bool {
	return c.v.GetBool(varDeveloperModeEnabled)
}

// IsAuthorizationEnabled returns true if space authorization enabled
// By default athorization is disabled in Developer Mode only (if F8_AUTHZ_ENABLED us not set)
// Set F8_AUTHZ_ENABLED env var to explictly disable or enable authorization regardless of Developer Mode settings
func (c *ConfigurationData) IsAuthorizationEnabled() bool {
	if c.v.IsSet(varAuthorizationEnabled) {
		return c.v.GetBool(varAuthorizationEnabled)
	}
	return !c.IsPostgresDeveloperModeEnabled()
}

// GetCacheControlWorkItemTypes returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item types.
func (c *ConfigurationData) GetCacheControlWorkItemTypes() string {
	return c.v.GetString(varCacheControlWorkItemTypes)
}

// GetCacheControlWorkItemType returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item type.
func (c *ConfigurationData) GetCacheControlWorkItemType() string {
	return c.v.GetString(varCacheControlWorkItemType)
}

// GetCacheControlWorkItemLinkTypes returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item types.
func (c *ConfigurationData) GetCacheControlWorkItemLinkTypes() string {
	return c.v.GetString(varCacheControlWorkItemLinkTypes)
}

// GetCacheControlWorkItemLinkType returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item type.
func (c *ConfigurationData) GetCacheControlWorkItemLinkType() string {
	return c.v.GetString(varCacheControlWorkItemLinkType)
}

// GetCacheControlWorkItems returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work items.
func (c *ConfigurationData) GetCacheControlWorkItems() string {
	return c.v.GetString(varCacheControlWorkItems)
}

// GetCacheControlWorkItem returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item.
func (c *ConfigurationData) GetCacheControlWorkItem() string {
	return c.v.GetString(varCacheControlWorkItem)
}

// GetCacheControlWorkItemLinks returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item links.
func (c *ConfigurationData) GetCacheControlWorkItemLinks() string {
	return c.v.GetString(varCacheControlWorkItemLinks)
}

// GetCacheControlWorkItemLink returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item.
func (c *ConfigurationData) GetCacheControlWorkItemLink() string {
	return c.v.GetString(varCacheControlWorkItemLink)
}

// GetCacheControlAreas returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work items.
func (c *ConfigurationData) GetCacheControlAreas() string {
	return c.v.GetString(varCacheControlAreas)
}

// GetCacheControlArea returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item (or a list of).
func (c *ConfigurationData) GetCacheControlArea() string {
	return c.v.GetString(varCacheControlArea)
}

// GetCacheControlLabels returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of labels.
func (c *ConfigurationData) GetCacheControlLabels() string {
	return c.v.GetString(varCacheControlLabels)
}

// GetCacheControlLabel returns the value to set in the "Cache-Control" HTTP response header
// when returning a label.
func (c *ConfigurationData) GetCacheControlLabel() string {
	return c.v.GetString(varCacheControlLabel)
}

// GetCacheControlSpaces returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of spaces.
func (c *ConfigurationData) GetCacheControlSpaces() string {
	return c.v.GetString(varCacheControlSpaces)
}

// GetCacheControlSpace returns the value to set in the "Cache-Control" HTTP response header
// when returning a space.
func (c *ConfigurationData) GetCacheControlSpace() string {
	return c.v.GetString(varCacheControlSpace)
}

// GetCacheControlIterations returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of iterations.
func (c *ConfigurationData) GetCacheControlIterations() string {
	return c.v.GetString(varCacheControlIterations)
}

// GetCacheControlIteration returns the value to set in the "Cache-Control" HTTP response header
// when returning an iteration.
func (c *ConfigurationData) GetCacheControlIteration() string {
	return c.v.GetString(varCacheControlIteration)
}

// GetCacheControlComments returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of comments.
func (c *ConfigurationData) GetCacheControlComments() string {
	return c.v.GetString(varCacheControlComments)
}

// GetCacheControlComment returns the value to set in the "Cache-Control" HTTP response header
// when returning a comment.
func (c *ConfigurationData) GetCacheControlComment() string {
	return c.v.GetString(varCacheControlComment)
}

// GetCacheControlFilters returns the value to set in the "Cache-Control" HTTP response header
// when returning comments.
func (c *ConfigurationData) GetCacheControlFilters() string {
	return c.v.GetString(varCacheControlFilters)
}

// GetCacheControlUsers returns the value to set in the "Cache-Control" HTTP response header
// when returning users.
func (c *ConfigurationData) GetCacheControlUsers() string {
	return c.v.GetString(varCacheControlUsers)
}

// GetCacheControlCollaborators returns the value to set in the "Cache-Control" HTTP response header
// when returning collaborators.
func (c *ConfigurationData) GetCacheControlCollaborators() string {
	return c.v.GetString(varCacheControlCollaborators)
}

// GetCacheControlUser returns the value to set in the "Cache-Control" HTTP response header
// when data for the current user.
func (c *ConfigurationData) GetCacheControlUser() string {
	return c.v.GetString(varCacheControlUser)
}

func (c *ConfigurationData) GetKeysEndpoint() string {
	return fmt.Sprintf("%s/api/token/keys", c.v.GetString(varAuthURL))
}

// GetAuthDevModeURL returns Auth Service URL used by default in Dev mode
func (c *ConfigurationData) GetAuthDevModeURL() string {
	return devModeAuthURL
}

// GetAuthDomainPrefix returns the domain prefix which should be used in requests to the auth service
func (c *ConfigurationData) GetAuthDomainPrefix() string {
	return c.v.GetString(varAuthDomainPrefix)
}

// GetAuthEndpointSpaces returns the <auth>/api/spaces endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> auth.service.domain.org
// or api.domain.org -> auth.domain.org
func (c *ConfigurationData) GetAuthEndpointSpaces(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/spaces")
}

func (c *ConfigurationData) getAuthEndpoint(req *http.Request, pathSufix string) (string, error) {
	return c.getServiceEndpoint(req, varAuthURL, devModeAuthURL, c.GetAuthDomainPrefix(), pathSufix)
}

// GetAuthEndpointLogin returns the <auth>/api/login endpoint
func (c *ConfigurationData) GetAuthEndpointLogin(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/login")
}

// GetAuthEndpointLogout returns the <auth>/api/logout endpoint
func (c *ConfigurationData) GetAuthEndpointLogout(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/logout")
}

// GetAuthEndpointLinksession returns the <auth>/api/link/session endpoint
func (c *ConfigurationData) GetAuthEndpointLinksession(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/link/session")
}

// GetAuthEndpointLink returns the <auth>/api/link endpoint
func (c *ConfigurationData) GetAuthEndpointLink(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/link")
}

// GetAuthEndpointTokenRefresh returns the <auth>/api/token/refresh endpoint
func (c *ConfigurationData) GetAuthEndpointTokenRefresh(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/token/refresh")
}

// GetAuthEndpointUsers returns the <auth>/api/users endpoint
func (c *ConfigurationData) GetAuthEndpointUsers(req *http.Request) (string, error) {
	return c.getAuthEndpoint(req, "api/users")
}

func (c *ConfigurationData) getServiceEndpoint(req *http.Request, varServiceURL string, devModeURL string, serviceDomainPrefix string, pathSufix string) (string, error) {
	var endpoint string
	var err error
	if c.v.IsSet(varServiceURL) {
		// Service URL is set. Calculate the URL endpoint
		endpoint = fmt.Sprintf("%s/%s", c.v.GetString(varServiceURL), pathSufix)
	} else {
		if c.IsPostgresDeveloperModeEnabled() {
			// Devmode is enabled. Calculate the URL endopoint using the devmode Service URL
			endpoint = fmt.Sprintf("%s/%s", devModeURL, pathSufix)
		} else {
			// Calculate relative URL based on request
			endpoint, err = c.getServiceURL(req, serviceDomainPrefix, pathSufix)
			if err != nil {
				return "", err
			}
		}
	}
	return endpoint, nil
}

// GetAuthNotApprovedRedirect returns the URL to redirect to if the user is not approved
// May return empty string which means an unauthorized error should be returned instead of redirecting the user
func (c *ConfigurationData) GetAuthNotApprovedRedirect() string {
	return c.v.GetString(varAuthNotApprovedRedirect)
}

// GetGithubAuthToken returns the actual Github OAuth Access Token
func (c *ConfigurationData) GetGithubAuthToken() string {
	return c.v.GetString(varGithubAuthToken)
}

// GetKeycloakSecret returns the keycloak client secret (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func (c *ConfigurationData) GetKeycloakSecret() string {
	return c.v.GetString(varKeycloakSecret)
}

// GetKeycloakClientID returns the keycloak client ID (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func (c *ConfigurationData) GetKeycloakClientID() string {
	return c.v.GetString(varKeycloakClientID)
}

// GetKeycloakDomainPrefix returns the domain prefix which should be used in all Keycloak requests
func (c *ConfigurationData) GetKeycloakDomainPrefix() string {
	return c.v.GetString(varKeycloakDomainPrefix)
}

// GetKeycloakRealm returns the keycloak realm name
func (c *ConfigurationData) GetKeycloakRealm() string {
	if c.v.IsSet(varKeycloakRealm) {
		return c.v.GetString(varKeycloakRealm)
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return devModeKeycloakRealm
	}
	return defaultKeycloakRealm
}

// GetKeycloakTestUserName returns the keycloak test user name used to obtain a test token (as set via config file or environment variable)
func (c *ConfigurationData) GetKeycloakTestUserName() string {
	return c.v.GetString(varKeycloakTesUserName)
}

// GetKeycloakTestUserSecret returns the keycloak test user password used to obtain a test token (as set via config file or environment variable)
func (c *ConfigurationData) GetKeycloakTestUserSecret() string {
	return c.v.GetString(varKeycloakTesUserSecret)
}

// GetKeycloakTestUser2Name returns the keycloak test user name used to obtain a test token (as set via config file or environment variable)
func (c *ConfigurationData) GetKeycloakTestUser2Name() string {
	return c.v.GetString(varKeycloakTesUser2Name)
}

// GetKeycloakTestUser2Secret returns the keycloak test user password used to obtain a test token (as set via config file or environment variable)
func (c *ConfigurationData) GetKeycloakTestUser2Secret() string {
	return c.v.GetString(varKeycloakTesUser2Secret)
}

// GetKeycloakEndpointAuth returns the keycloak auth endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointAuth(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "auth")
}

// GetKeycloakEndpointToken returns the keycloak token endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointToken(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "token")
}

// GetKeycloakEndpointUserInfo returns the keycloak userinfo endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointUserInfo(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "userinfo")
}

// GetKeycloakEndpointAdmin returns the <keycloak>/realms/admin/<realm> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointAdmin(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/admin/realms/"+c.GetKeycloakRealm())
}

// GetKeycloakEndpointAuthzResourceset returns the <keycloak>/realms/<realm>/authz/protection/resource_set endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointAuthzResourceset(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/authz/protection/resource_set")
}

// GetKeycloakEndpointClients returns the <keycloak>/admin/realms/<realm>/clients endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointClients(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/admin/realms/"+c.GetKeycloakRealm()+"/clients")
}

// GetKeycloakEndpointEntitlement returns the <keycloak>/realms/<realm>/authz/entitlement/<clientID> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointEntitlement(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/authz/entitlement/"+c.GetKeycloakClientID())
}

// GetKeycloakEndpointBroker returns the <keycloak>/realms/<realm>/authz/entitlement/<clientID> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointBroker(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/broker")
}

// GetKeycloakAccountEndpoint returns the API URL for Read and Update on Keycloak User Accounts.
func (c *ConfigurationData) GetKeycloakAccountEndpoint(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/account")
}

// GetKeycloakEndpointLogout returns the keycloak logout endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *ConfigurationData) GetKeycloakEndpointLogout(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "logout")
}

// GetKeycloakDevModeURL returns Keycloak URL (including realm name) used by default in Dev mode
// Returns "" if DevMode is not enabled
func (c *ConfigurationData) GetKeycloakDevModeURL() string {
	if c.IsPostgresDeveloperModeEnabled() {
		return fmt.Sprintf("%s/auth/realms/%s", devModeKeycloakURL, c.GetKeycloakRealm())
	}
	return ""
}

func (c *ConfigurationData) getKeycloakOpenIDConnectEndpoint(req *http.Request, pathSufix string) (string, error) {
	return c.getKeycloakEndpoint(req, c.openIDConnectPath(pathSufix))
}

func (c *ConfigurationData) getKeycloakEndpoint(req *http.Request, pathSufix string) (string, error) {
	return c.getServiceEndpoint(req, varKeycloakURL, devModeKeycloakURL, c.GetKeycloakDomainPrefix(), pathSufix)
}

func (c *ConfigurationData) openIDConnectPath(suffix string) string {
	return "auth/realms/" + c.GetKeycloakRealm() + "/protocol/openid-connect/" + suffix
}

func (c *ConfigurationData) getServiceURL(req *http.Request, serviceDomainPrefix string, path string) (string, error) {
	scheme := "http"
	if req.URL != nil && req.URL.Scheme == "https" { // isHTTPS
		scheme = "https"
	}
	xForwardProto := req.Header.Get("X-Forwarded-Proto")
	if xForwardProto != "" {
		scheme = xForwardProto
	}

	newHost, err := rest.ReplaceDomainPrefix(req.Host, serviceDomainPrefix)
	if err != nil {
		return "", err
	}
	newURL := fmt.Sprintf("%s://%s/%s", scheme, newHost, path)

	return newURL, nil
}

// GetCheStarterURL returns the URL for the Che Starter service used by codespaces to initiate code editing
func (c *ConfigurationData) GetCheStarterURL() string {
	return c.v.GetString(varCheStarterURL)
}

// GetOpenshiftTenantMasterURL returns the URL for the openshift cluster where the tenant services are running
func (c *ConfigurationData) GetOpenshiftTenantMasterURL() string {
	return c.v.GetString(varOpenshiftTenantMasterURL)
}

// GetLogLevel returns the loggging level (as set via config file or environment variable)
func (c *ConfigurationData) GetLogLevel() string {
	return c.v.GetString(varLogLevel)
}

// IsLogJSON returns if we should log json format (as set via config file or environment variable)
func (c *ConfigurationData) IsLogJSON() bool {
	if c.v.IsSet(varLogJSON) {
		return c.v.GetBool(varLogJSON)
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return false
	}
	return true
}

// GetValidRedirectURLs returns the RegEx of valid redirect URLs for auth requests
// If the F8_REDIRECT_VALID env var is not set then in Dev Mode all redirects allowed - *
// In prod mode the default regex will be returned
func (c *ConfigurationData) GetValidRedirectURLs(req *http.Request) (string, error) {
	if c.v.IsSet(varValidRedirectURLs) {
		return c.v.GetString(varValidRedirectURLs), nil
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return devModeValidRedirectURLs, nil
	}
	return c.checkLocalhostRedirectException(req)
}

func (c *ConfigurationData) checkLocalhostRedirectException(req *http.Request) (string, error) {
	if req.URL == nil {
		return DefaultValidRedirectURLs, nil
	}
	matched, err := regexp.MatchString(localhostRedirectException, req.URL.String())
	if err != nil {
		return "", err
	}
	if matched {
		return localhostRedirectURLs, nil
	}
	return DefaultValidRedirectURLs, nil
}

// GetTenantServiceURL returns the URL for the Tenant service used by login to initialize OSO tenant space
func (c *ConfigurationData) GetTenantServiceURL() string {
	return c.v.GetString(varTenantServiceURL)
}

// GetNotificationServiceURL returns the URL for the Notification service used for event notification
func (c *ConfigurationData) GetNotificationServiceURL() string {
	return c.v.GetString(varNotificationServiceURL)
}

const (
	defaultHeaderMaxLength = 5000 // bytes

	defaultLogLevel = "info"

	// Auth service URL to be used in dev mode. Can be overridden by setting up auth.url
	devModeAuthURL = "https://auth.prod-preview.openshift.io"

	defaultKeycloakClientID = "fabric8-online-platform"
	defaultKeycloakSecret   = "7a3d5a00-7f80-40cf-8781-b5b6f2dfd1bd"

	defaultKeycloakDomainPrefix = "sso"
	defaultKeycloakRealm        = "fabric8"

	// Github does not allow committing actual OAuth tokens no matter how less privilege the token has
	camouflagedAccessToken = "751e16a8b39c0985066-AccessToken-4871777f2c13b32be8550"

	defaultKeycloakTesUserName    = "testuser"
	defaultKeycloakTesUserSecret  = "testuser"
	defaultKeycloakTesUser2Name   = "testuser2"
	defaultKeycloakTesUser2Secret = "testuser2"

	// Keycloak vars to be used in dev mode. Can be overridden by setting up keycloak.url & keycloak.realm
	devModeKeycloakURL   = "https://sso.prod-preview.openshift.io"
	devModeKeycloakRealm = "fabric8-test"

	defaultOpenshiftTenantMasterURL = "https://tsrv.devshift.net:8443"
	defaultCheStarterURL            = "che-server"

	// DefaultValidRedirectURLs is a regex to be used to whitelist redirect URL for auth
	// If the F8_REDIRECT_VALID env var is not set then in Dev Mode all redirects allowed - *
	// In prod mode the following regex will be used by default:
	DefaultValidRedirectURLs = "^(https|http)://([^/]+[.])?(?i:openshift[.]io)(/.*)?$" // *.openshift.io/*
	devModeValidRedirectURLs = ".*"
	// Allow redirects to localhost when running in prod-preveiw
	localhostRedirectURLs      = "(" + DefaultValidRedirectURLs + "|^(https|http)://([^/]+[.])?(localhost|127[.]0[.]0[.]1)(:\\d+)?(/.*)?$)" // *.openshift.io/* or localhost/* or 127.0.0.1/*
	localhostRedirectException = "^(https|http)://([^/]+[.])?(?i:prod-preview[.]openshift[.]io)(:\\d+)?(/.*)?$"                             // *.prod-preview.openshift.io/*

)

// ActualToken is actual OAuth access token of github
var defaultActualToken = strings.Split(camouflagedAccessToken, "-AccessToken-")[0] + strings.Split(camouflagedAccessToken, "-AccessToken-")[1]
