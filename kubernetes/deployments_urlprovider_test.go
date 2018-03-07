package kubernetes_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/stretchr/testify/require"
)

const defaultAPIURL = "https://api.hostname/api"
const defaultAPIToken = "token1"

func getDefaultProvider() kubernetes.BaseURLProvider {
	return kubernetes.NewTestURLProvider(defaultAPIURL, defaultAPIToken)
}

func getDefaultProviderWithMetrics(metricsurl string, metricstoken string) kubernetes.BaseURLProvider {
	return kubernetes.NewTestURLWithMetricsProvider(defaultAPIURL, defaultAPIToken, metricsurl, metricstoken)
}

/***
type BaseURLProvider interface {
	GetAPIURL() string
	GetMetricsURL() (*string, error)
	GetConsoleURL(envNS string) (*string, error)
	GetLogURL(envNS string, deploy *deployment) (*string, error)

	GetAPIToken() *string
	GetMetricsToken() *string
***/

func TestAPIURL(t *testing.T) {
	p := getDefaultProvider()
	require.Equal(t, defaultAPIURL, p.GetAPIURL(), "GetAPIURL() returned wrong value")
}

func TestGetDefaultMetricsURL(t *testing.T) {
	p := getDefaultProvider()
	url, err := p.GetMetricsURL()
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	expected := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//metrics.", 1), "/api", "", 1)
	require.Equal(t, expected, *url, "GetMetricsURL() did not default to API URL")
}

func TestGetNewMetricsURL(t *testing.T) {
	const mURL = "https://api.fooo/api"
	const mToken = "metricstoken"
	p := getDefaultProviderWithMetrics(mURL, mToken)
	url, err := p.GetMetricsURL()
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, url)
	expected := strings.Replace(strings.Replace(mURL, "//api.", "//metrics.", 1), "/api", "", 1)
	require.Equal(t, expected, *url, "GetMetricsURL() did not return metrics URL")
}

func TestFailBadMetricsURL(t *testing.T) {
	const badURL = "https://junk.fooo/api"
	const mToken = "metricstoken"
	p := getDefaultProviderWithMetrics(badURL, mToken)
	url, err := p.GetMetricsURL()
	require.Error(t, err, "GetMetricsURL() did not an error")
	require.Nil(t, url)
}

func TestGetConsoleURL(t *testing.T) {
	const envNS = "someEnvNS"
	p := getDefaultProvider()
	url, err := p.GetConsoleURL(envNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	expected := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + envNS
	require.Equal(t, expected, *url, "GetConsoleURL() did not return correct value")
}

func TestGetLogURL(t *testing.T) {
	const envNS = "someEnvNS"
	const deployName = "aDeployName"
	p := getDefaultProvider()
	url, err := p.GetLogURL(envNS, deployName)
	require.NoError(t, err, "GetLogURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	expected := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + envNS
	expected = fmt.Sprintf("%s/browse/rc/%s?tab=logs", expected, deployName)
	require.Equal(t, expected, *url, "GetLogURL() did not return correct value")
}

func TestGetAPIToken(t *testing.T) {
	p := getDefaultProvider()
	require.Equal(t, defaultAPIToken, *p.GetAPIToken(), "GetAPIToken() did not return API token")
}

func TestGetDefaultMetricsToken(t *testing.T) {
	p := getDefaultProvider()
	require.Equal(t, defaultAPIToken, *p.GetMetricsToken(), "GetMetricsToken() did not default to API token")
}

func TestGetMetricsToken(t *testing.T) {
	const mURL = "https://mm/m"
	const mToken = "metricstoken"
	p := getDefaultProviderWithMetrics(mURL, mToken)
	require.Equal(t, mToken, *p.GetMetricsToken(), "GetMetricsToken() did not return metrics token")
}

func xxxtostring(item interface{}) string {
	bytes, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
