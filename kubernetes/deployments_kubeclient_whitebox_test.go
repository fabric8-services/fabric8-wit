package kubernetes

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

// Path to test resources
const pathToTestInput = "../test/kubernetes/"

func TestGetMostRecentByDeploymentVersion(t *testing.T) {
	testCases := []struct {
		testName       string
		rcs            map[string]*v1.ReplicationController
		expectedRCName string
		shouldFail     bool
	}{
		{
			testName: "Basic",
			rcs: map[string]*v1.ReplicationController{
				"world": createRC("world", "1"),
				"hello": createRC("hello", "2"),
			},
			expectedRCName: "hello",
		},
		{
			testName: "Empty",
			rcs:      map[string]*v1.ReplicationController{},
		},
		{
			testName: "Version Not Number",
			rcs: map[string]*v1.ReplicationController{
				"world": createRC("world", "1"),
				"hello": createRC("hello", "Not a number"),
			},
			shouldFail: true,
		},
		{
			testName: "First Without Version",
			rcs: map[string]*v1.ReplicationController{
				"world": createRC("world", ""),
				"hello": createRC("hello", "2"),
			},
			expectedRCName: "hello",
		},
		{
			testName: "Second Without Version",
			rcs: map[string]*v1.ReplicationController{
				"world": createRC("world", "1"),
				"hello": createRC("hello", ""),
			},
			expectedRCName: "world",
		},
		{
			testName: "Both Without Version",
			rcs: map[string]*v1.ReplicationController{
				"hello": createRC("hello", ""),
				"world": createRC("world", ""),
			},
			expectedRCName: "world",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			result, err := getMostRecentByDeploymentVersion(testCase.rcs)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				if len(testCase.expectedRCName) == 0 {
					require.Nil(t, result, "Expected nil result")
				} else {
					require.NotNil(t, result, "Expected result to not be nil")
					require.Equal(t, testCase.expectedRCName, result.Name)
				}
			}
		})
	}
}

func createRC(name string, version string) *v1.ReplicationController {
	annotations := make(map[string]string)
	if len(version) > 0 {
		annotations["openshift.io/deployment-config.latest-version"] = version
	}
	return &v1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
	}
}

func TestGetKubeRESTAPI(t *testing.T) {
	config := getKubeConfigWithTimeout()
	getter := &defaultGetter{}
	restAPI, err := getter.GetKubeRESTAPI(config)
	require.NoError(t, err, "Error occurred getting Kubernetes REST API")

	// Get config from underlying kubeAPIClient struct
	client, ok := restAPI.(*kubeAPIClient)
	require.True(t, ok, "GetKubeRESTAPI returned %s instead of *kubeAPIClient", reflect.TypeOf(client))
	restConfig := client.restConfig
	require.NotNil(t, restConfig, "rest.Config was not stored in kubeAPIClient")
	apiURL, err := config.BaseURLProvider.GetAPIURL()
	require.NoError(t, err, "Error getting API URL")
	require.NotNil(t, apiURL)
	apiToken, err := config.BaseURLProvider.GetAPIToken()
	require.NoError(t, err, "Error getting API Token")
	require.NotNil(t, apiToken)
	require.Equal(t, *apiURL, restConfig.Host, "Host config is not set to cluster URL")
	require.Equal(t, *apiToken, restConfig.BearerToken, "Bearer tokens do not match")
	require.Equal(t, config.Timeout, restConfig.Timeout, "Timeouts do not match")
	require.Equal(t, config.Transport, restConfig.Transport, "HTTP Transports do not match")
}

func TestGetOpenShiftRESTAPI(t *testing.T) {
	config := getKubeConfigWithTimeout()
	getter := &defaultGetter{}
	restAPI, err := getter.GetOpenShiftRESTAPI(config)
	require.NoError(t, err, "Error occurred getting OpenShift REST API")

	// Check that fields are correct in underlying openShiftAPIClient struct
	client, ok := restAPI.(*openShiftAPIClient)
	require.True(t, ok, "GetOpenShiftRESTAPI returned %s instead of *openShiftAPIClient", reflect.TypeOf(client))
	require.Equal(t, config, client.config, "KubeClientConfig in OpenShift client does not match")
	require.NotNil(t, client.httpClient, "No HTTP client present in OpenShift client")
	require.Equal(t, config.Timeout, client.httpClient.Timeout, "Timeouts do not match")
	require.Equal(t, config.Transport, client.httpClient.Transport, "HTTP Transports do not match")
}

