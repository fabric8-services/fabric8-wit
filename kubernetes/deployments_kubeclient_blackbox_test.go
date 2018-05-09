package kubernetes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/kubernetes"

	errs "github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
)

// Used for equality comparisons between float64s
const fltDelta = 0.00000001

// Path to JSON resources
const pathToTestJSON = "../test/kubernetes/"

type testFixture struct {
	metricsInput *metricsInput
	metrics      *testMetrics
}

type testURLProvider struct {
	apiURL       string
	apiToken     string
	clusterURL   string
	clusterToken string
}

func getDefaultURLProvider(baseurl string, token string) kubernetes.BaseURLProvider {
	return &testURLProvider{
		apiURL:       baseurl,
		apiToken:     token,
		clusterURL:   baseurl,
		clusterToken: token,
	}
}

func getDefaultKubeClient(fixture *testFixture, transport http.RoundTripper, t *testing.T) kubernetes.KubeClientInterface {

	config := &kubernetes.KubeClientConfig{
		BaseURLProvider: getDefaultURLProvider("http://api.myCluster", "myToken"),
		UserNamespace:   "myNamespace",
		MetricsGetter:   fixture,
		Transport:       transport,
	}

	kc, err := kubernetes.NewKubeClient(config)

	require.NoError(t, err)
	return kc
}

// Metrics fakes

type metricsHolder struct {
	pods      []*v1.Pod
	namespace string
	startTime time.Time
	endTime   time.Time
	limit     int
}

type metricsInput struct {
	cpu    []*app.TimedNumberTuple
	memory []*app.TimedNumberTuple
	netTx  []*app.TimedNumberTuple
	netRx  []*app.TimedNumberTuple
}

var defaultMetricsInput = &metricsInput{
	cpu: []*app.TimedNumberTuple{
		createTuple(1.2, 1517867612000),
		createTuple(0.7, 1517867613000),
	},
	memory: []*app.TimedNumberTuple{
		createTuple(1200, 1517867612000),
		createTuple(3000, 1517867613000),
	},
	netTx: []*app.TimedNumberTuple{
		createTuple(300, 1517867612000),
		createTuple(520, 1517867613000),
	},
	netRx: []*app.TimedNumberTuple{
		createTuple(700, 1517867612000),
		createTuple(100, 1517867613000),
	},
}

func createTuple(val float64, ts float64) *app.TimedNumberTuple {
	return &app.TimedNumberTuple{
		Value: &val,
		Time:  &ts,
	}
}

type testMetrics struct {
	config      *kubernetes.MetricsClientConfig
	fixture     *testFixture
	cpuParams   *metricsHolder
	memParams   *metricsHolder
	netTxParams *metricsHolder
	netRxParams *metricsHolder
	closed      bool
}

func (fixture *testFixture) GetMetrics(config *kubernetes.MetricsClientConfig) (kubernetes.Metrics, error) {
	metrics := &testMetrics{
		fixture: fixture,
		config:  config,
	}
	fixture.metrics = metrics
	return metrics, nil
}

func (tm *testMetrics) GetCPUMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return tm.getOneMetric(tm.fixture.metricsInput.cpu, pods, namespace, startTime, &tm.cpuParams)
}

func (tm *testMetrics) GetCPUMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
	limit int) ([]*app.TimedNumberTuple, error) {
	return tm.getManyMetrics(tm.fixture.metricsInput.cpu, pods, namespace, startTime, endTime, limit, &tm.cpuParams)
}

func (tm *testMetrics) GetMemoryMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return tm.getOneMetric(tm.fixture.metricsInput.memory, pods, namespace, startTime, &tm.memParams)
}

func (tm *testMetrics) GetMemoryMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
	limit int) ([]*app.TimedNumberTuple, error) {
	return tm.getManyMetrics(tm.fixture.metricsInput.memory, pods, namespace, startTime, endTime, limit, &tm.memParams)
}

func (tm *testMetrics) GetNetworkSentMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return tm.getOneMetric(tm.fixture.metricsInput.netTx, pods, namespace, startTime, &tm.netTxParams)
}

func (tm *testMetrics) GetNetworkSentMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
	limit int) ([]*app.TimedNumberTuple, error) {
	return tm.getManyMetrics(tm.fixture.metricsInput.netTx, pods, namespace, startTime, endTime, limit, &tm.netTxParams)
}

func (tm *testMetrics) GetNetworkRecvMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return tm.getOneMetric(tm.fixture.metricsInput.netRx, pods, namespace, startTime, &tm.netRxParams)
}

func (tm *testMetrics) GetNetworkRecvMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
	limit int) ([]*app.TimedNumberTuple, error) {
	return tm.getManyMetrics(tm.fixture.metricsInput.netRx, pods, namespace, startTime, endTime, limit, &tm.netRxParams)
}

func (tm *testMetrics) getOneMetric(metrics []*app.TimedNumberTuple, pods []*v1.Pod, namespace string,
	startTime time.Time, holderPtr **metricsHolder) (*app.TimedNumberTuple, error) {
	*holderPtr = &metricsHolder{
		pods:      pods,
		namespace: namespace,
		startTime: startTime,
	}
	return metrics[0], nil
}

