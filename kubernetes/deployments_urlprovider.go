package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// BaseURLProvider provides the BASE URL (minimal path) of several APIs used in Deployments
// for true multicluster support, every API in this inteface should take an environment namespace name.
// This hasn't been done, because the rest of fabric8 seems to assume the cluster is the same.
// For most uses, the proxy server will hide this issue - but not for mertics/logging and console.

/* typical URLS from tenant:
"console-url":"https://console.starter-us-east-2a.openshift.com/console/",
"logging-url":"https://console.starter-us-east-2a.openshift.com/console/",
"metrics-url":"https://metrics.starter-us-east-2a.openshift.com/",
*/

// BaseURLProvider provides all URLS used by the deployments Kubernetes implementation
// It takes into account proxies, Auth or Tenant services, etc.
type BaseURLProvider interface {
	GetAPIURL() string
	GetMetricsURL() (*string, error)
	GetConsoleURL(envNS string) (*string, error)
	GetLoggingURL(envNS string, deploymentName string) (*string, error)

	GetAPIToken() *string
	GetMetricsToken() *string
}

// there are several concrete instantiations:
//
// 1) the original Deployments implementation:
//    - authURLProvider
//    - access Auth and OSO directly
//
// 2) the interim implementation
//    - authURLProvider
//    - access Auth and OSO metrics directly,
//    - use proxy for normal OSO API calls
//
// 3) final implementation
//   - tenantURLProvider
//   - access Tenant instead of Auth
//   - use use proxy for normal OSO API calls
//   - access OSO metrics directly (until proxy supports this)

type authURLProvider struct {
	apiURL       string
	apiToken     string
	clusterURL   string
	clusterToken string
}

// ensure authURLProvider implements BaseURLProvider
var _ BaseURLProvider = &authURLProvider{}
var _ BaseURLProvider = (*authURLProvider)(nil)

type tenantURLProvider struct {
	apiURL     string
	apiToken   string
	tenant     *tenant.TenantAttributes
	namespaces map[string]*tenant.NamespaceAttributes
}

// ensure tenantURLProvider implements BaseURLProvider
var _ BaseURLProvider = &tenantURLProvider{}
var _ BaseURLProvider = (*tenantURLProvider)(nil)

// NewURLProvider looks at what servers are available and create a BaseURLProvder that fits
func NewURLProvider(ctx context.Context, config *configuration.Registry) (BaseURLProvider, error) {

	osProxyURL := config.GetOpenshiftProxyURL()
	tenantURL := config.GetTenantServiceURL()

	if len(tenantURL) != 0 {
		if len(osProxyURL) == 0 {
			return newTenantURLProvider(ctx, config)
		}
		return newTenantProxyURLProvider(ctx, config, osProxyURL)
	}
	if len(osProxyURL) == 0 {
		return newAuthNoProxyURLProvider(ctx, config)
	}
	return newAuthProxyURLProvider(ctx, config, osProxyURL)
}

// using tenant and proxy, access metrics directly
func newTenantProxyURLProvider(ctx context.Context, config *configuration.Registry, osProxyURL string) (*tenantURLProvider, error) {

	p, err := newTenantURLProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// all non-metric API calls go via the proxy
	p.apiURL = osProxyURL
	p.apiToken = goajwt.ContextJWT(ctx).Raw

	return p, nil
}

// using Tenant and no proxy
func newTenantURLProvider(ctx context.Context, config *configuration.Registry) (*tenantURLProvider, error) {

	// create Tenant API client
	tenantResponse, err := account.ShowTenant(ctx, config)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Tenant server")
		return nil, errs.Wrapf(err, "error creating Tenant client")
	}
	return newTenantURLProviderFromTenant(tenantResponse, goajwt.ContextJWT(ctx).Raw), nil
}

func newTenantURLProviderFromTenant(t *tenant.TenantSingle, token string) *tenantURLProvider {
	namespaceMap := make(map[string]*tenant.NamespaceAttributes)
	for i, namespace := range t.Data.Attributes.Namespaces {
		namespaceMap[*namespace.Name] = t.Data.Attributes.Namespaces[i]
	}

	defaultClusterURL := ""
	if len(t.Data.Attributes.Namespaces) == 0 {
		log.Error(nil, map[string]interface{}{
			"tenant": *t.Data.Attributes.Email,
		}, "this tenant has no namespaces: %s", *t.Data.Attributes.Email)
	} else {
		// use the cluster of the first namespace as the default
		// strictly speaking, there should be no default
		// having a default allows the new code to work with an old Auth or Tenant server
		defaultClusterURL = *t.Data.Attributes.Namespaces[0].ClusterURL
	}

	provider := &tenantURLProvider{
		apiURL:     defaultClusterURL,
		apiToken:   token,
		tenant:     t.Data.Attributes,
		namespaces: namespaceMap,
	}
	return provider
}

