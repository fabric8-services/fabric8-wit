package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// there are several concrete instantiations:
//
//   - access /api/user/services instead of Auth
//   - use proxy (if present) for normal OSO API calls
//   - access OSO metrics directly (until proxy supports this)

type tenantURLProvider struct {
	apiURL     string
	apiToken   string
	tenant     *app.UserService
	namespaces map[string]*app.NamespaceAttributes
	tokens     map[string]string
	TokenRetriever
}

// TokenRetriever is a service that will respond with an authorization token, given a service endpoint or name
type TokenRetriever interface {
	TokenForService(serviceURL string) (*string, error)
}

type tokenRetriever struct {
	authClient *authservice.Client
	context    context.Context
}

func (tr *tokenRetriever) TokenForService(forService string) (*string, error) {

	resp, err := tr.authClient.RetrieveToken(goasupport.ForwardContextRequestID(tr.context), authservice.RetrieveTokenPath(), forService, nil)
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

	return respType.AccessToken, nil
}

// ensure tenantURLProvider implements BaseURLProvider
var _ kubernetes.BaseURLProvider = &tenantURLProvider{}
var _ kubernetes.BaseURLProvider = (*tenantURLProvider)(nil)

// NewURLProvider looks at what servers are available and create a BaseURLProvder that fits
func NewURLProvider(ctx context.Context, config *configuration.Registry, osioclient OpenshiftIOClient) (kubernetes.BaseURLProvider, error) {

	userServices, err := osioclient.GetUserServices(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Tenant API")
		return nil, err
	}

	token := goajwt.ContextJWT(ctx).Raw
	proxyURL := config.GetOpenshiftProxyURL()

	up, err := newTenantURLProviderFromTenant(userServices, token, proxyURL)
	if err != nil {
		return nil, err
	}

	// create Auth API client - required to get OSO tokens
	authClient, err := auth.CreateClient(ctx, config)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Auth server")
		return nil, errs.Wrap(err, "error creating Auth client")
	}
	up.TokenRetriever = &tokenRetriever{
		authClient: authClient,
		context:    ctx,
	}

	// if we're not using a proxy then the API URL is actually the cluster of namespace 0,
	// so the apiToken should be the token for that cluster.
	// there should be no defaults, but that's deferred later
	if len(proxyURL) == 0 {
		tokenData, err := up.TokenForService(up.apiURL)
		if err != nil {
			return nil, err
		}
		up.apiToken = *tokenData
	}
	return up, nil
}

// newTenantURLProviderFromTenant create a provider from a UserService object
func newTenantURLProviderFromTenant(t *app.UserService, token string, proxyURL string) (*tenantURLProvider, error) {

	if t.ID == nil {
		log.Error(nil, map[string]interface{}{}, "app.UserService is malformed: no ID field")
		return nil, errs.New("app.UserService is malformed: no ID field")
	}

	if t.Attributes == nil {
		log.Error(nil, map[string]interface{}{
			"tenant": *t.ID,
		}, "app.UserService is malformed: no Attribute field ID=%s", *t.ID)
		return nil, errs.Errorf("app.UserService is malformed: no Attribute field (ID=%s)", *t.ID)
	}

	if len(t.Attributes.Namespaces) == 0 {
		log.Error(nil, map[string]interface{}{
			"tenant": *t.ID,
		}, "this tenant has no namespaces: %s", *t.ID)
		return nil, errs.Errorf("app.UserService is malformed: no Namespaces (ID=%s)", *t.ID)
	}

	defaultNamespace := t.Attributes.Namespaces[0]
	namespaceMap := make(map[string]*app.NamespaceAttributes)
	for i, namespace := range t.Attributes.Namespaces {
		namespaceMap[*namespace.Name] = t.Attributes.Namespaces[i]
		if *namespace.Type == "user" {
			defaultNamespace = namespace
		}
	}

	defaultClusterURL := *defaultNamespace.ClusterURL

	if len(proxyURL) != 0 {
		// all non-metric API calls go via the proxy
		defaultClusterURL = proxyURL
	}

	provider := &tenantURLProvider{
		apiURL:     defaultClusterURL,
		apiToken:   token,
		tenant:     t,
		namespaces: namespaceMap,
	}
	return provider, nil
}

// NewTenantURLProviderFromTenant create a provider from a UserService object (exposed for testing)
func NewTenantURLProviderFromTenant(t *app.UserService, token string, proxyURL string) (kubernetes.BaseURLProvider, error) {
	return newTenantURLProviderFromTenant(t, token, proxyURL)
}

func (up *tenantURLProvider) GetAPIToken() (*string, error) {
	return &up.apiToken, nil
}

func (up *tenantURLProvider) GetAPIURL() (*string, error) {
	// TODO this may be different for every namespace if no proxy
	return &up.apiURL, nil
}

func (up *tenantURLProvider) GetMetricsToken(envNS string) (*string, error) {
	// since metrics bypasses the proxy, this is the OSO cluster token
	token := up.tokens[envNS]
	if len(token) == 0 {
		ns := up.namespaces[envNS]
		if ns == nil {
			return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.ID)
		}
		if up.TokenRetriever != nil {
			tokenData, err := up.TokenForService(*ns.ClusterURL)
			if err != nil {
				return nil, err
			}
			token = *tokenData
		} else {
			tokenData, err := up.GetAPIToken()
			if err != nil {
				return nil, err
			}
			token = *tokenData
		}
	}
	return &token, nil
}

func (up *tenantURLProvider) GetConsoleURL(envNS string) (*string, error) {
	ns := up.namespaces[envNS]
	if ns == nil {
		return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.ID)
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
		return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.ID)
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

func (up *tenantURLProvider) GetMetricsURL(envNS string) (*string, error) {
	ns := up.namespaces[envNS]
	if ns == nil {
		return nil, errs.Errorf("Namespace '%s' is not in tenant '%s'", envNS, *up.tenant.ID)
	}

	baseURL := ns.ClusterMetricsURL
	if baseURL == nil || len(*baseURL) == 0 {
		// In the absence of a better way (i.e. tenant) to get the user's metrics URL,
		// substitute "api" with "metrics" in user's cluster URL
		mu, err := modifyURL(*ns.ClusterURL, "metrics", "")
		if err != nil {
			return nil, err
		}
		muStr := mu.String()
		baseURL = &muStr
	}
	// Hawkular implementation is sensitive and requires no trailing '/'
	if strings.HasSuffix(*baseURL, "/") {
		nurl := (*baseURL)[:len(*baseURL)-1]
		baseURL = &nurl
	}
	return baseURL, nil
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
