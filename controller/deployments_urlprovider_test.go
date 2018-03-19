package controller_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/kubernetes"

	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

const defaultAPIURL = "https://api.hostname/api"
const defaultAPIToken = "token1"

// testing Tenant-based URL provider
var defaultTenant *app.UserService

// Path to JSON resources
const pathToURLProviderJSON = "../test/kubernetes/urlprovider-"

func getTenantFromFile(filename string) (*app.UserService, error) {
	path := pathToURLProviderJSON + filename
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	t := &app.UserService{}
	err = json.Unmarshal(jsonBytes, t)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return t, nil
}

func getDefaultTenant() (*app.UserService, error) {
	if defaultTenant == nil {
		t, err := getTenantFromFile("tenant-default.json")
		if err != nil {
			fmt.Printf("error reading tenant: %s", err.Error())
			return nil, err
		}
		//fmt.Printf("default tenant = \n%s\n", tostring(*t))
		defaultTenant = t
	}
	return defaultTenant, nil
}

func getBadTenantProvider() (kubernetes.BaseURLProvider, error) {
	t, err := getTenantFromFile("tenant-missingurls.json")
	if err != nil {
		fmt.Printf("error reading bad tenant: %s", err.Error())
		return nil, err
	}
	return controller.NewTenantURLProviderFromTenant(t, defaultAPIToken, "")
}

func getDefaultTenantProvider() (kubernetes.BaseURLProvider, error) {
	t, err := getDefaultTenant()
	if err != nil {
		return nil, err
	}
	if t == nil {
		fmt.Printf("error reading default tenant: %s", err.Error())
	}
	return controller.NewTenantURLProviderFromTenant(t, defaultAPIToken, "")
}

func TestTenantAPIURL(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, defaultAPIURL, p.GetAPIURL(), "GetAPIURL() returned wrong value")
}

func TestTenantGetMalformedData(t *testing.T) {
	us, err := getTenantFromFile("tenant-malformed.json")
	require.NoError(t, err)
	require.NotNilf(t, us, "error reading test file %s", "tenant-malformed.json")

	up, err := controller.NewTenantURLProviderFromTenant(us, defaultAPIToken, "")
	require.Nil(t, up)
	require.Error(t, err)
}

func TestTenantGetDefaultMetricsURL(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	url, err := p.GetMetricsURL()
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	apiMetricsURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//metrics.", 1), "/api", "", 1)
	require.NotEqual(t, apiMetricsURL, *url, "GetMetricsURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterMetricsURL
	require.Equal(t, expected, *url, "GetMetricsURL() did not return the correct value from JSON")
}

func TestTenantGetMissingMetricsURL(t *testing.T) {
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	url, err := p.GetMetricsURL()
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	apiMetricsURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//metrics.", 1), "/api", "", 1)
	require.Equal(t, apiMetricsURL, *url, "empty or missing GetMetricsURL() must default to API URL")
}

func TestTenantGetConsoleURL(t *testing.T) {
	const envNS = "someEnvNS"
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	url, err := p.GetConsoleURL(envNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	apiConsoleURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/project/" + envNS
	require.NotEqual(t, apiConsoleURL, *url, "GetConsoleURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterConsoleURL
	expected = expected + "/project/" + envNS
	require.Equal(t, expected, *url, "GetConsoleURL() did not return the correct value from JSON")
}

func TestTenantGetMissingConsoleURL(t *testing.T) {
	const envNS = "someEnvNS"
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	url, err := p.GetConsoleURL(envNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)
	// Note that the Auth/Tenant appends /console to the hostname for console/logging - we have to to this here
	apiConsoleURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + envNS
	require.Equal(t, apiConsoleURL, *url, "GetConsoleURL()must default to API URL")
}
func TestTenantGetLoggingURL(t *testing.T) {
	const envNS = "someEnvNS"
	const deployName = "aDeployName"
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	url, err := p.GetLoggingURL(envNS, deployName)
	require.NoError(t, err, "GetLoggingURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	apiConsoleURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/project/" + envNS
	apiLoggingURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", apiConsoleURL, deployName)
	require.NotEqual(t, apiLoggingURL, *url, "GetLoggingURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterLoggingURL
	expected = expected + "/project/" + envNS + "/browse/rc/" + deployName + "?tab=logs"
	require.Equal(t, expected, *url, "GetLoggingURL() did not return correct value")
}

func TestTenantGetMissingLoggingURL(t *testing.T) {
	const envNS = "someEnvNS"
	const deployName = "aDeployName"
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	url, err := p.GetLoggingURL(envNS, deployName)
	require.NoError(t, err, "GetLoggingURL() returned an error")
	require.NotNil(t, url)
	// converts leading "api" to leading "metrics"
	// Note that the Auth/Tenant appends /console to the hostname for console/logging - we have to to this here
	apiConsoleURL := strings.Replace(strings.Replace(p.GetAPIURL(), "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + envNS
	apiLoggingURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", apiConsoleURL, deployName)
	require.Equal(t, apiLoggingURL, *url, "GetLoggingURL() must default to API URL")
}

func TestTenantGetAPIToken(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	require.Equal(t, defaultAPIToken, *p.GetAPIToken(), "GetAPIToken() did not return API token")
}

func TestTenantGetDefaultMetricsToken(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	require.Equal(t, defaultAPIToken, *p.GetMetricsToken("somenamespace"), "GetMetricsToken() did not default to API token")
}

//////////////////////////////////////////////////////////////////////////////////////////////////

func tostring(item interface{}) string {
	bytes, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}
