package kubernetes

import (
	"math/big"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	hawkular "github.com/hawkular/hawkular-client-go/metrics"
	v1 "k8s.io/client-go/pkg/api/v1"
)

type metricsClient struct {
	hawkularClient *hawkular.Client
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

func newMetricsClient(metricsURL string, token string) (*metricsClient, error) {
	params := hawkular.Parameters{
		Url:   metricsURL,
		Token: token,
	}
	client, err := hawkular.NewHawkularClient(params)
	if err != nil {
		return nil, err
	}

	mc := new(metricsClient)
	mc.hawkularClient = client

	return mc, nil
}

func (mc *metricsClient) getCPUMetrics(pods []v1.Pod, namespace string) (float64, int64, error) {
	// CPU metrics are in millicores
	// See: https://github.com/openshift/origin-web-console/blob/v3.6.0/app/scripts/services/metricsCharts.js#L15
	return mc.getBucketAverage(pods, namespace, cpuDesc)
}

func (mc *metricsClient) getCPUMetricsRange(pods []v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, cpuDesc, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	results := bucketsToTuples(buckets)
	return results, nil
}

func (mc *metricsClient) getMemoryMetrics(pods []v1.Pod, namespace string) (float64, int64, error) {
	return mc.getBucketAverage(pods, namespace, memDesc)
}

func (mc *metricsClient) getMemoryMetricsRange(pods []v1.Pod, namespace string,
	startTime time.Time, endTime time.Time, limit int) ([]*app.TimedNumberTuple, error) {
	buckets, err := mc.getBucketsInRange(pods, namespace, memDesc, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	results := bucketsToTuples(buckets)
	return results, nil
}

func bucketsToTuples(buckets []*hawkular.Bucketpoint) []*app.TimedNumberTuple {
	results := make([]*app.TimedNumberTuple, len(buckets))
	for idx, bucket := range buckets {
		// Use bucket start time as timestamp for data, which is what the OSO web console uses:
		// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/deploymentMetrics.js#L250
		bucketTimeUnix := float64(convertToUnixMillis(bucket.Start))
		results[idx] = &app.TimedNumberTuple{
			Value: &bucket.Avg,
			Time:  &bucketTimeUnix,
		}
	}
	return results
}

func convertToUnixMillis(t time.Time) int64 {
	return hawkular.ToUnixMilli(t)
}

func (mc *metricsClient) getBucketAverage(pods []v1.Pod, namespace, descTag string) (float64, int64, error) {
	result, err := mc.getLatestBucket(pods, namespace, descTag)
	if err != nil {
		return -1, -1, err
	} else if result == nil {
		return -1, -1, nil
	}

	// Use start time of bucket as timestamp
	timestamp := hawkular.ToUnixMilli(result.Start)
	// Return average from bucket
	return result.Avg, timestamp, err
}

func (mc *metricsClient) getMetricsForPods(pods []v1.Pod, namespace string, descTag string) (float64, int64, error) {
	// Get most recent sample from each pod's gauge
	samples, err := mc.readRaw(pods, namespace, descTag)
	if err != nil {
		return -1, -1, err
	} else if len(samples) == 0 {
		return -1, -1, nil
	}

	// Return sum of metrics for each pod, and average of timestamp
	var totalValue float64
	timestampsDiffer := false
	for _, sample := range samples {
		totalValue += sample.Value.(float64)
		// If the timestamps are not identical, at least one must differ from the first timestamp
		if !sample.Timestamp.Equal(samples[0].Timestamp) {
			timestampsDiffer = true
		}
	}

	/*
	* Only compute average timestamp in the unlikely case that timestamps are not identical.
	* Heapster uses the end time of its metric collection window as the timestamp for all
	* metrics gathered in that window.
	*
	* https://github.com/kubernetes/heapster/blob/v1.3.0/metrics/sources/kubelet/kubelet.go#L238
	* https://github.com/kubernetes/heapster/blob/v1.3.0/metrics/sinks/hawkular/driver.go#L124
	* https://github.com/kubernetes/heapster/blob/v1.3.0/metrics/sinks/hawkular/client.go#L278
	 */
	var avgTimestamp int64
	if timestampsDiffer {
		avgTimestamp = calcAvgTimestamp(samples)
	} else {
		avgTimestamp = hawkular.ToUnixMilli(samples[0].Timestamp)
	}

	return totalValue, avgTimestamp, err
}

func calcAvgTimestamp(samples []*hawkular.Datapoint) int64 {
	// Use big.Int for intermediate calculation to avoid overflow
	// (and loss of precision)
	bigAvg := big.NewInt(0)
	for _, sample := range samples {
		ts := big.NewInt(hawkular.ToUnixMilli(sample.Timestamp))
		bigAvg = bigAvg.Add(bigAvg, ts)
	}
	numSamples := big.NewInt(int64(len(samples)))
	avg := bigAvg.Div(bigAvg, numSamples).Int64()
	return avg
}

func (mc *metricsClient) getLatestBucket(pods []v1.Pod, namespace string, descTag string) (*hawkular.Bucketpoint, error) {
	// Get a bucket for the last minute
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Minute)
	buckets, err := mc.readBuckets(pods, namespace, descTag, hawkular.StartTimeFilter(startTime),
		hawkular.EndTimeFilter(endTime), hawkular.BucketsFilter(1))
	if err != nil {
		return nil, err
	} else if len(buckets) == 0 { // Should have gotten at most one bucket
		return nil, nil
	}
	return buckets[0], nil
}

// Use 1 minute duration for buckets
const bucketDuration = 1 * time.Minute

func (mc *metricsClient) getBucketsInRange(pods []v1.Pod, namespace string, descTag string, startTime time.Time,
	endTime time.Time, limit int) ([]*hawkular.Bucketpoint, error) {
	// Note: returned buckets are ordered by start time
	// https://github.com/hawkular/hawkular-metrics/blob/0.28.3/core/metrics-model/src/main/java/org/hawkular/metrics/model/BucketPoint.java#L70
	buckets, err := mc.readBuckets(pods, namespace, descTag, hawkular.StartTimeFilter(startTime),
		hawkular.EndTimeFilter(endTime), hawkular.BucketsDurationFilter(bucketDuration))
	if err != nil {
		return nil, err
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

func (mc *metricsClient) readBuckets(pods []v1.Pod, namespace string, descTag string,
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

	// Tenant should be set to OSO project name
	mc.hawkularClient.Tenant = namespace
	// Append other filters to those provided
	filters = append(filters, hawkular.TagsFilter(tags), hawkular.StackedFilter() /* Sum of each pod */)
	return mc.hawkularClient.ReadBuckets(hawkular.Gauge, hawkular.Filters(filters...))
}

func (mc *metricsClient) readRaw(pods []v1.Pod, namespace string, descTag string) ([]*hawkular.Datapoint, error) {
	numPods := len(pods)
	if numPods == 0 {
		return nil, nil
	}

	// Tenant should be set to OSO project name
	mc.hawkularClient.Tenant = namespace
	result := make([]*hawkular.Datapoint, 0, len(pods))
	for _, pod := range pods {
		// Gauge ID is "pod/<pod UID>/<descriptor>"
		gaugeID := typePod + "/" + string(pod.UID) + "/" + descTag
		// Get most recent sample from gauge
		points, err := mc.hawkularClient.ReadRaw(hawkular.Gauge, gaugeID, hawkular.Filters(hawkular.LimitFilter(1),
			hawkular.OrderFilter(hawkular.DESC)))
		if err != nil {
			return nil, err
		}

		// We should have received at most one datapoint
		if len(points) > 0 {
			result = append(result, points[0])
		}
	}
	return result, nil
}
