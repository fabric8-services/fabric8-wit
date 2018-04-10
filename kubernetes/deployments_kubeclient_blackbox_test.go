package kubernetes_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"

	"github.com/dnaeon/go-vcr/recorder"

	"github.com/stretchr/testify/require"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	errs "github.com/pkg/errors"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
)

// Used for equality comparisons between float64s
const fltDelta = 0.00000001

// Path to JSON resources
const pathToTestJSON = "../test/kubernetes/"

type testKube struct {
	kubernetes.KubeRESTAPI // Allows us to only implement methods we'll use
	fixture                *testFixture
	configMapHolder        *testConfigMap
	quotaHolder            *testResourceQuota
	rcHolder               *testReplicationController
	podHolder              *testPod
	svcHolder              *testService
	svcDelHolder           []*testDeleteByName
}

type testFixture struct {
	cmInput      *configMapInput
	rqInput      *resourceQuotaInput
	bcInput      string                // BC json file
	scaleInput   deploymentConfigInput // app name -> namespace -> DC scale json file
	metricsInput *metricsInput
	kube         *testKube
	os           *testOpenShift
	metrics      *testMetrics
	deploymentInput
}

// Collects input data necessary to retrieve a deployment
type deploymentInput struct {
	dcInput    deploymentConfigInput // app name -> namespace -> DC json file
	rcInput    map[string]string     // namespace -> RC JSON file
	podInput   map[string]string     // namespace -> pod JSON file
	svcInput   map[string]string     // namespace -> service JSON file
	routeInput map[string]string     // namespace -> route JSON file
}

var defaultDeploymentInput = deploymentInput{
	dcInput:    defaultDeploymentConfigInput,
	rcInput:    defaultReplicationControllerInput,
	podInput:   defaultPodInput,
	svcInput:   defaultServiceInput,
	routeInput: defaultRouteInput,
}

func getDefaultKubeClient(fixture *testFixture, transport http.RoundTripper, t *testing.T) kubernetes.KubeClientInterface {
	config := &kubernetes.KubeClientConfig{
		ClusterURL:    "http://api.myCluster",
		BearerToken:   "myToken",
		UserNamespace: "myNamespace",
		Transport:     transport,
		MetricsGetter: fixture,
	}

	kc, err := kubernetes.NewKubeClient(config)
	require.NoError(t, err)
	return kc
}

// Config Maps fakes

type configMapInput struct {
	data   map[string]string
	labels map[string]string
}

var defaultConfigMapInput *configMapInput = &configMapInput{
	labels: map[string]string{"provider": "fabric8"},
	data: map[string]string{
		"run":   "name: Run\nnamespace: my-run\norder: 1",
		"stage": "name: Stage\nnamespace: my-stage\norder: 0",
	},
}

type testConfigMap struct {
	corev1.ConfigMapInterface
	input     *configMapInput
	namespace string
	configMap *v1.ConfigMap
}

func (tk *testKube) ConfigMaps(ns string) corev1.ConfigMapInterface {
	input := tk.fixture.cmInput
	if input == nil {
		input = defaultConfigMapInput
	}
	result := &testConfigMap{
		input:     input,
		namespace: ns,
	}
	tk.configMapHolder = result
	return result
}

func (cm *testConfigMap) Get(name string, options metav1.GetOptions) (*v1.ConfigMap, error) {
	result := &v1.ConfigMap{
		Data: cm.input.data,
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: cm.input.labels,
		},
	}
	cm.configMap = result
	return result, nil
}

// Resource Quota fakes

type resourceQuotaInput struct {
	name       string
	namespace  string
	hard       map[v1.ResourceName]float64
	used       map[v1.ResourceName]float64
	shouldFail bool
}

var defaultResourceQuotaInput *resourceQuotaInput = &resourceQuotaInput{
	name:      "run",
	namespace: "my-run",
	hard: map[v1.ResourceName]float64{
		v1.ResourceLimitsCPU:    0.7,
		v1.ResourceLimitsMemory: 1024,
	},
	used: map[v1.ResourceName]float64{
		v1.ResourceLimitsCPU:    0.4,
		v1.ResourceLimitsMemory: 512,
	},
}

type testResourceQuota struct {
	corev1.ResourceQuotaInterface
	input     *resourceQuotaInput
	namespace string
	quota     *v1.ResourceQuota
}

