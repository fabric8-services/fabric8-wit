package configuration

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// String returns the current configuration as a string
func (c *Registry) String() string {
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
	varMetricsHTTPAddress           = "metrics.http.address"
	varDeveloperModeEnabled         = "developer.mode.enabled"
	varAuthDomainPrefix             = "auth.domain.prefix"
	varAuthShortServiceHostName     = "auth.servicehostname.short"
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

// Registry encapsulates the Viper configuration registry which stores the
// configuration data in-memory.
type Registry struct {
	v               *viper.Viper
	tokenPublicKey  *rsa.PublicKey
	tokenPrivateKey *rsa.PrivateKey
}

// New creates a configuration reader object using a configurable configuration
// file path.
func New(configFilePath string) (*Registry, error) {
	c := Registry{
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
			return nil, errs.Errorf("Fatal error config file: %s \n", err)
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
func (c *Registry) GetDefaultConfigurationFile() string {
	return defaultConfigFile
}

// Get is a wrapper over New() which reads configuration file path from the
// environment variable.
func Get() (*Registry, error) {
	cd, err := New(getConfigFilePath())
	return cd, err
}

func (c *Registry) setConfigDefaults() {
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
	c.v.SetDefault(varMetricsHTTPAddress, "0.0.0.0:8080")
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
	c.v.SetDefault(varAuthorizationEnabled, true)

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
	c.v.SetDefault(varOpenshiftTenantMasterURL, defaultOpenshiftTenantMasterURL)
	c.v.SetDefault(varCheStarterURL, defaultCheStarterURL)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *Registry) GetPostgresHost() string {
	return c.v.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *Registry) GetPostgresPort() int64 {
	return c.v.GetInt64(varPostgresPort)
}

// GetFeatureWorkitemRemote returns true if remote Work Item feaute is enabled
func (c *Registry) GetFeatureWorkitemRemote() bool {
	return c.v.GetBool(varFeatureWorkitemRemote)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *Registry) GetPostgresUser() string {
	return c.v.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *Registry) GetPostgresDatabase() string {
	return c.v.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *Registry) GetPostgresPassword() string {
	return c.v.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *Registry) GetPostgresSSLMode() string {
	return c.v.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func (c *Registry) GetPostgresConnectionTimeout() int64 {
	return c.v.GetInt64(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *Registry) GetPostgresConnectionRetrySleep() time.Duration {
	return c.v.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresTransactionTimeout returns the number of minutes to timeout a transaction
func (c *Registry) GetPostgresTransactionTimeout() time.Duration {
	return c.v.GetDuration(varPostgresTransactionTimeout)
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior
func (c *Registry) GetPostgresConnectionMaxIdle() int {
	return c.v.GetInt(varPostgresConnectionMaxIdle)
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior
func (c *Registry) GetPostgresConnectionMaxOpen() int {
	return c.v.GetInt(varPostgresConnectionMaxOpen)
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *Registry) GetPostgresConfigString() string {
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
func (c *Registry) GetPopulateCommonTypes() bool {
	return c.v.GetBool(varPopulateCommonTypes)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the wit server binds to (e.g. "0.0.0.0:8080")
func (c *Registry) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// GetMetricsHTTPAddress returns the address the /metrics endpoing will be mounted.
// By default GetMetricsHTTPAddress is the same as GetHTTPAddress
func (c *Registry) GetMetricsHTTPAddress() string {
	return c.v.GetString(varMetricsHTTPAddress)
}

// GetHeaderMaxLength returns the max length of HTTP headers allowed in the system
// For example it can be used to limit the size of bearer tokens returned by the api service
func (c *Registry) GetHeaderMaxLength() int64 {
	return c.v.GetInt64(varHeaderMaxLength)
}

// IsPostgresDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *Registry) IsPostgresDeveloperModeEnabled() bool {
	return c.v.GetBool(varDeveloperModeEnabled)
}

// IsAuthorizationEnabled returns true if space authorization enabled
func (c *Registry) IsAuthorizationEnabled() bool {
	return c.v.GetBool(varAuthorizationEnabled)
}

// GetCacheControlWorkItemTypes returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item types.
func (c *Registry) GetCacheControlWorkItemTypes() string {
	return c.v.GetString(varCacheControlWorkItemTypes)
}

// GetCacheControlWorkItemType returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item type.
func (c *Registry) GetCacheControlWorkItemType() string {
	return c.v.GetString(varCacheControlWorkItemType)
}

// GetCacheControlWorkItemLinkTypes returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item types.
func (c *Registry) GetCacheControlWorkItemLinkTypes() string {
	return c.v.GetString(varCacheControlWorkItemLinkTypes)
}

// GetCacheControlWorkItemLinkType returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item type.
func (c *Registry) GetCacheControlWorkItemLinkType() string {
	return c.v.GetString(varCacheControlWorkItemLinkType)
}

// GetCacheControlWorkItems returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work items.
func (c *Registry) GetCacheControlWorkItems() string {
	return c.v.GetString(varCacheControlWorkItems)
}

// GetCacheControlWorkItem returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item.
func (c *Registry) GetCacheControlWorkItem() string {
	return c.v.GetString(varCacheControlWorkItem)
}

// GetCacheControlWorkItemLinks returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work item links.
func (c *Registry) GetCacheControlWorkItemLinks() string {
	return c.v.GetString(varCacheControlWorkItemLinks)
}

// GetCacheControlWorkItemLink returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item.
func (c *Registry) GetCacheControlWorkItemLink() string {
	return c.v.GetString(varCacheControlWorkItemLink)
}

// GetCacheControlAreas returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of work items.
func (c *Registry) GetCacheControlAreas() string {
	return c.v.GetString(varCacheControlAreas)
}

// GetCacheControlArea returns the value to set in the "Cache-Control" HTTP response header
// when returning a work item (or a list of).
func (c *Registry) GetCacheControlArea() string {
	return c.v.GetString(varCacheControlArea)
}

// GetCacheControlLabels returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of labels.
func (c *Registry) GetCacheControlLabels() string {
	return c.v.GetString(varCacheControlLabels)
}

// GetCacheControlLabel returns the value to set in the "Cache-Control" HTTP response header
// when returning a label.
func (c *Registry) GetCacheControlLabel() string {
	return c.v.GetString(varCacheControlLabel)
}

// GetCacheControlSpaces returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of spaces.
func (c *Registry) GetCacheControlSpaces() string {
	return c.v.GetString(varCacheControlSpaces)
}

// GetCacheControlSpace returns the value to set in the "Cache-Control" HTTP response header
// when returning a space.
func (c *Registry) GetCacheControlSpace() string {
	return c.v.GetString(varCacheControlSpace)
}

// GetCacheControlIterations returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of iterations.
func (c *Registry) GetCacheControlIterations() string {
	return c.v.GetString(varCacheControlIterations)
}

// GetCacheControlIteration returns the value to set in the "Cache-Control" HTTP response header
// when returning an iteration.
func (c *Registry) GetCacheControlIteration() string {
	return c.v.GetString(varCacheControlIteration)
}

// GetCacheControlComments returns the value to set in the "Cache-Control" HTTP response header
// when returning a list of comments.
func (c *Registry) GetCacheControlComments() string {
	return c.v.GetString(varCacheControlComments)
}

// GetCacheControlComment returns the value to set in the "Cache-Control" HTTP response header
// when returning a comment.
func (c *Registry) GetCacheControlComment() string {
	return c.v.GetString(varCacheControlComment)
}

// GetCacheControlFilters returns the value to set in the "Cache-Control" HTTP response header
// when returning comments.
func (c *Registry) GetCacheControlFilters() string {
	return c.v.GetString(varCacheControlFilters)
}

// GetCacheControlUsers returns the value to set in the "Cache-Control" HTTP response header
// when returning users.
func (c *Registry) GetCacheControlUsers() string {
	return c.v.GetString(varCacheControlUsers)
}

// GetCacheControlCollaborators returns the value to set in the "Cache-Control" HTTP response header
// when returning collaborators.
func (c *Registry) GetCacheControlCollaborators() string {
	return c.v.GetString(varCacheControlCollaborators)
}

// GetCacheControlUser returns the value to set in the "Cache-Control" HTTP response header
// when data for the current user.
func (c *Registry) GetCacheControlUser() string {
	return c.v.GetString(varCacheControlUser)
}

// GetKeysEndpoint returns the endpoint to the auth service for key mgmt.
func (c *Registry) GetKeysEndpoint() string {
	return fmt.Sprintf("%s/api/token/keys", c.v.GetString(varAuthURL))
}

// GetAuthDevModeURL returns Auth Service URL used by default in Dev mode
func (c *Registry) GetAuthDevModeURL() string {
	return devModeAuthURL
}

// GetAuthDomainPrefix returns the domain prefix which should be used in requests to the auth service
func (c *Registry) GetAuthDomainPrefix() string {
	return c.v.GetString(varAuthDomainPrefix)
}

// GetAuthShortServiceHostName returns the short Auth service host name
// or the full Auth service URL if not set and Dev Mode enabled.
// Otherwise returns the default host - http://auth
func (c *Registry) GetAuthShortServiceHostName() string {
	if c.v.IsSet(varAuthShortServiceHostName) {
		return c.v.GetString(varAuthShortServiceHostName)
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return c.GetAuthServiceURL()
	}
	return defaultAuthShortServiceHostName
}

// GetAuthServiceURL returns the Auth Service URL
func (c *Registry) GetAuthServiceURL() string {
	return c.v.GetString(varAuthURL)
}

func (c *Registry) getServiceEndpoint(req *http.Request, varServiceURL string, devModeURL string, serviceDomainPrefix string, pathSufix string) (string, error) {
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
func (c *Registry) GetAuthNotApprovedRedirect() string {
	return c.v.GetString(varAuthNotApprovedRedirect)
}

// GetGithubAuthToken returns the actual Github OAuth Access Token
func (c *Registry) GetGithubAuthToken() string {
	return c.v.GetString(varGithubAuthToken)
}

// GetKeycloakSecret returns the keycloak client secret (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func (c *Registry) GetKeycloakSecret() string {
	return c.v.GetString(varKeycloakSecret)
}

// GetKeycloakClientID returns the keycloak client ID (as set via config file or environment variable)
// that is used to make authorized Keycloak API Calls.
func (c *Registry) GetKeycloakClientID() string {
	return c.v.GetString(varKeycloakClientID)
}

// GetKeycloakDomainPrefix returns the domain prefix which should be used in all Keycloak requests
func (c *Registry) GetKeycloakDomainPrefix() string {
	return c.v.GetString(varKeycloakDomainPrefix)
}

// GetKeycloakRealm returns the keycloak realm name
func (c *Registry) GetKeycloakRealm() string {
	if c.v.IsSet(varKeycloakRealm) {
		return c.v.GetString(varKeycloakRealm)
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return devModeKeycloakRealm
	}
	return defaultKeycloakRealm
}

// GetKeycloakTestUserName returns the keycloak test user name used to obtain a test token (as set via config file or environment variable)
func (c *Registry) GetKeycloakTestUserName() string {
	return c.v.GetString(varKeycloakTesUserName)
}

// GetKeycloakTestUser2Name returns the keycloak test user name used to obtain a test token (as set via config file or environment variable)
func (c *Registry) GetKeycloakTestUser2Name() string {
	return c.v.GetString(varKeycloakTesUser2Name)
}

// GetKeycloakEndpointAuth returns the keycloak auth endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointAuth(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "auth")
}

// GetKeycloakEndpointToken returns the keycloak token endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointToken(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "token")
}

// GetKeycloakEndpointUserInfo returns the keycloak userinfo endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointUserInfo(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "userinfo")
}

// GetKeycloakEndpointAdmin returns the <keycloak>/realms/admin/<realm> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointAdmin(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/admin/realms/"+c.GetKeycloakRealm())
}

// GetKeycloakEndpointAuthzResourceset returns the <keycloak>/realms/<realm>/authz/protection/resource_set endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointAuthzResourceset(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/authz/protection/resource_set")
}

// GetKeycloakEndpointClients returns the <keycloak>/admin/realms/<realm>/clients endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointClients(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/admin/realms/"+c.GetKeycloakRealm()+"/clients")
}

// GetKeycloakEndpointEntitlement returns the <keycloak>/realms/<realm>/authz/entitlement/<clientID> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointEntitlement(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/authz/entitlement/"+c.GetKeycloakClientID())
}

// GetKeycloakEndpointBroker returns the <keycloak>/realms/<realm>/authz/entitlement/<clientID> endpoint
// set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointBroker(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/broker")
}

// GetKeycloakAccountEndpoint returns the API URL for Read and Update on Keycloak User Accounts.
func (c *Registry) GetKeycloakAccountEndpoint(req *http.Request) (string, error) {
	return c.getKeycloakEndpoint(req, "auth/realms/"+c.GetKeycloakRealm()+"/account")
}

// GetKeycloakEndpointLogout returns the keycloak logout endpoint set via config file or environment variable.
// If nothing set then in Dev environment the defualt endopoint will be returned.
// In producion the endpoint will be calculated from the request by replacing the last domain/host name in the full host name.
// Example: api.service.domain.org -> sso.service.domain.org
// or api.domain.org -> sso.domain.org
func (c *Registry) GetKeycloakEndpointLogout(req *http.Request) (string, error) {
	return c.getKeycloakOpenIDConnectEndpoint(req, "logout")
}

// GetKeycloakDevModeURL returns Keycloak URL (including realm name) used by default in Dev mode
// Returns "" if DevMode is not enabled
func (c *Registry) GetKeycloakDevModeURL() string {
	if c.IsPostgresDeveloperModeEnabled() {
		return fmt.Sprintf("%s/auth/realms/%s", devModeKeycloakURL, c.GetKeycloakRealm())
	}
	return ""
}

func (c *Registry) getKeycloakOpenIDConnectEndpoint(req *http.Request, pathSufix string) (string, error) {
	return c.getKeycloakEndpoint(req, c.openIDConnectPath(pathSufix))
}

func (c *Registry) getKeycloakEndpoint(req *http.Request, pathSufix string) (string, error) {
	return c.getServiceEndpoint(req, varKeycloakURL, devModeKeycloakURL, c.GetKeycloakDomainPrefix(), pathSufix)
}

func (c *Registry) openIDConnectPath(suffix string) string {
	return "auth/realms/" + c.GetKeycloakRealm() + "/protocol/openid-connect/" + suffix
}

func (c *Registry) getServiceURL(req *http.Request, serviceDomainPrefix string, path string) (string, error) {
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
func (c *Registry) GetCheStarterURL() string {
	return c.v.GetString(varCheStarterURL)
}

// GetOpenshiftTenantMasterURL returns the URL for the openshift cluster where the tenant services are running
func (c *Registry) GetOpenshiftTenantMasterURL() string {
	return c.v.GetString(varOpenshiftTenantMasterURL)
}

// GetLogLevel returns the loggging level (as set via config file or environment variable)
func (c *Registry) GetLogLevel() string {
	return c.v.GetString(varLogLevel)
}

// IsLogJSON returns if we should log json format (as set via config file or environment variable)
func (c *Registry) IsLogJSON() bool {
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
func (c *Registry) GetValidRedirectURLs(req *http.Request) (string, error) {
	if c.v.IsSet(varValidRedirectURLs) {
		return c.v.GetString(varValidRedirectURLs), nil
	}
	if c.IsPostgresDeveloperModeEnabled() {
		return devModeValidRedirectURLs, nil
	}
	return c.checkLocalhostRedirectException(req)
}

func (c *Registry) checkLocalhostRedirectException(req *http.Request) (string, error) {
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
func (c *Registry) GetTenantServiceURL() string {
	return c.v.GetString(varTenantServiceURL)
}

// GetNotificationServiceURL returns the URL for the Notification service used for event notification
func (c *Registry) GetNotificationServiceURL() string {
	return c.v.GetString(varNotificationServiceURL)
}

const (
	defaultHeaderMaxLength = 5000 // bytes

	defaultLogLevel = "info"

	// Auth service URL to be used in dev mode. Can be overridden by setting up auth.url
	devModeAuthURL = "http://localhost:8089"

	defaultAuthShortServiceHostName = "http://auth"

	defaultKeycloakClientID = "fabric8-online-platform"
	defaultKeycloakSecret   = "7a3d5a00-7f80-40cf-8781-b5b6f2dfd1bd"

	defaultKeycloakDomainPrefix = "sso"
	defaultKeycloakRealm        = "fabric8"

	// Github does not allow committing actual OAuth tokens no matter how less privilege the token has
	camouflagedAccessToken = "751e16a8b39c0985066-AccessToken-4871777f2c13b32be8550"

	defaultKeycloakTesUserName  = "testuser"
	defaultKeycloakTesUser2Name = "testuser2"

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

	// DevModeRsaPrivateKey for signing JWT Tokens in Dev Mode
	// ssh-keygen -f alm_rsa
	DevModeRsaPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
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
)

// ActualToken is actual OAuth access token of github
var defaultActualToken = strings.Split(camouflagedAccessToken, "-AccessToken-")[0] + strings.Split(camouflagedAccessToken, "-AccessToken-")[1]