func (tm *testMetrics) Close() {
	tm.closed = true
}

func (tm *testMetrics) getManyMetrics(metrics []*app.TimedNumberTuple, pods []*v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int, holderPtr **metricsHolder) ([]*app.TimedNumberTuple, error) {
	*holderPtr = &metricsHolder{
		pods:      pods,
		namespace: namespace,
		startTime: startTime,
		endTime:   endTime,
		limit:     limit,
	}
	return metrics, nil
}

func TestGetMetrics(t *testing.T) {
	token := "myToken"
	testCases := []struct {
		name          string
		clusterURL    string
		expectedURL   string
		cassetteName  string
		shouldSucceed bool
	}{
		{"Basic", "https://api.myCluster.url:443/cluster", "https://metrics.myCluster.url", "newkubeclient-withport", true},
		{"Bad URL", "https://myCluster.url:443/cluster", "", "newkubeclient-badurl", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			config := &kubernetes.KubeClientConfig{
				BaseURLProvider: getDefaultURLProvider(testCase.clusterURL, token),
				UserNamespace:   "myNamespace",
				MetricsGetter:   fixture,
				Transport:       r.Transport,
			}
			kc, err := kubernetes.NewKubeClient(config)
			if testCase.shouldSucceed {
				require.NoError(t, err, "Unexpected error")
				require.NotNil(t, kc)
				mm, err := kc.GetMetricsClient("myNamespace")
				require.NoError(t, err)
				require.NotNil(t, mm)
				metricsConfig := fixture.metrics.config
				require.NotNil(t, metricsConfig, "Metrics config is nil")
				require.Equal(t, testCase.expectedURL, metricsConfig.MetricsURL, "Incorrect Metrics URL")
				require.Equal(t, token, metricsConfig.BearerToken, "Incorrect bearer token")
			} else {
				// bad URLs aren't detected until a metrics client tries to use themn
				_, err := kc.GetMetricsClient("myNamespace")
				require.Errorf(t, err, "URL %s should fail", testCase.clusterURL)
			}
		})
	}
}

// ensure testFixture implements all of MetricsGetter
var _ kubernetes.MetricsGetter = &testFixture{}
var _ kubernetes.MetricsGetter = (*testFixture)(nil)

func TestClose(t *testing.T) {
	r, err := recorder.New(pathToTestJSON + "newkubeclient")
	require.NoError(t, err, "Failed to open cassette")
	defer r.Stop()

	fixture := &testFixture{}
	kc := getDefaultKubeClient(fixture, r.Transport, t)

	mm, err := kc.GetMetricsClient("myNamespace")
	require.NoError(t, err)
	require.NotNil(t, mm)

	// Check that KubeClientInterface.Close invokes MetricsInterface.Close
	kc.Close()

	require.True(t, fixture.metrics.closed, "Metrics client not closed")
}

func TestConfigMapEnvironments(t *testing.T) {
	testCases := []struct {
		name         string
		cassetteName string
		shouldFail   bool
	}{
		{
			name:         "Basic",
			cassetteName: "newkubeclient",
		},
		{
			name:         "Empty Data",
			cassetteName: "newkubeclient-empty",
		},
		{
			name:         "Missing Colon",
			cassetteName: "newkubeclient-nocolon",
			shouldFail:   true,
		},
		{
			name:         "Missing Namespace",
			cassetteName: "newkubeclient-nonamespace",
			shouldFail:   true,
		},
		{
			name:         "No Provider",
			cassetteName: "newkubeclient-noprovider",
			shouldFail:   true,
		},
	}
	fixture := &testFixture{}
	userNamespace := "myNamespace"
	config := &kubernetes.KubeClientConfig{
		BaseURLProvider: getDefaultURLProvider("http://api.myCluster", "myToken"),
		UserNamespace:   userNamespace,
		MetricsGetter:   fixture,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			config.Transport = r.Transport

			kc, err := kubernetes.NewKubeClient(config)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err)
				require.NotNil(t, kc, "KubeClient must not be nil")
			}
		})
	}
}

type envTestData struct {
	envName string
	cpuUsed float64
	cpuHard float64
	memUsed float64
	memHard float64
}

