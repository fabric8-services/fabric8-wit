package kubernetes_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	hawkular "github.com/hawkular/hawkular-client-go/metrics"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/client-go/pkg/api/v1"
)

type testHawkular struct {
	getter *testHawkularGetter
	output *testMetricsOutput
}

type testHawkularGetter struct {
	result *testHawkular
	input  *testMetricsInput
}

type testMetricsInput struct {
	buckets   []*hawkular.Bucketpoint
	pods      []*v1.Pod
	namespace string
	startTime time.Time
	endTime   time.Time
	limit     int
}

type testMetricsOutput struct {
	metricType hawkular.MetricType
	namespace  string
	filters    url.Values
	closed     bool
}

func (getter *testHawkularGetter) GetHawkularRESTAPI(config *kubernetes.MetricsClientConfig) (kubernetes.HawkularRESTAPI, error) {
	helper := &testHawkular{
		getter: getter,
		output: &testMetricsOutput{},
	}
	getter.result = helper
	return helper, nil
}

func (helper *testHawkular) ReadBuckets(metricType hawkular.MetricType, namespace string,
	modifiers ...hawkular.Modifier) ([]*hawkular.Bucketpoint, error) {
	// Run modifiers on a dummy request to determine which filters were applied
	req := &http.Request{
		URL: &url.URL{},
	}
	for _, modifier := range modifiers {
		err := modifier(req)
		if err != nil {
			return nil, err
		}
	}
	helper.output.metricType = metricType
	helper.output.namespace = namespace
	helper.output.filters = req.URL.Query()

	buckets := helper.getter.input.buckets
	return buckets, nil
}

func (helper *testHawkular) Close() {
	helper.output.closed = true
}

var singleMetricTestCases []*testMetricsInput = []*testMetricsInput{
	{ // Basic test case
		buckets: []*hawkular.Bucketpoint{
			{
				Avg:   5.0,
				Start: hawkular.FromUnixMilli(1516301818000),
				End:   hawkular.FromUnixMilli(1516301878000),
			},
		},
		pods: []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("hello"),
				},
			},
		},
		namespace: "test",
		startTime: hawkular.FromUnixMilli(1516301818000),
	},
	{ // Multiple pods
		buckets: []*hawkular.Bucketpoint{
			{
				Avg:   5.0,
				Start: hawkular.FromUnixMilli(1516301818000),
				End:   hawkular.FromUnixMilli(1516301878000),
			},
		},
		pods: []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("hello"),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("world"),
				},
			},
		},
		namespace: "test",
		startTime: hawkular.FromUnixMilli(1516301818000),
	},
}

var metricRangeTestCases []*testMetricsInput = []*testMetricsInput{
	{ // Basic test case
		buckets: []*hawkular.Bucketpoint{
			{
				Avg:   5.0,
				Start: hawkular.FromUnixMilli(1516301758000),
				End:   hawkular.FromUnixMilli(1516301818000),
			},
			{
				Avg:   72.0,
				Start: hawkular.FromUnixMilli(1516301818000),
				End:   hawkular.FromUnixMilli(1516301878000),
			},
		},
		pods: []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("hello"),
				},
			},
		},
		namespace: "test",
		startTime: hawkular.FromUnixMilli(1516301758000),
		endTime:   hawkular.FromUnixMilli(1516301878000),
	},
	{ // Multiple pods
		buckets: []*hawkular.Bucketpoint{
			{
				Avg:   5.0,
				Start: hawkular.FromUnixMilli(1516301758000),
				End:   hawkular.FromUnixMilli(1516301818000),
			},
			{
				Avg:   72.0,
				Start: hawkular.FromUnixMilli(1516301818000),
				End:   hawkular.FromUnixMilli(1516301878000),
			},
		},
		pods: []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("hello"),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("world"),
				},
			},
		},
		namespace: "test",
		startTime: hawkular.FromUnixMilli(1516301758000),
		endTime:   hawkular.FromUnixMilli(1516301878000),
	},
	{ // Metrics limit
		buckets: []*hawkular.Bucketpoint{
			{
				Avg:   5.0,
				Start: hawkular.FromUnixMilli(1516301758000),
				End:   hawkular.FromUnixMilli(1516301818000),
			},
			{
				Avg:   72.0,
				Start: hawkular.FromUnixMilli(1516301818000),
				End:   hawkular.FromUnixMilli(1516301878000),
			},
		},
		pods: []*v1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					UID: types.UID("hello"),
				},
			},
		},
		namespace: "test",
		startTime: hawkular.FromUnixMilli(1516301758000),
		endTime:   hawkular.FromUnixMilli(1516301878000),
		limit:     1,
	},
}

