package controller

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"

	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	errs "github.com/pkg/errors"
)

// This incarnation uses the proxy for ALL OSO API calls and will not function without a proxy

type deploymentsURLProvider struct {
	apiURL   string
	apiToken string
	kubernetes.BaseURLProvider
}

// ensure deploymentsURLProvider implements BaseURLProvider
var _ kubernetes.BaseURLProvider = &deploymentsURLProvider{}
var _ kubernetes.BaseURLProvider = (*deploymentsURLProvider)(nil)

// NewURLProvider looks at what servers are available and create a BaseURLProvder that fits
func NewURLProvider(ctx context.Context, config *configuration.Registry, osioclient OpenshiftIOClient) (kubernetes.BaseURLProvider, error) {

	token := goajwt.ContextJWT(ctx).Raw
	proxyURL := config.GetOpenshiftProxyURL()

	if len(proxyURL) == 0 {
		log.Error(ctx, map[string]interface{}{}, "No Proxy URL configured")
		return nil, errs.Errorf("No Proxy URL configured")
	}

	up, err := NewProxyURLProvider(token, proxyURL)
	if err != nil {
		return nil, err
	}

	return up, nil
}

// NewProxyURLProvider create a provider from a UserService object (exposed for testing)
func NewProxyURLProvider(token string, proxyURL string) (kubernetes.BaseURLProvider, error) {
	provider := &deploymentsURLProvider{
		apiURL:   proxyURL,
		apiToken: token,
	}
	return provider, nil
}

func (up *deploymentsURLProvider) GetAPIToken() (*string, error) {
	return &up.apiToken, nil
}

func (up *deploymentsURLProvider) GetAPIURL() (*string, error) {
	// TODO this may be different for every namespace if no proxy
	return &up.apiURL, nil
}

func (up *deploymentsURLProvider) GetMetricsToken(envNS string) (*string, error) {
	return &up.apiToken, nil
}

func (up *deploymentsURLProvider) GetConsoleURL(envNS string) (*string, error) {
	mu, err := modifyPath(up.apiURL, "/console")
	if err != nil {
		return nil, err
	}
	urlStr := mu.String()

	consoleURL := fmt.Sprintf("%s/project/%s", urlStr, envNS)
	return &consoleURL, nil
}

func (up *deploymentsURLProvider) GetLoggingURL(envNS string, deployName string) (*string, error) {
	mu, err := modifyPath(up.apiURL, "/logs")
	if err != nil {
		return nil, err
	}
	urlStr := mu.String()

	loggingURL := fmt.Sprintf("%s/project/%s/browse/rc/%s?tab=logs", urlStr, envNS, deployName)
	return &loggingURL, nil
}

func (up *deploymentsURLProvider) GetMetricsURL(envNS string) (*string, error) {

	// substitute "api" with "metrics" in user's cluster URL
	mu, err := modifyPath(up.apiURL, "/metrics")
	if err != nil {
		return nil, err
	}
	urlStr := mu.String()

	return &urlStr, nil
}

func modifyPath(apiURLStr string, path string) (*url.URL, error) {
	// Parse as URL to give us easy access to the hostname
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Construct URL using just scheme from API URL, modified hostname and supplied path
	newURL := &url.URL{
		Scheme: apiURL.Scheme,
		Host:   apiURL.Hostname(),
		Path:   path,
	}
	return newURL, nil
}