func TestGetEnvironments(t *testing.T) {
	testCases := []struct {
		testName     string
		cassetteName string
		shouldFail   bool
		data         map[string]*envTestData
	}{
		{
			testName:     "Basic",
			cassetteName: "getenvironments",
			data: map[string]*envTestData{
				"run": {
					envName: "run",
					cpuUsed: 0.488,
					cpuHard: 2.0,
					memUsed: 262144000.0,
					memHard: 1073741824.0,
				},
				"stage": {
					envName: "stage",
					cpuUsed: 1.488,
					cpuHard: 2.0,
					memUsed: 799014912.0,
					memHard: 1073741824.0,
				},
				"test": {
					envName: "test",
					cpuUsed: 0.0,
					cpuHard: 2.0,
					memUsed: 0.0,
					memHard: 1073741824.0,
				},
			},
		},
		{
			testName:     "Missing Quota",
			cassetteName: "getenvironments-missingquota",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			envs, err := kc.GetEnvironments()
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")

				require.Equal(t, len(testCase.data), len(envs), "Wrong number of environments returned")
				for _, env := range envs {
					require.NotNil(t, env, "Environment must not be nil")
					require.NotNil(t, env.Attributes, "Environment attributes are nil")
					require.NotNil(t, env.Attributes.Name, "Environment name must not be nil")

					envName := *env.Attributes.Name
					envData := testCase.data[envName]
					require.NotNil(t, envData, "Unknown app: "+envName)
					require.Equal(t, envData.envName, envName, "Wrong environment name")
					verifyEnvironment(env, envData, t)
				}
			}
		})
	}
}

func TestGetEnvironment(t *testing.T) {
	testCases := []struct {
		testName     string
		cassetteName string
		shouldFail   bool
		envTestData
	}{
		{
			testName:     "Basic",
			cassetteName: "getenvironments",
			envTestData: envTestData{
				envName: "run",
				cpuUsed: 0.488,
				cpuHard: 2.0,
				memUsed: 262144000.0,
				memHard: 1073741824.0,
			},
		},
		{
			testName:     "Bad Environment",
			cassetteName: "getenvironments",
			envTestData: envTestData{
				envName: "doesNotExist",
			},
			shouldFail: true,
		},
		{
			testName:     "Missing Quota",
			cassetteName: "getenvironments-missingquota",
			envTestData: envTestData{
				envName: "run",
			},
			shouldFail: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			env, err := kc.GetEnvironment(testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")

				require.NotNil(t, env, "Environment must not be nil")
				require.NotNil(t, env.Attributes, "Environment attributes are nil")
				require.NotNil(t, env.Attributes.Name, "Environment name must not be nil")

				verifyEnvironment(env, &testCase.envTestData, t)
			}
		})
	}
}

func verifyEnvironment(env *app.SimpleEnvironment, testCase *envTestData, t *testing.T) {
	require.Equal(t, testCase.envName, *env.Attributes.Name, "Wrong environment name")

	cpuQuota := env.Attributes.Quota.Cpucores
	require.InDelta(t, testCase.cpuHard, *cpuQuota.Quota, fltDelta, "Incorrect CPU quota for %s", testCase.envName)
	require.InDelta(t, testCase.cpuUsed, *cpuQuota.Used, fltDelta, "Incorrect CPU usage for %s", testCase.envName)

	memQuota := env.Attributes.Quota.Memory
	require.InDelta(t, testCase.memHard, *memQuota.Quota, fltDelta, "Incorrect memory quota for %s", testCase.envName)
	require.InDelta(t, testCase.memUsed, *memQuota.Used, fltDelta, "Incorrect memory usage for %s", testCase.envName)
}

type spaceTestData struct {
	testName     string
	spaceName    string
	shouldFail   bool
	appTestData  map[string]*appTestData // Keys are app names
	cassetteName string
}

var defaultSpaceTestData = &spaceTestData{
	testName:     "Basic",
	spaceName:    "mySpace",
	appTestData:  map[string]*appTestData{"myApp": defaultAppTestData},
	cassetteName: "getspace",
}

type appTestData struct {
	testName       string
	spaceName      string
	appName        string
	shouldFail     bool
	deployTestData map[string]*deployTestData // Keys are environment names
	cassetteName   string
}

var defaultAppTestData = &appTestData{
	testName:       "Basic",
	spaceName:      "mySpace",
	appName:        "myApp",
	deployTestData: map[string]*deployTestData{"run": defaultDeployTestData},
	cassetteName:   "getdeployment",
}

type deployTestData struct {
	testName                string
	spaceName               string
	appName                 string
	envName                 string
	expectVersion           string
	expectPodStatus         [][]string
	expectPodsTotal         int
	expectPodsQuotaCpucores float64
	expectPodsQuotaMemory   float64
	expectConsoleURL        string
	expectLogURL            string
	expectAppURL            string
	shouldFail              bool
	cassetteName            string
}

var defaultDeployTestData = &deployTestData{
	testName:      "Basic",
	spaceName:     "mySpace",
	appName:       "myApp",
	envName:       "run",
	expectVersion: "1.0.2",
	expectPodStatus: [][]string{
		{"Running", "2"},
	},
	expectPodsQuotaCpucores: 0.976,
	expectPodsQuotaMemory:   524288000,
	expectPodsTotal:         2,
	expectConsoleURL:        "http://console.myCluster/console/project/my-run",
	expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myDeploy-1?tab=logs",
	expectAppURL:            "http://myDeploy-my-run.example.com",
	cassetteName:            "getdeployment",
}

type deployStatsTestData struct {
	testName      string
	spaceName     string
	appName       string
	envName       string
	envNS         string
	cassetteName  string
	shouldFail    bool
	metricsInput  *metricsInput
	startTime     time.Time
	endTime       time.Time
	expectStart   int64
	expectEnd     int64
	expectPodUIDs []string
	limit         int
}