// NewTenantURLProviderFromTenant exposed for testing
func NewTenantURLProviderFromTenant(t *tenant.TenantSingle, token string) BaseURLProvider {
	return newTenantURLProviderFromTenant(t, token)
}

func (up *tenantURLProvider) GetAPIToken() *string {
	return &up.apiToken
}

func (up *tenantURLProvider) GetMetricsToken() *string {
	return &up.apiToken
}

func (up *tenantURLProvider) GetAPIURL() string {
	return up.apiURL
}

func (up *tenantURLProvider) GetConsoleURL(envNS string) (*string, error) {
	ns := up.namespaces[envNS]
	if ns == nil {
		return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.Email)
	}
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	baseURL := ns.ClusterConsoleURL
	if baseURL == nil || len(*baseURL) == 0 {
		// if it's missing, modify the cluster URL
		bu, err := modifyURL(*ns.ClusterURL, "console", "/console")
		if err != nil {
			return nil, err
		}
		buStr := bu.String()
		baseURL = &buStr
	}
	consoleURL := fmt.Sprintf("%s/project/%s", *baseURL, envNS)
	return &consoleURL, nil
}

func (up *tenantURLProvider) GetLoggingURL(envNS string, deployName string) (*string, error) {
	ns := up.namespaces[envNS]
	if ns == nil {
		return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.Email)
	}
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	baseURL := ns.ClusterLoggingURL
	if baseURL == nil || len(*baseURL) == 0 {
		// if it's missing, modify the cluster URL
		bu, err := modifyURL(*ns.ClusterURL, "console", "/console")
		if err != nil {
			return nil, err
		}
		buStr := bu.String()
		baseURL = &buStr
	}
	loggingURL := fmt.Sprintf("%s/project/%s/browse/rc/%s?tab=logs", *baseURL, envNS, deployName)
	return &loggingURL, nil
}

func (up *tenantURLProvider) GetMetricsURL() (*string, error) {

	// this code should be migrated to take an envNS instead of using namespaces[0]
	metricsURL := up.tenant.Namespaces[0].ClusterMetricsURL
	if metricsURL == nil || len(*metricsURL) == 0 {
		// In the absence of a better way (i.e. tenant) to get the user's metrics URL,
		// substitute "api" with "metrics" in user's cluster URL
		mu, err := modifyURL(*up.tenant.Namespaces[0].ClusterURL, "metrics", "")
		if err != nil {
			return nil, err
		}
		muStr := mu.String()
		metricsURL = &muStr
	}
	return metricsURL, nil
}

// using auth and proxy, access metrics directly
func newAuthProxyURLProvider(ctx context.Context, config *configuration.Registry, osProxyURL string) (*authURLProvider, error) {

	// this is inefficient; we still need to get the cluster and OSO tokens so we can access metrics
	// the console, log and API urls should come from Auth or Tenant services instead of calculating in this code.
	p, err := newAuthURLProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// all non-metric API calls go via the proxy
	p.apiURL = osProxyURL
	p.apiToken = goajwt.ContextJWT(ctx).Raw

	return p, nil
}

// using Auth and no proxy
func newAuthNoProxyURLProvider(ctx context.Context, config *configuration.Registry) (*authURLProvider, error) {

	p, err := newAuthURLProvider(ctx, config)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// using Auth, no proxy (internal call)
func newAuthURLProvider(ctx context.Context, config *configuration.Registry) (*authURLProvider, error) {
	// create Auth API client
	authClient, err := auth.CreateClient(ctx, config)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Auth server")
		return nil, errs.Wrapf(err, "error creating Auth client")
	}

	authUser, err := getUser(ctx, *authClient)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error retrieving user definition from Auth client")
		return nil, errs.Wrapf(err, "error retrieving user definition from Auth client")
	}

	if authUser == nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error retrieving user from Auth server")
		return nil, errs.New("error getting user from Auth Server")
	}

	if authUser.Data.Attributes.Cluster == nil {
		log.Error(ctx, map[string]interface{}{
			"err":     err,
			"user_id": *authUser.Data.Attributes.UserID,
		}, "error retrieving user cluster from Auth server")
		return nil, errs.Errorf("error getting user cluster from Auth Server: %s", tostring(authUser))
	}

	// get the openshift/kubernetes auth info for the cluster OpenShift API
	osauth, err := getTokenData(ctx, *authClient, *authUser.Data.Attributes.Cluster)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":     err,
			"user_id": *authUser.Data.Attributes.UserID,
			"cluster": *authUser.Data.Attributes.Cluster,
		}, "error getting openshift credentials for user from Auth server")
		return nil, errs.Wrapf(err, "error getting openshift credentials")
	}

	provider := &authURLProvider{
		apiURL:       *authUser.Data.Attributes.Cluster,
		apiToken:     *osauth.AccessToken,
		clusterURL:   *authUser.Data.Attributes.Cluster,
		clusterToken: *osauth.AccessToken,
	}

	return provider, nil
}