func TestGetDeploymentConfigNameForApp(t *testing.T) {
	testCases := []struct {
		testName       string
		appName        string
		expectedDCName string
		shouldFail     bool
	}{
		{
			testName:       "Basic",
			appName:        "myApp",
			expectedDCName: "myDeploy",
		},
		{
			testName:   "Not Found",
			appName:    "notFound",
			shouldFail: true,
		},
		{
			testName:   "Wrong List",
			appName:    "badList",
			shouldFail: true,
		},
		{
			testName:   "No Items",
			appName:    "noItems",
			shouldFail: true,
		},
		{
			testName:   "Build Not Object",
			appName:    "badBuild",
			shouldFail: true,
		},
		{
			testName:   "No Metadata",
			appName:    "noMeta",
			shouldFail: true,
		},
		{
			testName: "No Annotations",
			appName:  "noAnnotations",
		},
		{
			testName: "No Environment Services",
			appName:  "noEnvServices",
		},
		{
			testName: "Bad Environment Services",
			appName:  "badEnvServices",
		},
		{
			testName: "Environment Services Not YAML",
			appName:  "badYAML",
		},
		{
			testName: "No Deployment Versions",
			appName:  "noDepVer",
		},
		{
			testName: "Deployment Version Not String",
			appName:  "badDepVer",
		},
		{
			testName:       "Multiple Builds",
			appName:        "manyBuilds",
			expectedDCName: "myDeploy2",
		},
		{
			testName:   "Missing Status",
			appName:    "noStatus",
			shouldFail: true,
		},
		{
			testName: "Missing Phase",
			appName:  "noPhase",
		},
		{
			testName: "Missing Completion Timestamp",
			appName:  "noDate",
		},
		{
			testName:   "Completion Timestamp Not Date",
			appName:    "badDate",
			shouldFail: true,
		},
		{
			testName: "Empty Name",
			appName:  "emptyKey",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestInput + "get-dc-name-for-app")
			require.NoError(t, err, "Failed to open cassette")

			urlProvider := getTestURLProvider("http://api.myCluster", "myToken")
			config := &KubeClientConfig{
				BaseURLProvider: urlProvider,
				UserNamespace:   "myNamespace",
				Transport:       r.Transport,
			}
			kcInterface, err := NewKubeClient(config)
			require.NoError(t, err, "Error occurred creating KubeClient")
			kc, ok := kcInterface.(*kubeClient)
			require.True(t, ok, "NewKubeClient should return *kubeClient type")

			dcName, err := kc.getDeploymentConfigNameForApp("my-run", testCase.appName, "mySpace")
			if testCase.shouldFail {
				require.Error(t, err, "Test case expects an error from getDeploymentConfigNameForApp")
			} else {
				require.NoError(t, err, "getDeploymentConfigNameForApp should return without error")
				// Fallback is to use application name
				expectedDCName := testCase.expectedDCName
				if len(expectedDCName) == 0 {
					expectedDCName = testCase.appName
				}
				require.Equal(t, expectedDCName, dcName, "getDeploymentConfigNameForApp returned incorrect name")
			}
		})
	}
}

func getKubeConfigWithTimeout() *KubeClientConfig {
	urlProvider := getTestURLProvider("http://api.myCluster", "myToken")
	return &KubeClientConfig{
		BaseURLProvider: urlProvider,
		UserNamespace:   "myNamespace",
		Timeout:         30 * time.Second,
		Transport:       &http.Transport{},
	}
}

type testURLProvider struct {
	apiURL   string
	apiToken string
}

func getTestURLProvider(baseurl string, token string) BaseURLProvider {
	return &testURLProvider{
		apiURL:   baseurl,
		apiToken: token,
	}
}

// code for test URL provider

func (up *testURLProvider) GetAPIToken() (*string, error) {
	return &up.apiToken, nil
}

func (up *testURLProvider) GetMetricsToken(envNS string) (*string, error) {
	return &up.apiToken, nil
}

func (up *testURLProvider) GetAPIURL() (*string, error) {
	return &up.apiURL, nil
}

func (up *testURLProvider) GetConsoleURL(envNS string) (*string, error) {
	return &up.apiURL, nil
}

func (up *testURLProvider) GetLoggingURL(envNS string, deployName string) (*string, error) {
	return &up.apiURL, nil
}

func (up *testURLProvider) GetMetricsURL(envNS string) (*string, error) {
	return &up.apiURL, nil
}