var defaultDeployStatsTestData = &deployStatsTestData{
	testName:     "Basic",
	spaceName:    "mySpace",
	appName:      "myApp",
	envName:      "run",
	envNS:        "my-run",
	cassetteName: "getdeployment",
	startTime:    convertToTime(1517867603000),
	endTime:      convertToTime(1517867643000),
	expectStart:  1517867612000,
	expectEnd:    1517867613000,
	expectPodUIDs: []string{
		"f04e8f3b-5c4a-4ffd-94ec-0e8bcbc7b468",
		"447b7d6f-7072-4e9a-8cba-7e29c2f53761",
	},
	limit:        10,
	metricsInput: defaultMetricsInput,
}

func convertToTime(unixMillis int64) time.Time {
	return time.Unix(0, unixMillis*int64(time.Millisecond))
}

func TestGetSpace(t *testing.T) {
	testCases := []*spaceTestData{
		defaultSpaceTestData,
		{
			testName:     "Empty List",
			spaceName:    "mySpace",
			cassetteName: "getspace-empty-bc",
		},
		{
			testName:     "Wrong List",
			spaceName:    "mySpace",
			cassetteName: "getspace-wrong-list",
			shouldFail:   true,
		},
		{
			testName:     "No Items",
			spaceName:    "mySpace",
			cassetteName: "getspace-no-items",
			shouldFail:   true,
		},
		{
			testName:     "Not Object",
			spaceName:    "mySpace",
			cassetteName: "getspace-not-object",
			shouldFail:   true,
		},
		{
			testName:     "No Metadata",
			spaceName:    "mySpace",
			cassetteName: "getspace-no-metadata",
			shouldFail:   true,
		},
		{
			testName:     "No Name",
			spaceName:    "mySpace",
			cassetteName: "getspace-no-name",
			shouldFail:   true,
		},
		{
			testName:     "Two Apps One Deployed",
			spaceName:    "mySpace", // Test two BCs, but only one DC
			cassetteName: "getspace-two-apps-one-deploy",
			appTestData: map[string]*appTestData{
				"myApp": defaultAppTestData,
				"myOtherApp": {
					spaceName: "mySpace",
					appName:   "myOtherApp",
				},
			},
		},
		{
			testName:     "Two Apps Both Deployed",
			spaceName:    "mySpace", // Test two deployed applications, with two environments
			cassetteName: "getspace-two-apps-two-deploy",
			appTestData: map[string]*appTestData{
				"myApp": {
					spaceName: "mySpace",
					appName:   "myApp",
					deployTestData: map[string]*deployTestData{
						"run": {
							spaceName:     "mySpace",
							appName:       "myApp",
							envName:       "run",
							expectVersion: "1.0.2",
							expectPodStatus: [][]string{
								{"Running", "2"},
							},
							expectPodsTotal:         2,
							expectPodsQuotaCpucores: 0.976,
							expectPodsQuotaMemory:   524288000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-run",
							expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myDeploy-1?tab=logs",
							expectAppURL:            "http://myDeploy-my-run.example.com",
						},
						"stage": {
							spaceName:     "mySpace",
							appName:       "myApp",
							envName:       "stage",
							expectVersion: "1.0.3",
							expectPodStatus: [][]string{
								{"Running", "1"},
								{"Terminating", "1"},
							},
							expectPodsTotal:         2,
							expectPodsQuotaCpucores: 0.976,
							expectPodsQuotaMemory:   524288000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-stage",
							expectLogURL:            "http://console.myCluster/console/project/my-stage/browse/rc/myDeploy-1?tab=logs",
						},
					},
				},
				"myOtherApp": {
					spaceName: "mySpace",
					appName:   "myOtherApp",
					deployTestData: map[string]*deployTestData{
						"run": {
							spaceName:     "mySpace",
							appName:       "myOtherApp",
							envName:       "run",
							expectVersion: "1.0.1",
							expectPodStatus: [][]string{
								{"Running", "1"},
							},
							expectPodsTotal:         1,
							expectPodsQuotaCpucores: 0.488,
							expectPodsQuotaMemory:   262144000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-run",
							expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myOtherDeploy-1?tab=logs",
							expectAppURL:            "http://myOtherDeploy-my-run.example.com",
						},
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			space, err := kc.GetSpace(testCase.spaceName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.NotNil(t, space, "Space is nil")
				require.NotNil(t, space.Attributes, "Space attributes are nil")
				require.Equal(t, testCase.spaceName, space.Attributes.Name, "Space name is incorrect")
				require.NotNil(t, space.Attributes.Applications, "Applications are nil")
				require.Equal(t, len(testCase.appTestData), len(space.Attributes.Applications), "Wrong number of applications")
				for _, app := range space.Attributes.Applications {
					var appInput *appTestData
					if app != nil {
						appInput = testCase.appTestData[app.Attributes.Name]
						require.NotNil(t, appInput, "Unknown app: "+app.Attributes.Name)
					}
					verifyApplication(app, appInput, t)
				}
			}
		})
	}
}

func TestGetApplication(t *testing.T) {
	testCases := []*appTestData{
		defaultAppTestData,
		{
			testName:  "Two Environments",
			spaceName: "mySpace",
			appName:   "myApp",
			deployTestData: map[string]*deployTestData{
				"run": defaultDeployTestData,
				"stage": {
					spaceName:     "mySpace",
					appName:       "myApp",
					envName:       "stage",
					expectVersion: "1.0.3",
					expectPodStatus: [][]string{
						{"Running", "1"},
						{"Terminating", "1"},
					},
					expectPodsTotal:         2,
					expectPodsQuotaCpucores: 0.976,
					expectPodsQuotaMemory:   524288000,
					expectConsoleURL:        "http://console.myCluster/console/project/my-stage",
					expectLogURL:            "http://console.myCluster/console/project/my-stage/browse/rc/myDeploy-1?tab=logs",
				},
			},
			cassetteName: "getapplication",
		},
		{
			testName:  "No Pods",
			spaceName: "mySpace", // Test deployment with no pods
			appName:   "myOtherApp",
			deployTestData: map[string]*deployTestData{
				"run": {
					envName:          "run",
					expectVersion:    "1.0.1",
					expectConsoleURL: "http://console.myCluster/console/project/my-run",
					expectLogURL:     "http://console.myCluster/console/project/my-run/browse/rc/myOtherDeploy-1?tab=logs",
					expectAppURL:     "http://myOtherDeploy-my-run.example.com",
				},
			},
			cassetteName: "getapplication-nopods",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			app, err := kc.GetApplication(testCase.spaceName, testCase.appName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				verifyApplication(app, testCase, t)
			}
		})
	}
}

func TestGetDeployment(t *testing.T) {
	testCases := []*deployTestData{
		defaultDeployTestData,
		{
			testName:     "Bad Environment",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "doesNotExist",
			cassetteName: "getdeployment",
			shouldFail:   true,
		},
		{
			// Verifies that a newer scaled down deployment is favoured over
			// an older one with replicas and an even newer deployment that failed
			testName:         "Scaled Down",
			spaceName:        "mySpace",
			appName:          "myApp",
			envName:          "run",
			expectVersion:    "1.0.2",
			expectPodStatus:  [][]string{},
			expectPodsTotal:  0,
			expectConsoleURL: "http://console.myCluster/console/project/my-run",
			expectLogURL:     "http://console.myCluster/console/project/my-run/browse/rc/myDeploy-2?tab=logs",
			expectAppURL:     "http://myDeploy-my-run.example.com",
			// Contains RCs in ascending deployment version:
			// 1. Visible 2. Scaled-down "active" 3. Failed
			cassetteName: "getdeployment-scaled-down",
		},
		{
			// Tests handling of a deployment config with missing space label
			// FIXME When our workaround is no longer needed, we should expect
			// an error
			testName:      "No Space Label",
			spaceName:     "mySpace",
			appName:       "myApp",
			envName:       "run",
			expectVersion: "1.0.2",
			expectPodStatus: [][]string{
				{"Running", "2"},
			},
			expectPodsTotal:         2,
			expectPodsQuotaCpucores: 0.976,
			expectPodsQuotaMemory:   524288000,
			expectConsoleURL:        "http://console.myCluster/console/project/my-run",
			expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myDeploy-1?tab=logs",
			expectAppURL:            "http://myDeploy-my-run.example.com",
			cassetteName:            "getdeployment-nospace",
		},
		{
			// Tests that we don't accept deployment configs with a space
			// label different from the argument passed to GetDeployment
			testName:     "Wrong Space Label",
			spaceName:    "myWrongSpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "getdeployment-wrongspace",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			dep, err := kc.GetDeployment(testCase.spaceName, testCase.appName, testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				verifyDeployment(dep, testCase, t)
			}
		})
	}
}