func (tk *testKube) ResourceQuotas(ns string) corev1.ResourceQuotaInterface {
	input := tk.fixture.rqInput
	if input == nil {
		input = defaultResourceQuotaInput
	}
	result := &testResourceQuota{
		input:     input,
		namespace: ns,
	}
	tk.quotaHolder = result
	return result
}

func (rq *testResourceQuota) Get(name string, options metav1.GetOptions) (*v1.ResourceQuota, error) {
	if rq.input.hard == nil || rq.input.used == nil { // Used to indicate no quota object
		return nil, nil
	}
	hardQuantity, err := stringToQuantityMap(rq.input.hard)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	usedQuantity, err := stringToQuantityMap(rq.input.used)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	result := &v1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1.ResourceQuotaStatus{
			Hard: hardQuantity,
			Used: usedQuantity,
		},
	}
	rq.quota = result
	return result, nil
}

func stringToQuantityMap(input map[v1.ResourceName]float64) (v1.ResourceList, error) {
	result := make(map[v1.ResourceName]resource.Quantity)
	for k, v := range input {
		strVal := strconv.FormatFloat(v, 'f', -1, 64)
		q, err := resource.ParseQuantity(strVal)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		result[k] = q
	}
	return result, nil
}

// Replication Controller fakes

var defaultReplicationControllerInput = map[string]string{
	"my-run": "replicationcontroller.json",
}

type testReplicationController struct {
	corev1.ReplicationControllerInterface
	inputFile string
	namespace string
}

func (tk *testKube) ReplicationControllers(ns string) corev1.ReplicationControllerInterface {
	input := tk.fixture.rcInput[ns]
	result := &testReplicationController{
		inputFile: input,
		namespace: ns,
	}
	tk.rcHolder = result
	return result
}

func (rc *testReplicationController) List(options metav1.ListOptions) (*v1.ReplicationControllerList, error) {
	var result v1.ReplicationControllerList
	if len(rc.inputFile) == 0 {
		// No matching RC
		return &result, nil
	}
	err := readJSON(rc.inputFile, &result)
	return &result, err
}

// Pod fakes

var defaultPodInput = map[string]string{
	"my-run": "pods.json",
}

type testPod struct {
	corev1.PodInterface
	inputFile string
	namespace string
}

func (tk *testKube) Pods(ns string) corev1.PodInterface {
	input := tk.fixture.podInput[ns]
	result := &testPod{
		inputFile: input,
		namespace: ns,
	}
	tk.podHolder = result
	return result
}

func (pod *testPod) List(options metav1.ListOptions) (*v1.PodList, error) {
	var result v1.PodList
	if len(pod.inputFile) == 0 {
		// No matching pods
		return &result, nil
	}
	err := readJSON(pod.inputFile, &result)
	return &result, err
}

func (fixture *testFixture) GetKubeRESTAPI(config *kubernetes.KubeClientConfig) (kubernetes.KubeRESTAPI, error) {
	mock := &testKube{
		fixture: fixture,
	}
	fixture.kube = mock
	return mock, nil
}

// Service fakes

var defaultServiceInput = map[string]string{
	"my-run": "services-two.json",
}

type testService struct {
	corev1.ServiceInterface
	inputFile string
	namespace string
	kube      *testKube
}

func (tk *testKube) Services(ns string) corev1.ServiceInterface {
	input := tk.fixture.svcInput[ns]
	result := &testService{
		inputFile: input,
		namespace: ns,
		kube:      tk,
	}
	tk.svcHolder = result
	return result
}

func (svc *testService) List(options metav1.ListOptions) (*v1.ServiceList, error) {
	var result v1.ServiceList
	if len(svc.inputFile) == 0 {
		// No matching service
		return &result, nil
	}
	err := readJSON(svc.inputFile, &result)
	return &result, err
}

