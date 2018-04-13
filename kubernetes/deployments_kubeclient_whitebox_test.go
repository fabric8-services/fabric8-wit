package kubernetes

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

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