func TestScaleDeployment(t *testing.T) {
	testCases := []struct {
		testName      string
		spaceName     string
		appName       string
		envName       string
		expectPutURLs map[string]struct{}
		cassetteName  string
		newReplicas   int
		oldReplicas   int
		shouldFail    bool
	}{
		{
			testName:     "Basic",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "scaledeployment",
			expectPutURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myDeploy/scale": {},
			},
			newReplicas: 3,
			oldReplicas: 2,
		},
		{
			testName:     "Zero Replicas",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "scaledeployment-zero",
			expectPutURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myDeploy/scale": {},
			},
			newReplicas: 1,
			oldReplicas: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			r.SetMatcher(func(actual *http.Request, expected cassette.Request) bool {
				if cassette.DefaultMatcher(actual, expected) {
					// Check scale request body when sending PUT
					if actual.Method == "PUT" {
						var buf bytes.Buffer
						reqBody := actual.Body
						_, err := buf.ReadFrom(reqBody)
						require.NoError(t, err, "Error reading request body")
						defer reqBody.Close()

						// Mark interaction as seen
						reqURL := actual.URL.String()
						_, pres := testCase.expectPutURLs[reqURL]
						require.True(t, pres, "Unexpected PUT request %s", reqURL)
						delete(testCase.expectPutURLs, reqURL)

						// Check spec/replicas modified correctly
						var scaleOutput map[string]interface{}
						err = json.Unmarshal(buf.Bytes(), &scaleOutput)
						require.NoError(t, err, "Request body must be JSON object")
						spec, ok := scaleOutput["spec"].(map[string]interface{})
						require.True(t, ok, "Spec property is missing or invalid")
						newReplicas, ok := spec["replicas"].(float64)
						require.True(t, ok, "Replicas property is missing or invalid")
						require.Equal(t, testCase.newReplicas, int(newReplicas), "Wrong modified number of replicas")

						// Replace body
						actual.Body = ioutil.NopCloser(&buf)
					}
					return true
				}
				return false
			})
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			old, err := kc.ScaleDeployment(testCase.spaceName, testCase.appName, testCase.envName, testCase.newReplicas)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
				require.NotNil(t, old, "Previous replicas are nil")
				require.Equal(t, testCase.oldReplicas, *old, "Wrong number of previous replicas")
			}

			// Check we saw all expected PUT requests
			require.Empty(t, testCase.expectPutURLs, "Not all PUT requests sent: %v", testCase.expectPutURLs)
		})
	}
}