func (svc *testService) Delete(name string, options *metav1.DeleteOptions) error {
	delHolder := &testDeleteByName{
		namespace: svc.namespace,
		name:      name,
		opts:      options,
	}
	svc.kube.svcDelHolder = append(svc.kube.svcDelHolder, delHolder)
	return nil
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

// OpenShift API fakes

type testOpenShift struct {
	fixture        *testFixture
	scaleHolder    *testScale
	routeHolder    *testGetResult
	delDCHolder    *testDeleteByName
	delRouteHolder []*testDeleteByName
}

type testScale struct {
	scaleOutput map[string]interface{}
	namespace   string
	dcName      string
}

type testGetResult struct {
	namespace     string
	labelSelector string
}

type testDeleteByName struct {
	namespace string
	name      string
	opts      *metav1.DeleteOptions
}

func (fixture *testFixture) GetOpenShiftRESTAPI(config *kubernetes.KubeClientConfig) (kubernetes.OpenShiftRESTAPI, error) {
	oapi := &testOpenShift{
		fixture: fixture,
	}
	fixture.os = oapi
	return oapi, nil
}

func (to *testOpenShift) GetBuildConfigs(namespace string, labelSelector string) (map[string]interface{}, error) {
	var result map[string]interface{}
	input := to.fixture.bcInput
	if len(input) == 0 {
		// No matching BCs
		return result, nil
	}
	err := readJSON(input, &result)
	return result, err
}

type deploymentConfigInput map[string]map[string]string

var defaultDeploymentConfigInput = deploymentConfigInput{
	"myApp": {
		"my-run": "deploymentconfig-one.json",
	},
}

func (input deploymentConfigInput) getInput(appName string, envNS string) *string {
	inputForApp, pres := input[appName]
	if !pres {
		return nil
	}
	inputForEnv, pres := inputForApp[envNS]
	if !pres {
		return nil
	}
	return &inputForEnv
}

func (to *testOpenShift) GetDeploymentConfig(namespace string, name string) (map[string]interface{}, error) {
	input := to.fixture.dcInput.getInput(name, namespace)
	if input == nil {
		// No matching DC
		return nil, nil
	}
	var result map[string]interface{}
	err := readJSON(*input, &result)
	return result, err
}

func (to *testOpenShift) DeleteDeploymentConfig(namespace string, name string, opts *metav1.DeleteOptions) error {
	to.delDCHolder = &testDeleteByName{
		namespace: namespace,
		name:      name,
		opts:      opts,
	}
	return nil
}

func (to *testOpenShift) GetDeploymentConfigScale(namespace string, name string) (map[string]interface{}, error) {
	input := to.fixture.scaleInput.getInput(name, namespace)
	if input == nil {
		// No matching DC scale
		return nil, nil
	}
	var result map[string]interface{}
	err := readJSON(*input, &result)
	return result, err
}

func (to *testOpenShift) SetDeploymentConfigScale(namespace string, name string, scale map[string]interface{}) error {
	to.scaleHolder = &testScale{
		namespace:   namespace,
		dcName:      name,
		scaleOutput: scale,
	}
	return nil
}

var defaultRouteInput = map[string]string{
	"my-run": "routes-two.json",
}

func (to *testOpenShift) GetRoutes(namespace string, labelSelector string) (map[string]interface{}, error) {
	var result map[string]interface{}
	input := to.fixture.routeInput[namespace]
	if len(input) == 0 {
		// No matching routes
		return result, nil
	}
	to.routeHolder = &testGetResult{
		namespace:     namespace,
		labelSelector: labelSelector,
	}
	err := readJSON(input, &result)
	return result, err
}

func (to *testOpenShift) DeleteRoute(namespace string, name string, opts *metav1.DeleteOptions) error {
	delHolder := &testDeleteByName{
		namespace: namespace,
		name:      name,
		opts:      opts,
	}
	to.delRouteHolder = append(to.delRouteHolder, delHolder)
	return nil
}

func readJSON(filename string, dest interface{}) error {
	path := pathToTestJSON + filename
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return errs.WithStack(err)
	}
	err = json.Unmarshal(jsonBytes, dest)
	if err != nil {
		return errs.WithStack(err)
	}
	return nil
}

func TestGetMetrics(t *testing.T) {
	fixture := &testFixture{}

	token := "myToken"
	testCases := []struct {
		name          string
		clusterURL    string
		expectedURL   string
		shouldSucceed bool
	}{
		{"Basic", "https://api.myCluster.url:443/cluster", "https://metrics.myCluster.url", true},
		{"Bad URL", "https://myCluster.url:443/cluster", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := &kubernetes.KubeClientConfig{
				ClusterURL:        testCase.clusterURL,
				BearerToken:       token,
				UserNamespace:     "myNamespace",
				KubeRESTAPIGetter: fixture,
				MetricsGetter:     fixture,
			}
			_, err := kubernetes.NewKubeClient(config)
			if testCase.shouldSucceed {
				require.NoError(t, err, "Unexpected error")

				metricsConfig := fixture.metrics.config
				require.NotNil(t, metricsConfig, "Metrics config is nil")
				require.Equal(t, testCase.expectedURL, metricsConfig.MetricsURL, "Incorrect Metrics URL")
				require.Equal(t, token, metricsConfig.BearerToken, "Incorrect bearer token")
			} else {
				require.Error(t, err, "Expected error, but was successful")
			}
		})
	}
}

