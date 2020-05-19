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
const defaultNS = "myDefaultNS"

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
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	require.Equal(t, defaultAPIURL, *apiurl, "GetAPIURL() returned wrong value")
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
	murl, err := p.GetMetricsURL(defaultNS)
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, murl)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// converts leading "api" to leading "metrics"
	apiMetricsURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//metrics.", 1), "/api", "", 1)
	require.NotEqual(t, apiMetricsURL, *murl, "GetMetricsURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterMetricsURL
	require.Equal(t, expected, *murl, "GetMetricsURL() did not return the correct value from JSON")
}

func TestTenantGetMissingMetricsURL(t *testing.T) {
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	murl, err := p.GetMetricsURL(defaultNS)
	require.NoError(t, err, "GetMetricsURL() returned an error")
	require.NotNil(t, murl)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// converts leading "api" to leading "metrics"
	apiMetricsURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//metrics.", 1), "/api", "", 1)
	require.Equal(t, apiMetricsURL, *murl, "empty or missing GetMetricsURL() must default to API URL")
}

func TestTenantGetConsoleURL(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	url, err := p.GetConsoleURL(defaultNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	apiConsoleURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//console.", 1), "/api", "", 1) + "/project/" + defaultNS
	require.NotEqual(t, apiConsoleURL, *url, "GetConsoleURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterConsoleURL
	expected = expected + "/project/" + defaultNS
	require.Equal(t, expected, *url, "GetConsoleURL() did not return the correct value from JSON")
}

func TestTenantGetMissingConsoleURL(t *testing.T) {
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	url, err := p.GetConsoleURL(defaultNS)
	require.NoError(t, err, "GetConsoleURL() returned an error")
	require.NotNil(t, url)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// Note that the Auth/Tenant appends /console to the hostname for console/logging - we have to to this here
	apiConsoleURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + defaultNS
	require.Equal(t, apiConsoleURL, *url, "GetConsoleURL()must default to API URL")
}
func TestTenantGetLoggingURL(t *testing.T) {
	const deployName = "aDeployName"
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	url, err := p.GetLoggingURL(defaultNS, deployName)
	require.NoError(t, err, "GetLoggingURL() returned an error")
	require.NotNil(t, url)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// converts leading "api" to leading "metrics"
	// Note that the Auth/Tenant appends /console to the hostname for console/logging
	apiConsoleURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//console.", 1), "/api", "", 1) + "/project/" + defaultNS
	apiLoggingURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", apiConsoleURL, deployName)
	require.NotEqual(t, apiLoggingURL, *url, "GetLoggingURL() defaulted to API URL")
	expected := *defaultTenant.Attributes.Namespaces[0].ClusterLoggingURL
	expected = expected + "/project/" + defaultNS + "/browse/rc/" + deployName + "?tab=logs"
	require.Equal(t, expected, *url, "GetLoggingURL() did not return correct value")
}

func TestTenantGetMissingLoggingURL(t *testing.T) {
	const deployName = "aDeployName"
	p, err := getBadTenantProvider()
	require.NoError(t, err)
	url, err := p.GetLoggingURL(defaultNS, deployName)
	require.NoError(t, err, "GetLoggingURL() returned an error")
	require.NotNil(t, url)
	apiurl, err := p.GetAPIURL()
	require.NoError(t, err)
	require.NotNil(t, apiurl)
	// converts leading "api" to leading "metrics"
	// Note that the Auth/Tenant appends /console to the hostname for console/logging - we have to to this here
	apiConsoleURL := strings.Replace(strings.Replace(*apiurl, "//api.", "//console.", 1), "/api", "", 1) + "/console/project/" + defaultNS
	apiLoggingURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", apiConsoleURL, deployName)
	require.Equal(t, apiLoggingURL, *url, "GetLoggingURL() must default to API URL")
}

func TestTenantGetAPIToken(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	token, err := p.GetAPIToken()
	require.NoError(t, err)
	require.NotNil(t, token)
	require.Equal(t, defaultAPIToken, *token, "GetAPIToken() did not return API token")
}

func TestTenantGetDefaultMetricsToken(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	mtoken, err := p.GetMetricsToken(defaultNS)
	require.NoError(t, err)
	require.NotNil(t, mtoken)
	require.Equal(t, defaultAPIToken, *mtoken, "GetMetricsToken() did not default to API token")
}

func TestTenantGetUnknownMetricsToken(t *testing.T) {
	p, err := getDefaultTenantProvider()
	require.NoError(t, err)
	mtoken, err := p.GetMetricsToken("unknown NS")
	require.Error(t, err)
	require.Nil(t, mtoken)
}

func TestTenantGetEnvironmentMapping(t *testing.T) {
	testCases := []struct {
		testName    string
		inputFile   string
		expectedMap map[string]string
	}{
		{
			testName:  "Basic",
			inputFile: "user-services.json",
			expectedMap: map[string]string{
				"user":    "theuser",
				"run":     "theuser-run",
				"stage":   "theuser-stage",
				"che":     "theuser-che",
				"jenkins": "theuser-jenkins",
			},
		},
		{
			testName:  "No Type",
			inputFile: "user-services-no-type.json",
			expectedMap: map[string]string{
				"user":    "theuser",
				"run":     "theuser-run",
				"che":     "theuser-che",
				"jenkins": "theuser-jenkins",
			},
		},
		{
			testName:  "Empty Type",
			inputFile: "user-services-empty-type.json",
			expectedMap: map[string]string{
				"user":    "theuser",
				"run":     "theuser-run",
				"che":     "theuser-che",
				"jenkins": "theuser-jenkins",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			userSvc, err := getTenantFromFile(testCase.inputFile)
			require.NoError(t, err, "error reading tenant")
			provider, err := controller.NewTenantURLProviderFromTenant(userSvc, defaultAPIToken, "")
			require.NoError(t, err, "error creating URL provider")

			envMap := provider.GetEnvironmentMapping()
			require.NotNil(t, envMap)
			require.Equal(t, testCase.expectedMap, envMap, "GetEnvironmentMapping() did not return the expected environments")
		})
	}
}

func TestTenantCanDeploy(t *testing.T) {
	testCases := []struct {
		envType  string
		expected bool
	}{
		{"user", false},
		{"test", true},
		{"stage", true},
		{"run", true},
		{"che", false},
		{"jenkins", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.envType, func(t *testing.T) {
			provider, err := getDefaultTenantProvider()
			require.NoError(t, err)
			result := provider.CanDeploy(testCase.envType)
			require.Equal(t, testCase.expected, result, "Incorrect result from CanDeploy")
		})
	}
}