func TestDeleteDeployment(t *testing.T) {
	// DeleteOptions do not change
	policy := metav1.DeletePropagationForeground
	expectOpts := metav1.DeleteOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
		PropagationPolicy: &policy,
	}

	testCases := []struct {
		testName         string
		spaceName        string
		appName          string
		envName          string
		cassetteName     string
		expectDeleteURLs map[string]struct{}
		shouldFail       bool
	}{
		{
			testName:     "Basic",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myDeploy": {},
				"http://api.myCluster/oapi/v1/namespaces/my-run/routes/myDeploy":            {},
				"http://api.myCluster/api/v1/namespaces/my-run/services/myDeploy":           {},
			},
		},
		{
			testName:         "Bad Environment",
			spaceName:        "mySpace",
			appName:          "myApp",
			envName:          "doesNotExist",
			cassetteName:     "deletedeployment",
			expectDeleteURLs: map[string]struct{}{},
			shouldFail:       true,
		},
		{
			testName:     "Wrong Space",
			spaceName:    "otherSpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-wrongspace",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/routes/myDeploy":  {},
				"http://api.myCluster/api/v1/namespaces/my-run/services/myDeploy": {},
			},
			shouldFail: true,
		},
		{
			testName:     "No Routes",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-noroutes",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myDeploy": {},
				"http://api.myCluster/api/v1/namespaces/my-run/services/myDeploy":           {},
			},
		},
		{
			testName:     "No Services",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-noservices",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myDeploy": {},
				"http://api.myCluster/oapi/v1/namespaces/my-run/routes/myDeploy":            {},
			},
		},
		{
			testName:     "No DeploymentConfig",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-nodc",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/routes/myDeploy":  {},
				"http://api.myCluster/api/v1/namespaces/my-run/services/myDeploy": {},
			},
			shouldFail: true,
		},
		{
			// Tests failure to map application name to OpenShift resources
			// falls back to app name
			testName:     "No Builds",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-nobuilds",
			expectDeleteURLs: map[string]struct{}{
				"http://api.myCluster/oapi/v1/namespaces/my-run/routes/myApp":            {},
				"http://api.myCluster/api/v1/namespaces/my-run/services/myApp":           {},
				"http://api.myCluster/oapi/v1/namespaces/my-run/deploymentconfigs/myApp": {},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			r.SetMatcher(func(actual *http.Request, expected cassette.Request) bool {
				if cassette.DefaultMatcher(actual, expected) {
					// Check request body when sending DELETE
					if actual.Method == "DELETE" {
						var buf bytes.Buffer
						reqBody := actual.Body
						_, err := buf.ReadFrom(reqBody)
						require.NoError(t, err, "Error reading request body")
						defer reqBody.Close()

						// Mark interaction as seen
						reqURL := actual.URL.String()
						_, pres := testCase.expectDeleteURLs[reqURL]
						require.True(t, pres, "Unexpected DELETE request %s", reqURL)
						delete(testCase.expectDeleteURLs, reqURL)

						// Check delete options are correct
						var deleteOutput metav1.DeleteOptions
						err = json.Unmarshal(buf.Bytes(), &deleteOutput)
						require.NoError(t, err, "Request body must be DeleteOptions")
						require.Equal(t, expectOpts, deleteOutput, "DeleteOptions do not match")

						// Replace body
						actual.Body = ioutil.NopCloser(&buf)
					}
					return true
				}
				return false
			})
			defer r.Stop()

			fixture := &testFixture{}
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			err = kc.DeleteDeployment(testCase.spaceName, testCase.appName, testCase.envName)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")
			}

			// Check we saw all expected DELETE requests
			require.Empty(t, testCase.expectDeleteURLs, "Not all DELETE requests sent: %v", testCase.expectDeleteURLs)
		})
	}
}

