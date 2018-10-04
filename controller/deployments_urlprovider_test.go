package controller_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetes"

	"github.com/stretchr/testify/require"
)

const defaultAPIURL = "https://proxy.hostname/api"
const defaultAPIToken = "token1"
const defaultNS = "myDefaultNS"

func getDefaultURLProvider() (kubernetes.BaseURLProvider, error) {
	return controller.NewProxyURLProvider(defaultAPIToken, defaultAPIURL)
}

func TestGetAPIURL(t *testing.T) {
	p, err := getDefaultURLProvider()
	require.NoError(t, err)
	require.NotNil(t, p)

	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	require.Equal(t, defaultAPIURL, *apiurl, "GetAPIURL() returned wrong value")
}

func TestGetDefaultMetricsURL(t *testing.T) {
	p, err := getDefaultURLProvider()
	require.NoError(t, err)

	murl, err := p.GetMetricsURL(defaultNS)
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, murl)

	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	require.NotEqual(t, apiurl, *murl, "GetMetricsURL() defaulted to API URL")

	expected := strings.Replace(*apiurl, "/api", "/metrics", 1)
	require.Equal(t, expected, *murl, "GetMetricsURL() has wrong value")
}

func TestGetConsoleURL(t *testing.T) {
	p, err := getDefaultURLProvider()
	require.NoError(t, err)

	url, err := p.GetConsoleURL(defaultNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)

	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	require.NotEqual(t, *apiurl, *url, "GetConsoleURL() defaulted to API URL")

	expected := strings.Replace(*apiurl, "/api", "/console", 1) + "/project/" + defaultNS
	require.Equal(t, expected, *url, "GetConsoleURL() did not return the correct value from JSON")
}

func TestGetLoggingURL(t *testing.T) {
	const deployName = "aDeployName"
	p, err := getDefaultURLProvider()
	require.NoError(t, err)

	url, err := p.GetLoggingURL(defaultNS, deployName)
	require.NoError(t, err, "GetLoggingURL() returned an error")
	require.NotNil(t, url)

	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	require.NotEqual(t, *apiurl, *url, "GetLoggingURL() defaulted to API URL")

	expected := strings.Replace(*apiurl, "/api", "/logs", 1) + "/project/" + defaultNS + "/browse/rc/" + deployName + "?tab=logs"
	require.Equal(t, expected, *url, "GetLoggingURL() did not return correct value")
}

func TestGetAPIToken(t *testing.T) {
	p, err := getDefaultURLProvider()
	require.NoError(t, err)
	token, err := p.GetAPIToken()
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t, defaultAPIToken, *token, "GetAPIToken() did not return API token")
}

func TestGetDefaultMetricsToken(t *testing.T) {
	p, err := getDefaultURLProvider()
	require.NoError(t, err)
	mtoken, err := p.GetMetricsToken(defaultNS)
	require.NoError(t, err)
	require.NotNil(t, mtoken)
	require.Equal(t, defaultAPIToken, *mtoken, "GetMetricsToken() did not return API token")
}
