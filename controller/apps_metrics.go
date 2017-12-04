package controller

import (
	"math/big"
	"strings"
	"time"

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
	return mc.getMetricsForPods(pods, namespace, cpuDesc)
}

func (mc *metricsClient) getMemoryMetrics(pods []v1.Pod, namespace string) (float64, int64, error) {
	return mc.getMetricsForPods(pods, namespace, memDesc)
}

func (mc *metricsClient) getBucketAverage(pods []v1.Pod, namespace, descTag string) (float64, error) {
	result, err := mc.readBuckets(pods, namespace, descTag)
	if err != nil {
		return -1, err
	} else if result == nil {
		return -1, nil
	}

	// Return average from bucket
	return result.Avg, err
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

func (mc *metricsClient) readBuckets(pods []v1.Pod, namespace string, descTag string) (*hawkular.Bucketpoint, error) {
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
	// Get a bucket for the last 2 minutes (OSO's bucket duration for last hour shown)
	startTime := time.Now().Add(-120000 * time.Millisecond)
	buckets, err := mc.hawkularClient.ReadBuckets(hawkular.Gauge, hawkular.Filters(hawkular.TagsFilter(tags),
		hawkular.BucketsFilter(1), hawkular.StackedFilter() /* Sum of each pod */, hawkular.StartTimeFilter(startTime)))
	//	hawkular.BucketsDurationFilter(120000*time.Millisecond), hawkular.StartTimeFilter(time.Now().Add(-60*time.Minute)))) What OSO uses
	if err != nil {
		return nil, err
	}

	// XXX Raw request examples:
	// {"tags":"descriptor_name:memory/usage|cpu/usage_rate,type:pod_container,pod_id:myuid1|myuid2,container_name:mycontainer","bucketDuration":"120000ms","start":"-60mn"}
	// {"tags":"descriptor_name:network/tx_rate|network/rx_rate,type:pod,pod_id:myuid1|myuid2","bucketDuration":"120000ms","start":1511293645209}

	// Should have gotten at most one bucket
	if len(buckets) == 0 {
		return nil, nil
	}
	return buckets[0], nil
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
