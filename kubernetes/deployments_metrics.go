package kubernetes

import (
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	hawkular "github.com/hawkular/hawkular-client-go/metrics"
	errs "github.com/pkg/errors"
	v1 "k8s.io/client-go/pkg/api/v1"
)

// MetricsClientConfig holds configuration data needed to create a new MetricsInterface
// with kubernetes.NewMetricsClient
type MetricsClientConfig struct {
	// URL to the Kubernetes cluster's metrics server
	MetricsURL string
	// An authorized token to access the cluster
	BearerToken string
	// Provides access to the underlying Hawkular API, uses default implementation if not set
	HawkularGetter
}

type metricsClient struct {
	HawkularRESTAPI
}

// HawkularGetter has a method to access the HawkularRESTAPI interface
type HawkularGetter interface {
	GetHawkularRESTAPI(config *MetricsClientConfig) (HawkularRESTAPI, error)
}

// Metrics provides methods to obtain performance metrics of a deployed application
type Metrics interface {
	GetCPUMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error)
	GetCPUMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
		limit int) ([]*app.TimedNumberTuple, error)
	GetMemoryMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error)
	GetMemoryMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
		limit int) ([]*app.TimedNumberTuple, error)
	GetNetworkSentMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error)
	GetNetworkSentMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
		limit int) ([]*app.TimedNumberTuple, error)
	GetNetworkRecvMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error)
	GetNetworkRecvMetricsRange(pods []*v1.Pod, namespace string, startTime time.Time, endTime time.Time,
		limit int) ([]*app.TimedNumberTuple, error)
	Close()
}

// HawkularRESTAPI collects methods that call out to the Hawkular metrics server over the network
type HawkularRESTAPI interface {
	ReadBuckets(metricType hawkular.MetricType, namespace string,
		modifiers ...hawkular.Modifier) ([]*hawkular.Bucketpoint, error)
	Close()
}

// Default receiver for HawkularRESTAPI methods
type hawkularHelper struct {
	client *hawkular.Client
}

const (
	descriptorTag string = "descriptor_name"
	cpuDesc       string = "cpu/usage_rate"
	memDesc       string = "memory/usage"
	netSent       string = "network/tx_rate"
	netRecv       string = "network/rx_rate"
	typeTag       string = "type"
	typePod       string = "pod"
	podIDTag      string = "pod_id"
)

// Use 1 minute duration for buckets
const bucketDuration = 1 * time.Minute

// CPU metrics are in millicores
// See: https://github.com/openshift/origin-web-console/blob/v3.6.0/app/scripts/services/metricsCharts.js#L15
const millicoreToCoreScale = 0.001
const noScale = 1

// NewMetricsClient creates a Metrics object given a configuration
func NewMetricsClient(config *MetricsClientConfig) (Metrics, error) {
	// Use default implementation if no HawkularGetter is specified
	if config.HawkularGetter == nil {
		config.HawkularGetter = &defaultGetter{}
	}
	helper, err := config.GetHawkularRESTAPI(config)
	if err != nil {
		return nil, err
	}
	mc := &metricsClient{
		HawkularRESTAPI: helper,
	}

	return mc, nil
}

func (*defaultGetter) GetHawkularRESTAPI(config *MetricsClientConfig) (HawkularRESTAPI, error) {
	params := hawkular.Parameters{
		Url:   config.MetricsURL,
		Token: config.BearerToken,
	}
	client, err := hawkular.NewHawkularClient(params)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	helper := &hawkularHelper{
		client: client,
	}
	return helper, nil
}

func (mc *metricsClient) Close() {
	mc.HawkularRESTAPI.Close()
}

func (mc *metricsClient) GetCPUMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return mc.getBucketAverage(pods, namespace, cpuDesc, startTime, millicoreToCoreScale)
}

func (mc *metricsClient) GetCPUMetricsRange(pods []*v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, cpuDesc, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	results := bucketsToTuples(buckets, millicoreToCoreScale)
	return results, nil
}

func (mc *metricsClient) GetMemoryMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return mc.getBucketAverage(pods, namespace, memDesc, startTime, noScale)
}

func (mc *metricsClient) GetMemoryMetricsRange(pods []*v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, memDesc, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	results := bucketsToTuples(buckets, noScale)
	return results, nil
}

func (mc *metricsClient) GetNetworkSentMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return mc.getBucketAverage(pods, namespace, netSent, startTime, noScale)
}

func (mc *metricsClient) GetNetworkSentMetricsRange(pods []*v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, netSent, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	results := bucketsToTuples(buckets, noScale)
	return results, nil
}

func (mc *metricsClient) GetNetworkRecvMetrics(pods []*v1.Pod, namespace string, startTime time.Time) (*app.TimedNumberTuple, error) {
	return mc.getBucketAverage(pods, namespace, netRecv, startTime, noScale)
}

func (mc *metricsClient) GetNetworkRecvMetricsRange(pods []*v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, netRecv, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	results := bucketsToTuples(buckets, noScale)
	return results, nil
}