func TestGetDeploymentStats(t *testing.T) {
	testCases := []*deployStatsTestData{
		defaultDeployStatsTestData,
		{
			testName:     "Bad Environment",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "doesNotExist",
			cassetteName: "getdeployment",
			metricsInput: defaultMetricsInput,
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			fixture.metricsInput = testCase.metricsInput
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			stats, err := kc.GetDeploymentStats(testCase.spaceName, testCase.appName, testCase.envName, testCase.startTime)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")

				require.NotNil(t, stats, "GetDeploymentStats returned nil")
				result := fixture.metrics
				require.NotNil(t, result, "Metrics API not called")

				// Check each metric type
				require.NotNil(t, stats.Attributes, "Stat attributes are nil")
				verifyNumberTuple(testCase.metricsInput.cpu[0], stats.Attributes.Cores, t, "CPU")
				verifyMetricsParams(testCase, result.cpuParams, t, "CPU metrics")
				verifyNumberTuple(testCase.metricsInput.memory[0], stats.Attributes.Memory, t, "memory")
				verifyMetricsParams(testCase, result.memParams, t, "Memory metrics")
				verifyNumberTuple(testCase.metricsInput.netTx[0], stats.Attributes.NetTx, t, "network sent")
				verifyMetricsParams(testCase, result.netTxParams, t, "Network sent metrics")
				verifyNumberTuple(testCase.metricsInput.netRx[0], stats.Attributes.NetRx, t, "network received")
				verifyMetricsParams(testCase, result.netRxParams, t, "Network received metrics")
			}
		})
	}
}

func verifyNumberTuple(expected *app.TimedNumberTuple, actual *app.TimedNumberTuple, t *testing.T, metricName string) {
	require.NotNil(t, actual, "%s metric is nil", metricName)
	require.InDelta(t, *expected.Time, *actual.Time, fltDelta, "Incorrect %s timestamp", metricName)
	require.InDelta(t, *expected.Value, *actual.Value, fltDelta, "Incorrect %s value", metricName)
}

func TestGetDeploymentStatSeries(t *testing.T) {
	testCases := []*deployStatsTestData{
		defaultDeployStatsTestData,
		{
			testName:     "Bad Environment",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "doesNotExist",
			metricsInput: defaultMetricsInput,
			cassetteName: "getdeployment",
			shouldFail:   true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			fixture.metricsInput = testCase.metricsInput
			kc := getDefaultKubeClient(fixture, r.Transport, t)

			stats, err := kc.GetDeploymentStatSeries(testCase.spaceName, testCase.appName, testCase.envName,
				testCase.startTime, testCase.endTime, testCase.limit)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
			} else {
				require.NoError(t, err, "Unexpected error occurred")

				require.NotNil(t, stats, "GetDeploymentStats returned nil")
				result := fixture.metrics
				require.NotNil(t, result, "Metrics API not called")

				// Check each metric type
				verifyNumberTuples(testCase.metricsInput.cpu, stats.Cores, t, "CPU")
				verifyMetricsParams(testCase, result.cpuParams, t, "CPU metrics")
				verifyNumberTuples(testCase.metricsInput.memory, stats.Memory, t, "memory")
				verifyMetricsParams(testCase, result.memParams, t, "Memory metrics")
				verifyNumberTuples(testCase.metricsInput.netTx, stats.NetTx, t, "network sent")
				verifyMetricsParams(testCase, result.netTxParams, t, "Network sent metrics")
				verifyNumberTuples(testCase.metricsInput.netRx, stats.NetRx, t, "network received")
				verifyMetricRangeParams(testCase, result.netRxParams, t, "Network received metrics")

				// Check time range
				require.Equal(t, testCase.expectStart, int64(*stats.Start), "Incorrect start time")
				require.Equal(t, testCase.expectEnd, int64(*stats.End), "Incorrect end time")
			}
		})
	}
}

func verifyNumberTuples(expected []*app.TimedNumberTuple, actual []*app.TimedNumberTuple, t *testing.T, metricName string) {
	require.Equal(t, len(expected), len(actual), "Wrong number of %s metrics", metricName)
	for idx := range expected {
		verifyNumberTuple(expected[idx], actual[idx], t, metricName)
	}
}

func verifyMetricsParams(testCase *deployStatsTestData, params *metricsHolder, t *testing.T,
	metricName string) {
	require.Equal(t, testCase.envNS, params.namespace, metricName+" called with wrong namespace")
	require.Equal(t, testCase.startTime, params.startTime, metricName+" called with wrong start time")

	// Check each method called with expected pods
	podUIDs := make([]string, len(params.pods))
	for idx, pod := range params.pods {
		podUIDs[idx] = string(pod.UID)
	}
	require.ElementsMatch(t, testCase.expectPodUIDs, podUIDs, metricName+" called with unexpected pods")
}