func TestGetNetworkRecv(t *testing.T) {
	testCases := singleMetricTestCases
	test := &testHawkularGetter{}
	config := &kubernetes.MetricsClientConfig{
		MetricsURL:     "myMetricsServer",
		BearerToken:    "token",
		HawkularGetter: test,
	}
	client, err := kubernetes.NewMetricsClient(config)
	require.NoError(t, err, "Failed to create metrics client")

	for _, testCase := range testCases {
		test.input = testCase
		metric, err := client.GetNetworkRecvMetrics(testCase.pods, testCase.namespace, testCase.startTime)
		require.NoError(t, err, "Getting network metrics failed")
		require.NotNil(t, metric, "Nil result from network metrics")

		// Check that the result has the correct value and timestamp and that the Hawkular API was called
		// with the expected values
		metrics := []*app.TimedNumberTuple{metric}
		output := test.result.output
		verifyMetrics(metrics, testCase, output, "network/rx_rate", t)

		// Verify the remaining filters
		verifySingleMetricFilters(testCase, output.filters, t)
	}
}

func TestGetNetworkRecvRange(t *testing.T) {
	testCases := metricRangeTestCases
	test := &testHawkularGetter{}
	config := &kubernetes.MetricsClientConfig{
		MetricsURL:     "myMetricsServer",
		BearerToken:    "token",
		HawkularGetter: test,
	}
	client, err := kubernetes.NewMetricsClient(config)
	require.NoError(t, err, "Failed to create metrics client")

	for _, testCase := range testCases {
		test.input = testCase
		metrics, err := client.GetNetworkRecvMetricsRange(testCase.pods, testCase.namespace, testCase.startTime,
			testCase.endTime, testCase.limit)
		require.NoError(t, err, "Getting network metrics failed")
		require.NotNil(t, metrics, "Nil result from network metrics")

		// Check that the result has the correct value and timestamp and that the Hawkular API was called
		// with the expected values
		output := test.result.output
		verifyMetrics(metrics, testCase, output, "network/rx_rate", t)

		// Verify the remaining filters
		verifyMetricRangeFilters(testCase, output.filters, t)
	}
}

func TestGetNetworkSent(t *testing.T) {
	testCases := singleMetricTestCases
	test := &testHawkularGetter{}
	config := &kubernetes.MetricsClientConfig{
		MetricsURL:     "myMetricsServer",
		BearerToken:    "token",
		HawkularGetter: test,
	}
	client, err := kubernetes.NewMetricsClient(config)
	require.NoError(t, err, "Failed to create metrics client")

	for _, testCase := range testCases {
		test.input = testCase
		metric, err := client.GetNetworkSentMetrics(testCase.pods, testCase.namespace, testCase.startTime)
		require.NoError(t, err, "Getting network metrics failed")
		require.NotNil(t, metric, "Nil result from network metrics")

		// Check that the result has the correct value and timestamp and that the Hawkular API was called
		// with the expected values
		metrics := []*app.TimedNumberTuple{metric}
		output := test.result.output
		verifyMetrics(metrics, testCase, output, "network/tx_rate", t)

		// Verify the remaining filters
		verifySingleMetricFilters(testCase, output.filters, t)
	}
}

func TestGetNetworkSentRange(t *testing.T) {
	testCases := metricRangeTestCases
	test := &testHawkularGetter{}
	config := &kubernetes.MetricsClientConfig{
		MetricsURL:     "myMetricsServer",
		BearerToken:    "token",
		HawkularGetter: test,
	}
	client, err := kubernetes.NewMetricsClient(config)
	require.NoError(t, err, "Failed to create metrics client")

	for _, testCase := range testCases {
		test.input = testCase
		metrics, err := client.GetNetworkSentMetricsRange(testCase.pods, testCase.namespace, testCase.startTime, testCase.endTime, 0)
		require.NoError(t, err, "Getting network metrics failed")
		require.NotNil(t, metrics, "Nil result from network metrics")

		// Check that the result has the correct value and timestamp and that the Hawkular API was called
		// with the expected values
		output := test.result.output
		verifyMetrics(metrics, testCase, output, "network/tx_rate", t)

		// Verify the remaining filters
		verifyMetricRangeFilters(testCase, output.filters, t)
	}
}