func bucketsToTuples(buckets []*hawkular.Bucketpoint, scale float64) []*app.TimedNumberTuple {
	results := make([]*app.TimedNumberTuple, len(buckets))
	for idx, bucket := range buckets {
		results[idx] = bucketToTuple(bucket, scale)
	}
	return results
}

func bucketToTuple(bucket *hawkular.Bucketpoint, scale float64) *app.TimedNumberTuple {
	// Use bucket start time as timestamp for data, which is what the OSO web console uses:
	// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/deploymentMetrics.js#L250
	bucketTimeUnix := float64(convertToUnixMillis(bucket.Start))
	scaledAvg := bucket.Avg * scale
	result := &app.TimedNumberTuple{
		Value: &scaledAvg,
		Time:  &bucketTimeUnix,
	}
	return result
}

func convertToUnixMillis(t time.Time) int64 {
	return hawkular.ToUnixMilli(t)
}

func (mc *metricsClient) getBucketAverage(pods []*v1.Pod, namespace, descTag string,
	startTime time.Time, scale float64) (*app.TimedNumberTuple, error) {
	result, err := mc.getLatestBucket(pods, namespace, descTag, startTime)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if result == nil {
		return nil, nil
	}

	tuple := bucketToTuple(result, scale)
	return tuple, err
}

func (mc *metricsClient) getLatestBucket(pods []*v1.Pod, namespace string, descTag string,
	startTime time.Time) (*hawkular.Bucketpoint, error) {
	// Get one bucket after the specified start time
	endTime := startTime.Add(bucketDuration)
	buckets, err := mc.readBuckets(pods, namespace, descTag, hawkular.StartTimeFilter(startTime),
		hawkular.EndTimeFilter(endTime), hawkular.BucketsFilter(1))
	if err != nil {
		return nil, errs.WithStack(err)
	} else if len(buckets) == 0 { // Should have gotten at most one bucket
		return nil, nil
	}
	return buckets[0], nil
}

func (mc *metricsClient) getBucketsInRange(pods []*v1.Pod, namespace string, descTag string, startTime time.Time,
	endTime time.Time, limit int) ([]*hawkular.Bucketpoint, error) {
	// Note: returned buckets are ordered by start time
	// https://github.com/hawkular/hawkular-metrics/blob/0.28.3/core/metrics-model/src/main/java/org/hawkular/metrics/model/BucketPoint.java#L70
	buckets, err := mc.readBuckets(pods, namespace, descTag, hawkular.StartTimeFilter(startTime),
		hawkular.EndTimeFilter(endTime), hawkular.BucketsDurationFilter(bucketDuration))
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Hawkular buckets may extend beyond the requested endpoint if
	// (endTime - startTime) is not evenly divisible by the bucket duration.
	// https://github.com/hawkular/hawkular-metrics/blob/0.28.3/core/metrics-model/src/main/java/org/hawkular/metrics/model/Buckets.java#L156
	//
	// If the end time is in the future, this bucket may be empty or have fewer
	// samples than other buckets. This may create outliers in our charts. So
	// we drop any buckets whose end time is greater than the requested end time
	//
	// For comparison, the OSO web console unconditionally drops the last bucket:
	// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/deploymentMetrics.js#L422
	numBuckets := len(buckets)
	if numBuckets > 0 {
		lastBucket := buckets[numBuckets-1]
		if lastBucket.End.After(endTime) {
			buckets = buckets[:numBuckets-1]
			numBuckets-- // Later used for limit
		}

		// If number of buckets is greater than requested limit n, take newest n buckets
		if limit >= 0 && numBuckets > limit {
			start := numBuckets - limit
			buckets = buckets[start:]
		}
	}

	return buckets, nil
}

func (mc *metricsClient) readBuckets(pods []*v1.Pod, namespace string, descTag string,
	filters ...hawkular.Filter) ([]*hawkular.Bucketpoint, error) {
	numPods := len(pods)
	if numPods == 0 {
		return nil, nil
	}

	// Extract UIDs from pods
	podUIDs := make([]string, numPods)
	for idx, pod := range pods {
		podUIDs[idx] = string(pod.UID)
	}
	// Build Hawkular tags for query
	podsForTag := strings.Join(podUIDs, "|")
	tags := map[string]string{
		descriptorTag: descTag,
		typeTag:       typePod,
		podIDTag:      podsForTag,
	}

	// Append other filters to those provided
	filters = append(filters, hawkular.TagsFilter(tags), hawkular.StackedFilter() /* Sum of each pod */)
	return mc.ReadBuckets(hawkular.Gauge, namespace, hawkular.Filters(filters...))
}

func (helper *hawkularHelper) ReadBuckets(metricType hawkular.MetricType, namespace string,
	modifiers ...hawkular.Modifier) ([]*hawkular.Bucketpoint, error) {
	// Tenant should be set to OSO project name
	helper.client.Tenant = namespace
	return helper.client.ReadBuckets(metricType, modifiers...)
}

func (helper *hawkularHelper) Close() {
	helper.client.Close()
}