func verifyMetricRangeParams(testCase *deployStatsTestData, params *metricsHolder, t *testing.T,
	metricName string) {
	verifyMetricsParams(testCase, params, t, metricName)
	require.Equal(t, testCase.endTime, params.endTime, metricName+" called with wrong end time")
	require.Equal(t, testCase.limit, params.limit, metricName+" called with wrong limit")
}

func verifyApplication(app *app.SimpleApp, testCase *appTestData, t *testing.T) {
	require.NotNil(t, app, "Application is nil")
	require.NotNil(t, app.Attributes, "Application attributes are nil")
	require.Equal(t, testCase.appName, app.Attributes.Name, "Incorrect application name")
	require.NotNil(t, app.Attributes.Deployments, "Deployments are nil")
	require.Equal(t, len(testCase.deployTestData), len(app.Attributes.Deployments), "Wrong number of deployments")
	for _, dep := range app.Attributes.Deployments {
		var depInput *deployTestData
		if dep != nil {
			depInput = testCase.deployTestData[dep.Attributes.Name]
			require.NotNil(t, depInput, "Unknown env: "+dep.Attributes.Name)
		}
		verifyDeployment(dep, depInput, t)
	}
}

func verifyDeployment(dep *app.SimpleDeployment, testCase *deployTestData, t *testing.T) {
	require.NotNil(t, dep, "Deployment is nil")
	require.NotNil(t, dep.Attributes, "Deployment attributes are nil")
	require.Equal(t, testCase.envName, dep.Attributes.Name, "Incorrect deployment name")
	require.NotNil(t, dep.Attributes.Version, "Deployments version is nil")
	require.Equal(t, testCase.expectVersion, *dep.Attributes.Version, "Incorrect deployment version")

	// Check pod status and total
	require.NotNil(t, dep.Attributes.Pods, "Pods are nil")
	require.ElementsMatch(t, testCase.expectPodStatus, dep.Attributes.Pods, "Incorrect pod status %v", dep.Attributes.Pods)
	require.NotNil(t, dep.Attributes.PodTotal, "Pod total is nil")
	require.Equal(t, testCase.expectPodsTotal, *dep.Attributes.PodTotal, "Wrong number of total pods")

	// Check pod quota and total
	require.NotNil(t, dep.Attributes.PodsQuota, "PodsQuota is nil")
	require.NotNil(t, dep.Attributes.PodsQuota.Cpucores, "PodsQuota.Cpucores is nil")
	require.NotNil(t, dep.Attributes.PodsQuota.Memory, "PodsQuota.Memory is nil")

	require.Equal(t, testCase.expectPodsQuotaCpucores, *dep.Attributes.PodsQuota.Cpucores, "Incorrect pods quota cpucores")
	require.Equal(t, testCase.expectPodsQuotaMemory, *dep.Attributes.PodsQuota.Memory, "Incorrect pods quota cpucores")

	// Check related URLs
	require.NotNil(t, dep.Links, "Related URLs are nil")
	require.NotNil(t, dep.Links.Console, "Console URL is nil")
	require.Equal(t, testCase.expectConsoleURL, *dep.Links.Console, "Console URL is incorrect")
	if len(testCase.expectAppURL) == 0 {
		require.Nil(t, dep.Links.Application, "Application URL is not nil")
	} else {
		require.NotNil(t, dep.Links.Application, "Application URL is nil")
		require.Equal(t, testCase.expectAppURL, *dep.Links.Application, "Application URL is incorrect")
	}
	if len(testCase.expectLogURL) == 0 {
		require.Nil(t, dep.Links.Application, "Logs URL is not nil")
	} else {
		require.NotNil(t, dep.Links.Logs, "Logs URL is nil")
		require.Equal(t, testCase.expectLogURL, *dep.Links.Logs, "Logs URL is incorrect")
	}
}

// code for test URL provider

func (up *testURLProvider) GetAPIToken() (*string, error) {
	return &up.apiToken, nil
}

func (up *testURLProvider) GetMetricsToken(envNS string) (*string, error) {
	return &up.clusterToken, nil
}

func (up *testURLProvider) GetAPIURL() (*string, error) {
	return &up.apiURL, nil
}

func (up *testURLProvider) GetConsoleURL(envNS string) (*string, error) {
	path := fmt.Sprintf("console/project/%s", envNS)
	// Replace "api" prefix with "console" and append path
	consoleURL, err := modifyURL(up.clusterURL, "console", path)
	if err != nil {
		return nil, err
	}
	consoleURLStr := consoleURL.String()
	return &consoleURLStr, nil
}

func (up *testURLProvider) GetLoggingURL(envNS string, deployName string) (*string, error) {
	consoleURL, err := up.GetConsoleURL(envNS)
	if err != nil {
		return nil, err
	}
	logURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", *consoleURL, deployName)
	return &logURL, nil
}

func (up *testURLProvider) GetMetricsURL(envNS string) (*string, error) {
	metricsURL, err := modifyURL(up.clusterURL, "metrics", "")
	if err != nil {
		return nil, err
	}
	mu := metricsURL.String()
	return &mu, nil
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