func TestCloseHawkular(t *testing.T) {
	test := &testHawkularGetter{}
	config := &kubernetes.MetricsClientConfig{
		MetricsURL:     "myMetricsServer",
		BearerToken:    "token",
		HawkularGetter: test,
	}
	client, err := kubernetes.NewMetricsClient(config)
	require.NoError(t, err, "Failed to create metrics client")

	// Check that MetricsInterface.Close invokes Hawkular's Client.Close
	client.Close()
	require.True(t, test.result.output.closed, "Hawkular client not closed")
}

func verifyMetrics(metrics []*app.TimedNumberTuple, testCase *testMetricsInput, result *testMetricsOutput,
	gaugeDesc string, t *testing.T) {
	// If limit is specified, check that the number of metrics doesn't exceed that limit
	numMetrics := len(metrics)
	if testCase.limit > 0 {
		require.True(t, numMetrics <= testCase.limit, "Too many metrics returned")
	}

	// Check that the result has the correct values and timestamps
	require.True(t, numMetrics <= len(testCase.buckets), "More metrics than buckets") // Sanity check
	for i := 0; i < numMetrics; i++ {
		// Iterate backwards since earlier buckets may have been discarded due to limit parameter
		metric := metrics[numMetrics-1-i]
		bucket := testCase.buckets[len(testCase.buckets)-1-i]
		require.NotNil(t, metric.Value, "Nil value in network metric")
		require.InEpsilon(t, bucket.Avg, *metric.Value, fltEpsilon, "Incorrect value in network metric")
		require.NotNil(t, metric.Time, "Nil time in network metric")
		require.InEpsilon(t, hawkular.ToUnixMilli(bucket.Start), *metric.Time, fltEpsilon, "Incorrect time in network metric")
	}

	// Check that ReadBuckets was called with the correct inputs
	require.Equal(t, testCase.namespace, result.namespace, "ReadBuckets called with incorrect namespace")
	require.Equal(t, hawkular.Gauge, result.metricType, "Incorrect Hawkular metric type")

	// Check that the tags used in the Hawkular query are correct
	uids := make([]string, len(testCase.pods))
	for idx := range testCase.pods {
		uids[idx] = string(testCase.pods[idx].UID)
	}
	expectedPodTag := strings.Join(uids, "|")
	expectedTags := map[string]string{
		"descriptor_name": gaugeDesc,
		"type":            "pod",
		"pod_id":          expectedPodTag,
	}
	tags := tagsToMap(result.filters.Get("tags"), t)
	if t.Failed() {
		return
	}
	for key, value := range expectedTags {
		require.Equal(t, value, tags[key], "Tag mismatch")
	}
}

func verifySingleMetricFilters(testCase *testMetricsInput, filters url.Values, t *testing.T) {
	require.Equal(t, "1", filters.Get("buckets"), "Buckets parameter missing or incorrect")
	require.Equal(t, strconv.FormatInt(hawkular.ToUnixMilli(testCase.startTime), 10),
		filters.Get("start"), "Start parameter missing or incorrect")
	require.Equal(t, strconv.FormatInt(hawkular.ToUnixMilli(testCase.startTime.Add(time.Minute)), 10),
		filters.Get("end"), "End parameter missing or incorrect")
	require.Equal(t, "true", filters.Get("stacked"), "Stacked parameter missing or incorrect")
}

func verifyMetricRangeFilters(testCase *testMetricsInput, filters url.Values, t *testing.T) {
	require.Equal(t, "60000ms", filters.Get("bucketDuration"), "BucketDuration parameter missing or incorrect")
	require.Equal(t, strconv.FormatInt(hawkular.ToUnixMilli(testCase.startTime), 10),
		filters.Get("start"), "Start parameter missing or incorrect")
	require.Equal(t, strconv.FormatInt(hawkular.ToUnixMilli(testCase.endTime), 10),
		filters.Get("end"), "End parameter missing or incorrect")
	require.Equal(t, "true", filters.Get("stacked"), "Stacked parameter missing or incorrect")
}

func tagsToMap(tagsParam string, t *testing.T) map[string]string {
	tags := strings.Split(tagsParam, ",")
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagSplit := strings.SplitN(tag, ":", 2)
		require.Len(t, tagSplit, 2, "Tag in wrong format")
		tagMap[tagSplit[0]] = tagSplit[1]
	}
	return tagMap
}
