package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// BaseURLProvider provides the BASE URL (minimal path) of several APIs used in Deployments
type BaseURLProvider interface {
	GetAPIURL() string
	GetMetricsURL() (*string, error)
	GetConsoleURL(envNS string) (*string, error)
	GetLogURL(envNS string, deploymentName string) (*string, error)

	GetAPIToken() *string
	GetMetricsToken() *string
}

// there are several concrete instantiations:
//
// 1) the original Deployments implementation:
//    - access Auth and OSO directly
//
// 2) the interim implementation
//    - access Auth and OSO metrics directly,
//    - use proxy for normal OSO API calls
//
// 3) final implementation
//   - Access Tenant isntead of Auth
//   - use use proxy for normal OSO API calls
//   - access OSO metrics directly (until proxy supports this)

type originalURLProvider struct {
	apiURL       string
	apiToken     string
	metricURL    string
	metricsToken string
}

// ensure kubeClient implements KubeClientInterface
var _ BaseURLProvider = &originalURLProvider{}
var _ BaseURLProvider = (*originalURLProvider)(nil)

// NewURLProvider looks at what servers are available and create a BaseURLProvder that fits
func NewURLProvider(ctx context.Context, config *configuration.Registry) (BaseURLProvider, error) {

	osProxyURL := config.GetOpenshiftProxyURL()

	if len(osProxyURL) == 0 {
		return newAuthNoProxyURLProvider(ctx, config)
	}
	return newInterimAuthProxyFinalURLProvider(ctx, config, osProxyURL)
}

// using auth and proxy, access metrics directly
func newInterimAuthProxyFinalURLProvider(ctx context.Context, config *configuration.Registry, osProxyURL string) (*originalURLProvider, error) {

	// this is inefficient; we still need to get the cluster and OSO tokens so we can access metrics
	// the console, log and API urls should come from Auth or Tenant services instead of calculating in this code.
	p, err := newAuthNoProxyURLProvider(ctx, config)
	if err != nil {
		return nil, err
	}

	// all non-metric API calls go via the proxy
	p.apiURL = osProxyURL
	p.apiToken = goajwt.ContextJWT(ctx).Raw

	return p, nil
}

// using auth and proxy, access metrics via proxy (somehow)
func newFinalAuthProxyURLProvider(ctx context.Context, config *configuration.Registry, osProxyURL string) (*originalURLProvider, error) {

	provider := &originalURLProvider{
		apiURL:       osProxyURL,
		apiToken:     goajwt.ContextJWT(ctx).Raw,
		metricURL:    osProxyURL,
		metricsToken: goajwt.ContextJWT(ctx).Raw,
	}

	return provider, nil
}

// using Auth and no proxy
func newAuthNoProxyURLProvider(ctx context.Context, config *configuration.Registry) (*originalURLProvider, error) {

	p, err := newOriginalURLProvider(ctx, config)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// using Auth, no proxy (internal call)
func newOriginalURLProvider(ctx context.Context, config *configuration.Registry) (*originalURLProvider, error) {
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

	provider := &originalURLProvider{
		apiURL:       *authUser.Data.Attributes.Cluster,
		apiToken:     *osauth.AccessToken,
		metricURL:    *authUser.Data.Attributes.Cluster,
		metricsToken: *osauth.AccessToken,
	}

	return provider, nil
}

// NewTestURLProvider creates a provider with the same URL and token for both API and metrics
func NewTestURLProvider(clusterURL string, token string) BaseURLProvider {
	provider := &originalURLProvider{
		apiURL:       clusterURL,
		apiToken:     token,
		metricURL:    clusterURL,
		metricsToken: token,
	}

	return provider
}

// NewTestURLWithMetricsProvider creates a provider with the different URL and token for API and metrics
func NewTestURLWithMetricsProvider(clusterURL string, token string, metricsClusterURL string, metricsToken string) BaseURLProvider {
	provider := &originalURLProvider{
		apiURL:       clusterURL,
		apiToken:     token,
		metricURL:    metricsClusterURL,
		metricsToken: metricsToken,
	}

	return provider
}

func (up *originalURLProvider) GetAPIToken() *string {
	return &up.apiToken
}

func (up *originalURLProvider) GetMetricsToken() *string {
	return &up.metricsToken
}

func (up *originalURLProvider) GetClusterBaseURL() (*string, error) {
	return &up.apiURL, nil
}

func (up *originalURLProvider) GetAPIURL() string {
	return up.apiURL
}

func (up *originalURLProvider) GetConsoleURL(envNS string) (*string, error) {
	path := fmt.Sprintf("console/project/%s", envNS)
	// Replace "api" prefix with "console" and append path
	consoleURL, err := modifyURL(up.apiURL, "console", path)
	if err != nil {
		return nil, err
	}
	consoleURLStr := consoleURL.String()
	return &consoleURLStr, nil
}

func (up *originalURLProvider) GetLogURL(envNS string, deployName string) (*string, error) {
	consoleURL, err := up.GetConsoleURL(envNS)
	if err != nil {
		return nil, err
	}
	logURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", *consoleURL, deployName)
	return &logURL, nil
}

func (up *originalURLProvider) GetMetricsURL() (*string, error) {

	// In the absence of a better way to get the user's metrics URL,
	// substitute "api" with "metrics" in user's cluster URL

	metricsURL, err := modifyURL(up.metricURL, "metrics", "")
	if err != nil {
		return nil, err
	}
	urlstr := metricsURL.String()
	return &urlstr, nil
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