// NewTestURLProvider creates a provider with the same URL and token for both API and metrics
func NewTestURLProvider(clusterURL string, token string) BaseURLProvider {
	provider := &authURLProvider{
		apiURL:       clusterURL,
		apiToken:     token,
		clusterURL:   clusterURL,
		clusterToken: token,
	}

	return provider
}

// NewTestURLWithMetricsProvider creates a provider with the different URL and token for API and metrics
func NewTestURLWithMetricsProvider(apiURL string, token string, clusterURL string, clusterToken string) BaseURLProvider {
	provider := &authURLProvider{
		apiURL:       apiURL,
		apiToken:     token,
		clusterURL:   clusterURL,
		clusterToken: clusterToken,
	}

	return provider
}

func (up *authURLProvider) GetAPIToken() *string {
	return &up.apiToken
}

func (up *authURLProvider) GetMetricsToken() *string {
	return &up.clusterToken
}

func (up *authURLProvider) GetAPIURL() string {
	return up.apiURL
}

func (up *authURLProvider) GetConsoleURL(envNS string) (*string, error) {
	path := fmt.Sprintf("console/project/%s", envNS)
	// Replace "api" prefix with "console" and append path
	consoleURL, err := modifyURL(up.clusterURL, "console", path)
	if err != nil {
		return nil, err
	}
	consoleURLStr := consoleURL.String()
	return &consoleURLStr, nil
}

func (up *authURLProvider) GetLoggingURL(envNS string, deployName string) (*string, error) {
	consoleURL, err := up.GetConsoleURL(envNS)
	if err != nil {
		return nil, err
	}
	logURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", *consoleURL, deployName)
	return &logURL, nil
}

func (up *authURLProvider) GetMetricsURL() (*string, error) {
	// metrics URL is taken from the cluster URL
	// In the absence of a better way (i.e. tenant) to get the user's metrics URL,
	// substitute "api" with "metrics" in user's cluster URL
	metricsURL, err := modifyURL(up.clusterURL, "metrics", "")
	if err != nil {
		return nil, err
	}
	mu := metricsURL.String()
	return &mu, nil
}

func getTokenData(ctx context.Context, authClient authservice.Client, forService string) (*authservice.TokenData, error) {

	resp, err := authClient.RetrieveToken(ctx, authservice.RetrieveTokenPath(), forService, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to retrieve Auth token for '%s' service", forService)
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"request_path": authservice.ShowUserPath(),
			"for_service":  forService,
			"http_status":  status,
		}, "failed to GET token from auth service due to HTTP error %s", status)
		return nil, errs.Errorf("failed to GET Auth token for '%s' service due to status code %d", forService, status)
	}

	var respType authservice.TokenData
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"for_service":   forService,
			"http_status":   status,
			"response_body": respBody,
		}, "unable to unmarshal Auth token")
		return nil, errs.Wrapf(err, "unable to unmarshal Auth token for '%s' service from Auth service", forService)
	}
	return &respType, nil
}

func getUser(ctx context.Context, authClient authservice.Client) (*authservice.User, error) {
	// get the user definition (for cluster URL)
	resp, err := authClient.ShowUser(ctx, authservice.ShowUserPath(), nil, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to retrieve user from Auth service")
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"response_body": respBody,
			"http_status":   status,
		}, "failed to GET user from auth service due to HTTP error %s", status)
		return nil, errs.Errorf("failed to GET user due to status code %d", status)
	}

	var respType authservice.User
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"response_body": respBody,
		}, "unable to unmarshal user definition from Auth service")
		return nil, errs.Wrapf(err, "unable to unmarshal user definition from Auth service")
	}
	return &respType, nil
}

func modifyURL(apiURLStr string, prefix string, path string) (*url.URL, error) {
	// Parse as URL to give us easy access to the hostname
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, err
	}

	// Get the hostname (without port) and replace api prefix with prefix arg
	apiHostname := apiURL.Hostname()
	if !strings.HasPrefix(apiHostname, "api") {
		return nil, errs.Errorf("cluster URL does not begin with \"api\": %s", apiHostname)
	}
	newHostname := strings.Replace(apiHostname, "api", prefix, 1)
	// Construct URL using just scheme from API URL, modified hostname and supplied path
	newURL := &url.URL{
		Scheme: apiURL.Scheme,
		Host:   newHostname,
		Path:   path,
	}
	return newURL, nil
}

func tostring(item interface{}) string {
	bytes, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