func TestClose(t *testing.T) {
	fixture := &testFixture{}
	kc := getDefaultKubeClient(fixture, nil, t)

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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName)
			require.NoError(t, err, "Failed to open cassette")
			defer r.Stop()

			fixture := &testFixture{}
			config := &kubernetes.KubeClientConfig{
				ClusterURL:    "http://api.myCluster",
				BearerToken:   "myToken",
				UserNamespace: "myNamespace",
				Transport:     r.Transport,
				MetricsGetter: fixture,
			}

			kc, err := kubernetes.NewKubeClient(config)
			if testCase.shouldFail {
				require.Error(t, err, "Expected an error")
				require.True(t, fixture.metrics.closed, "Metrics must be closed after error")
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
	bcJson       string
	appTestData  map[string]*appTestData // Keys are app names
	cassetteName string
}

var defaultSpaceTestData = &spaceTestData{
	testName:     "Basic",
	spaceName:    "mySpace",
	bcJson:       "buildconfigs-one.json",
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
	envNS                   string
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
	envNS:         "my-run",
	expectVersion: "1.0.2",
	expectPodStatus: [][]string{
		{"Running", "2"},
	},
	expectPodsQuotaCpucores: 0.976,
	expectPodsQuotaMemory:   524288000,
	expectPodsTotal:         2,
	expectConsoleURL:        "http://console.myCluster/console/project/my-run",
	expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myApp-1?tab=logs",
	expectAppURL:            "http://myApp-my-run.example.com",
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
			bcJson:       "buildconfigs-emptylist.json",
			cassetteName: "getspace-empty-bc",
		},
		{
			testName:     "Wrong List",
			spaceName:    "mySpace",
			bcJson:       "buildconfigs-wronglist.json",
			cassetteName: "getspace-wrong-list",
			shouldFail:   true,
		},
		{
			testName:     "No Items",
			spaceName:    "mySpace",
			bcJson:       "buildconfigs-noitems.json",
			cassetteName: "getspace-no-items",
			shouldFail:   true,
		},
		{
			testName:     "Not Object",
			spaceName:    "mySpace",
			bcJson:       "buildconfigs-notobject.json",
			cassetteName: "getspace-not-object",
			shouldFail:   true,
		},
		{
			testName:     "No Metadata",
			spaceName:    "mySpace",
			bcJson:       "buildconfigs-nometadata.json",
			cassetteName: "getspace-no-metadata",
			shouldFail:   true,
		},
		{
			testName:     "No Name",
			spaceName:    "mySpace",
			bcJson:       "buildconfigs-noname.json",
			cassetteName: "getspace-no-name",
			shouldFail:   true,
		},
		{
			testName:     "Two Apps One Deployed",
			spaceName:    "mySpace", // Test two BCs, but only one DC
			bcJson:       "buildconfigs-two.json",
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
			bcJson:       "buildconfigs-two.json",
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
							envNS:         "my-run",
							expectVersion: "1.0.2",
							expectPodStatus: [][]string{
								{"Running", "2"},
							},
							expectPodsTotal:         2,
							expectPodsQuotaCpucores: 0.976,
							expectPodsQuotaMemory:   524288000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-run",
							expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myApp-1?tab=logs",
							expectAppURL:            "http://myApp-my-run.example.com",
						},
						"stage": {
							spaceName:     "mySpace",
							appName:       "myApp",
							envName:       "stage",
							envNS:         "my-stage",
							expectVersion: "1.0.3",
							expectPodStatus: [][]string{
								{"Running", "1"},
								{"Terminating", "1"},
							},
							expectPodsTotal:         2,
							expectPodsQuotaCpucores: 0.976,
							expectPodsQuotaMemory:   524288000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-stage",
							expectLogURL:            "http://console.myCluster/console/project/my-stage/browse/rc/myApp-1?tab=logs",
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
							envNS:         "my-run",
							expectVersion: "1.0.1",
							expectPodStatus: [][]string{
								{"Running", "1"},
							},
							expectPodsTotal:         1,
							expectPodsQuotaCpucores: 0.488,
							expectPodsQuotaMemory:   262144000,
							expectConsoleURL:        "http://console.myCluster/console/project/my-run",
							expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myOtherApp-1?tab=logs",
							expectAppURL:            "http://myOtherApp-my-run.example.com",
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
					envNS:         "my-stage",
					expectVersion: "1.0.3",
					expectPodStatus: [][]string{
						{"Running", "1"},
						{"Terminating", "1"},
					},
					expectPodsTotal:         2,
					expectPodsQuotaCpucores: 0.976,
					expectPodsQuotaMemory:   524288000,
					expectConsoleURL:        "http://console.myCluster/console/project/my-stage",
					expectLogURL:            "http://console.myCluster/console/project/my-stage/browse/rc/myApp-1?tab=logs",
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
					envNS:            "my-run",
					expectVersion:    "1.0.1",
					expectConsoleURL: "http://console.myCluster/console/project/my-run",
					expectLogURL:     "http://console.myCluster/console/project/my-run/browse/rc/myOtherApp-1?tab=logs",
					expectAppURL:     "http://myOtherApp-my-run.example.com",
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
			envNS:            "my-run",
			expectVersion:    "1.0.2",
			expectPodStatus:  [][]string{},
			expectPodsTotal:  0,
			expectConsoleURL: "http://console.myCluster/console/project/my-run",
			expectLogURL:     "http://console.myCluster/console/project/my-run/browse/rc/myApp-2?tab=logs",
			expectAppURL:     "http://myApp-my-run.example.com",
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
			envNS:         "my-run",
			expectVersion: "1.0.2",
			expectPodStatus: [][]string{
				{"Running", "2"},
			},
			expectPodsTotal:         2,
			expectPodsQuotaCpucores: 0.976,
			expectPodsQuotaMemory:   524288000,
			expectConsoleURL:        "http://console.myCluster/console/project/my-run",
			expectLogURL:            "http://console.myCluster/console/project/my-run/browse/rc/myApp-1?tab=logs",
			expectAppURL:            "http://myApp-my-run.example.com",
			cassetteName:            "getdeployment-nospace",
		},
		{
			// Tests that we don't accept deployment configs with a space
			// label different from the argument passed to GetDeployment
			testName:     "Wrong Space Label",
			spaceName:    "myWrongSpace",
			appName:      "myApp",
			envName:      "run",
			envNS:        "my-run",
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
		testName     string
		spaceName    string
		appName      string
		envName      string
		expectedNS   string
		cassetteName string
		newReplicas  int
		oldReplicas  int
		shouldFail   bool
	}{
		{
			testName:     "Basic",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			expectedNS:   "my-run",
			cassetteName: "scaledeployment",
			newReplicas:  3,
			oldReplicas:  2,
		},
		{
			testName:     "Zero Replicas",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			expectedNS:   "my-run",
			cassetteName: "scaledeployment-zero",
			newReplicas:  1,
			oldReplicas:  0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			r, err := recorder.New(pathToTestJSON + testCase.cassetteName) // TODO check all interactions occurred?
			r.SetMatcher(func(actual *http.Request, expected cassette.Request) bool {
				if cassette.DefaultMatcher(actual, expected) {
					// Check scale request body when sending PUT
					if actual.Method == "PUT" {
						var buf bytes.Buffer
						reqBody := actual.Body
						_, err := buf.ReadFrom(reqBody)
						require.NoError(t, err, "Error reading request body")
						defer reqBody.Close()

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
		testName     string
		spaceName    string
		appName      string
		envName      string
		cassetteName string
		expectNS     string
		shouldFail   bool
		deploymentInput
	}{
		{
			testName:     "Basic",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment",
			expectNS:     "my-run",
		},
		{
			testName:     "Bad Environment",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "doesNotExist",
			cassetteName: "deletedeployment",
			expectNS:     "my-run",
			shouldFail:   true,
		},
		{
			testName:     "Wrong Space",
			spaceName:    "otherSpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment",
			expectNS:     "my-run",
			shouldFail:   true,
		},
		{
			testName:     "No Routes",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-noroutes",
			expectNS:     "my-run",
		},
		{
			testName:     "No Services",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-noservices",
			expectNS:     "my-run",
		},
		{
			testName:     "No DeploymentConfig",
			spaceName:    "mySpace",
			appName:      "myApp",
			envName:      "run",
			cassetteName: "deletedeployment-nodc",
			expectNS:     "my-run",
			shouldFail:   true,
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
		})
	}
}

func createSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	var empty struct{}
	for _, value := range values {
		result[value] = empty
	}
	return result
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
