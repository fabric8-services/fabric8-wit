package kubernetes

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
	rest "k8s.io/client-go/rest"

	"github.com/fabric8-services/fabric8-wit/app"
)

// KubeClientConfig holds configuration data needed to create a new KubeClientInterface
// with kubernetes.NewKubeClient
type KubeClientConfig struct {
	// URL to the Kubernetes cluster's API server
	ClusterURL string
	// An authorized token to access the cluster
	BearerToken string
	// Kubernetes namespace in the cluster of type 'user'
	UserNamespace string
	// Provides access to the Kubernetes REST API, uses default implementation if not set
	KubeRESTAPIGetter
	// Provides access to the metrics API, uses default implementation if not set
	MetricsGetter
}

// KubeRESTAPIGetter has a method to access the KubeRESTAPI interface
type KubeRESTAPIGetter interface {
	GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error)
}

// KubeClientInterface contains configuration and methods for interacting with a Kubernetes cluster
type KubeClientInterface interface {
	GetSpace(spaceName string) (*app.SimpleSpace, error)
	GetApplication(spaceName string, appName string) (*app.SimpleApp, error)
	GetDeployment(spaceName string, appName string, envName string) (*app.SimpleDeployment, error)
	ScaleDeployment(spaceName string, appName string, envName string, deployNumber int) (*int, error)
	GetDeploymentStats(spaceName string, appName string, envName string,
		startTime time.Time) (*app.SimpleDeploymentStats, error)
	GetDeploymentStatSeries(spaceName string, appName string, envName string, startTime time.Time,
		endTime time.Time, limit int) (*app.SimpleDeploymentStatSeries, error)
	GetEnvironments() ([]*app.SimpleEnvironment, error)
	GetEnvironment(envName string) (*app.SimpleEnvironment, error)
	GetPodsInNamespace(nameSpace string, appName string) ([]v1.Pod, error)
}

type kubeClient struct {
	config *KubeClientConfig
	envMap map[string]string
	KubeRESTAPI
	MetricsInterface
}

// KubeRESTAPI collects methods that call out to the Kubernetes API server over the network
type KubeRESTAPI interface {
	corev1.CoreV1Interface
}

type deployment struct {
	dcUID      types.UID
	appVersion string
	currentUID types.UID
}

// Receiver for default implementation of KubeRESTAPIGetter and MetricsGetter
type defaultGetter struct{}

// NewKubeClient creates a KubeClientInterface given a configuration
func NewKubeClient(config *KubeClientConfig) (KubeClientInterface, error) {
	// Use default implementation if no KubernetesGetter is specified
	if config.KubeRESTAPIGetter == nil {
		config.KubeRESTAPIGetter = defaultGetter{}
	}
	kubeAPI, err := config.GetKubeRESTAPI(config)
	if err != nil {
		return nil, err
	}

	// Use default implementation if no MetricsGetter is specified
	if config.MetricsGetter == nil {
		config.MetricsGetter = defaultGetter{}
	}
	// In the absence of a better way to get the user's metrics URL,
	// substitute "api" with "metrics" in user's cluster URL
	metricsURL, err := getMetricsURLFromAPIURL(config.ClusterURL)
	if err != nil {
		return nil, err
	}
	// Create MetricsClient for talking with Hawkular API
	metrics, err := config.GetMetrics(metricsURL, config.BearerToken)
	if err != nil {
		return nil, err
	}

	kubeClient := &kubeClient{
		config:           config,
		KubeRESTAPI:      kubeAPI,
		MetricsInterface: metrics,
	}

	// Get environments from config map
	envMap, err := kubeClient.getEnvironmentsFromConfigMap()
	if err != nil {
		return nil, err
	}
	kubeClient.envMap = envMap
	return kubeClient, nil
}

func (defaultGetter) GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error) {
	restConfig := &rest.Config{
		Host:        config.ClusterURL,
		BearerToken: config.BearerToken,
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1(), nil
}

func (defaultGetter) GetMetrics(metricsURL string, bearerToken string) (MetricsInterface, error) {
	return newMetricsClient(metricsURL, bearerToken)
}

// GetSpace returns a space matching the provided name, containing all applications that belong to it
func (kc *kubeClient) GetSpace(spaceName string) (*app.SimpleSpace, error) {
	// Get BuildConfigs within the user namespace that have a matching 'space' label
	// This is similar to how pipelines are displayed in fabric8-ui
	// https://github.com/fabric8-ui/fabric8-ui/blob/master/src/app/space/create/pipelines/pipelines.component.ts
	buildconfigs, err := kc.getBuildConfigs(spaceName)
	if err != nil {
		return nil, err
	}

	// Get all applications in this space using BuildConfig names
	apps := make([]*app.SimpleApp, 0)
	for _, bc := range buildconfigs {
		appn, err := kc.GetApplication(spaceName, bc)
		if err != nil {
			return nil, err
		}
		apps = append(apps, appn)
	}

	result := &app.SimpleSpace{
		Applications: apps,
	}

	return result, nil
}

// GetApplication retrieves an application with the given space and application names, with the status
// of that application's deployment in each environment
func (kc *kubeClient) GetApplication(spaceName string, appName string) (*app.SimpleApp, error) {
	// Get all deployments of this app for each environment in this space
	deployments := make([]*app.SimpleDeployment, 0)
	for envName := range kc.envMap {
		deployment, err := kc.GetDeployment(spaceName, appName, envName)
		if err != nil {
			return nil, err
		} else if deployment != nil {
			deployments = append(deployments, deployment)
		}
	}

	result := &app.SimpleApp{
		Name:     &appName,
		Pipeline: deployments,
	}
	return result, nil
}

// ScaleDeployment adjusts the desired number of replicas for a specified application, returning the
// previous number of desired replicas
func (kc *kubeClient) ScaleDeployment(spaceName string, appName string, envName string, deployNumber int) (*int, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	// Look up the Scale for the DeploymentConfig corresponding to the application name in the provided environment
	dcScaleURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s/scale", envNS, appName)
	scale, err := kc.getResource(dcScaleURL, true)
	if err != nil {
		return nil, err
	} else if scale == nil {
		return nil, nil
	}

	spec, ok := scale["spec"].(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("Invalid deployment config returned from endpoint: missing 'spec'")
	}

	replicasYaml, pres := spec["replicas"]
	oldReplicas := 0 // replicas property may be missing from spec if set to 0
	if pres {
		oldReplicas, ok = replicasYaml.(int)
		if !ok {
			return nil, errors.New("Invalid deployment config returned from endpoint: 'replicas' is not an integer")
		}
	}
	spec["replicas"] = deployNumber

	yamlScale, err := yaml.Marshal(scale)
	if err != nil {
		return nil, err
	}

	_, err = kc.putResource(dcScaleURL, yamlScale)
	if err != nil {
		return nil, err
	}

	return &oldReplicas, nil
}

// GetDeployment returns information about the current deployment of an application within a
// particular environment. The application must exist within the provided space.
func (kc *kubeClient) GetDeployment(spaceName string, appName string, envName string) (*app.SimpleDeployment, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || len(deploy.currentUID) == 0 {
		return nil, nil
	}

	// Get all pods created by this deployment
	pods, err := kc.getPods(envNS, deploy.currentUID)
	if err != nil {
		return nil, err
	}
	// Get the status of each pod in the deployment
	podStats, err := kc.getPodStatus(pods)
	if err != nil {
		return nil, err
	}

	verString := string(deploy.appVersion)
	result := &app.SimpleDeployment{
		Name:    &envName,
		Version: &verString,
		Pods:    podStats,
	}
	return result, nil
}

// GetDeploymentStats returns performance metrics of an application for a period of 1 minute
// beyond the specified start time, which are then aggregated into a single data point.
func (kc *kubeClient) GetDeploymentStats(spaceName string, appName string, envName string,
	startTime time.Time) (*app.SimpleDeploymentStats, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || len(deploy.currentUID) == 0 {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.currentUID)
	if err != nil {
		return nil, err
	}

	// Gather the statistics we need about the current deployment
	cpuUsage, err := kc.GetCPUMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}
	memoryUsage, err := kc.GetMemoryMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}

	result := &app.SimpleDeploymentStats{
		Cores:  cpuUsage,
		Memory: memoryUsage,
	}

	return result, nil
}

// GetDeploymentStatSeries returns performance metrics of an application as a time series bounded by
// the provided time range in startTime and endTime. If there are more data points than the
// limit argument, only the newest datapoints within that limit are returned.
func (kc *kubeClient) GetDeploymentStatSeries(spaceName string, appName string, envName string,
	startTime time.Time, endTime time.Time, limit int) (*app.SimpleDeploymentStatSeries, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}

	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || len(deploy.currentUID) == 0 {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.currentUID)
	if err != nil {
		return nil, err
	}

	// Get CPU and memory metrics for pods in deployment
	cpuMetrics, err := kc.GetCPUMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	memoryMetrics, err := kc.GetMemoryMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	// Get the earliest and latest timestamps
	minTime, maxTime := getTimestampEndpoints(cpuMetrics, memoryMetrics)
	result := &app.SimpleDeploymentStatSeries{
		Cores:  cpuMetrics,
		Memory: memoryMetrics,
		Start:  minTime,
		End:    maxTime,
	}

	return result, nil
}

// GetEnvironments retrieves information on all environments in the cluster
// for the current user
func (kc *kubeClient) GetEnvironments() ([]*app.SimpleEnvironment, error) {
	envs := make([]*app.SimpleEnvironment, 0)
	for envName := range kc.envMap {
		env, err := kc.GetEnvironment(envName)
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}

// GetEnvironment returns information on an environment with the provided name
func (kc *kubeClient) GetEnvironment(envName string) (*app.SimpleEnvironment, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}

	envStats, err := kc.getResourceQuota(envNS)
	if err != nil {
		return nil, err
	}

	env := &app.SimpleEnvironment{
		Name:  &envName,
		Quota: envStats,
	}
	return env, nil
}

func getMetricsURLFromAPIURL(apiURLStr string) (string, error) {
	// Parse as URL to give us easy access to the hostname
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return "", err
	}

	// Get the hostname (without port) and replace api prefix with metrics
	apiHostname := apiURL.Hostname()
	if !strings.HasPrefix(apiHostname, "api") {
		return "", errors.New("Cluster URL does not begin with \"api\": " + apiHostname)
	}
	metricsHostname := strings.Replace(apiHostname, "api", "metrics", 1)
	// Construct URL using just scheme from API URL and metrics hostname
	metricsURL := url.URL{
		Scheme: apiURL.Scheme,
		Host:   metricsHostname,
	}
	return metricsURL.String(), nil
}

func getTimestampEndpoints(metricsSeries ...[]*app.TimedNumberTuple) (minTime, maxTime *float64) {
	// Metrics arrays are ordered by timestamp, so just check beginning and end
	for _, series := range metricsSeries {
		if len(series) > 0 {
			first := series[0].Time
			if minTime == nil || *first < *minTime {
				minTime = first
			}
			last := series[len(series)-1].Time
			if maxTime == nil || *last > *maxTime {
				maxTime = last
			}
		}
	}
	return minTime, maxTime
}

func (kc *kubeClient) getBuildConfigs(space string) ([]string, error) {
	// BuildConfigs are OpenShift objects, so access REST API using HTTP directly until
	// there is a Go client for OpenShift

	// BuildConfigs created by fabric8 have a "space" label indicating the space they belong to
	queryParam := url.QueryEscape("space=" + space)
	bcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/buildconfigs?labelSelector=%s", kc.config.UserNamespace, queryParam)
	result, err := kc.getResource(bcURL, false)
	if err != nil {
		return nil, err
	}
	// Parse build configs from result
	kind, ok := result["kind"].(string)
	if !ok || kind != "BuildConfigList" {
		return nil, errors.New("No build configs returned from endpoint")
	}
	items, ok := result["items"].([]interface{})
	if !ok {
		return nil, errors.New("Malformed response from endpoint")
	}

	// Extract the names of the BuildConfigs from the response
	buildconfigs := make([]string, 0)
	for _, item := range items {
		bc, ok := item.(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("Malformed build config")
		}
		metadata, ok := bc["metadata"].(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("Metadata missing from build config")
		}
		name, ok := metadata["name"].(string)
		if !ok || len(name) == 0 {
			return nil, errors.New("Malformed metadata in build config")
		}
		buildconfigs = append(buildconfigs, name)
	}
	return buildconfigs, nil
}

func (kc *kubeClient) getEnvironmentsFromConfigMap() (map[string]string, error) {
	// fabric8 creates a ConfigMap in the user namespace with information on environments
	const envConfigMap string = "fabric8-environments"
	const providerLabel string = "fabric8"
	configmap, err := kc.ConfigMaps(kc.config.UserNamespace).Get(envConfigMap, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	// Check that config map has the expected label
	if configmap.Labels["provider"] != providerLabel {
		return nil, errors.New("Unknown or missing provider for environments config map")
	}
	// Parse config map data to construct environments map
	envMap := make(map[string]string)
	const namespaceProp string = "namespace"
	// Config map keys are environment names
	for key, value := range configmap.Data {
		// Look through value for namespace property
		var namespace string
		lines := strings.Split(value, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, namespaceProp) {
				tokens := strings.SplitN(line, ":", 2)
				if len(tokens) < 2 {
					return nil, errors.New("Malformed environments config map")
				}
				namespace = strings.TrimSpace(tokens[1])
			}
		}
		if len(namespace) == 0 {
			return nil, errors.New("No namespace for environment " + key + " in config map")
		}
		envMap[key] = namespace
	}
	return envMap, nil
}

func (kc *kubeClient) getEnvironmentNamespace(envName string) (string, error) {
	envNS, pres := kc.envMap[envName]
	if !pres {
		return "", errors.New("Unknown environment: " + envName)
	}
	return envNS, nil
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (kc *kubeClient) putResource(url string, putBody []byte) (*string, error) {
	fullURL := strings.TrimSuffix(kc.config.ClusterURL, "/") + url
	req, err := http.NewRequest("PUT", fullURL, bytes.NewBuffer(putBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("Authorization", "Bearer "+kc.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	status := resp.StatusCode
	if status < 200 || status > 300 {
		return nil, fmt.Errorf("Failed to PUT url %s: status code %d", fullURL, status)
	}
	bodyStr := string(body)
	return &bodyStr, nil
}

func (kc *kubeClient) getDeploymentConfig(namespace string, appName string, space string) (*deployment, error) {
	dcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, appName)
	result, err := kc.getResource(dcURL, true)
	if err != nil {
		return nil, err
	} else if result == nil {
		return nil, nil
	}

	// Parse deployment config from result
	kind, ok := result["kind"].(string)
	if !ok || kind != "DeploymentConfig" {
		return nil, errors.New("No deployment config returned from endpoint")
	}
	metadata, ok := result["metadata"].(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("Metadata missing from deployment config")
	}
	// Check the space label is what we expect
	labels, ok := metadata["labels"].(map[interface{}]interface{})
	if !ok {
		return nil, errors.New("Labels missing from deployment config")
	}
	spaceLabel, ok := labels["space"].(string)
	if !ok || len(spaceLabel) == 0 {
		return nil, errors.New("Space label missing from deployment config")
	}
	if spaceLabel != space {
		return nil, errors.New("Deployment config " + appName + " is part of space " +
			spaceLabel + ", expected space " + space)
	}
	// Get UID from deployment config
	uid, ok := metadata["uid"].(string)
	if !ok || len(uid) == 0 {
		return nil, errors.New("Malformed metadata in deployment config")
	}
	// Read application version from label
	version := labels["version"].(string)
	if !ok || len(version) == 0 {
		return nil, errors.New("Version missing from deployment config")
	}

	dc := &deployment{
		dcUID:      types.UID(uid),
		appVersion: version,
	}
	return dc, nil
}

func (kc *kubeClient) getCurrentDeployment(space string, appName string, namespace string) (*deployment, error) {
	// Look up DeploymentConfig corresponding to the application name in the provided environment
	result, err := kc.getDeploymentConfig(namespace, appName, space)
	if err != nil {
		return nil, err
	} else if result == nil {
		return nil, nil
	}
	// Find the current deployment for the DC we just found. This should correspond to the deployment
	// shown in the OpenShift web console's overview page
	rcs, err := kc.getReplicationControllers(namespace, result.dcUID)
	if err != nil {
		return nil, err
	} else if len(rcs) == 0 {
		return result, nil
	}

	// Find newest RC created by this DC, which is also considered visible according to the
	// OpenShift web console's criteria:
	// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/controllers/overview.js#L658
	const deploymentPhaseAnnotation string = "openshift.io/deployment.phase"
	var newest *v1.ReplicationController
	for idx := range rcs {
		rc := &rcs[idx]
		if newest == nil || newest.CreationTimestamp.Before(rc.CreationTimestamp) {
			visible := false
			// Check if this RC has replicas running
			if rc.Status.Replicas > 0 {
				visible = true
			} else { // Check if RC is in progress
				phase := rc.Annotations[deploymentPhaseAnnotation]
				if phase == "New" || phase == "Pending" || phase == "Running" {
					visible = true
				}
			}
			if visible {
				newest = rc
			}
		}
	}
	if newest != nil {
		result.currentUID = newest.UID
	}
	return result, nil
}

func (kc *kubeClient) getReplicationControllers(namespace string, dcUID types.UID) ([]v1.ReplicationController, error) {
	rcs, err := kc.ReplicationControllers(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Current Kubernetes concept used to represent OpenShift Deployments
	rcsForDc := make([]v1.ReplicationController, 0)
	for _, rc := range rcs.Items {

		// Use OwnerReferences to map RC to DC that created it
		match := false
		for _, ref := range rc.OwnerReferences {
			if ref.UID == dcUID && ref.Controller != nil && *ref.Controller {
				match = true
			}
		}
		if match {
			rcsForDc = append(rcsForDc, rc)
		}
	}

	return rcsForDc, nil
}

func (kc *kubeClient) getResourceQuota(namespace string) (*app.EnvStats, error) {
	const computeResources string = "compute-resources"
	quota, err := kc.ResourceQuotas(namespace).Get(computeResources, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	} else if quota == nil {
		return nil, errors.New("No resource quota with name: " + computeResources)
	}

	// Convert quantities to floating point, as this should provide enough
	// precision in practice
	cpuLimit, err := quantityToFloat64(quota.Status.Hard[v1.ResourceLimitsCPU])
	if err != nil {
		return nil, err
	}
	cpuUsed, err := quantityToFloat64(quota.Status.Used[v1.ResourceLimitsCPU])
	if err != nil {
		return nil, err
	}

	cpuStats := &app.EnvStatCores{
		Quota: &cpuLimit,
		Used:  &cpuUsed,
	}

	memLimit, err := quantityToFloat64(quota.Status.Hard[v1.ResourceLimitsMemory])
	if err != nil {
		return nil, err
	}

	memUsed, err := quantityToFloat64(quota.Status.Used[v1.ResourceLimitsMemory])
	if err != nil {
		return nil, err
	}

	memUnits := "bytes"
	memStats := &app.EnvStatMemory{
		Quota: &memLimit,
		Used:  &memUsed,
		Units: &memUnits,
	}

	result := &app.EnvStats{
		Cpucores: cpuStats,
		Memory:   memStats,
	}

	return result, nil
}

func quantityToFloat64(q resource.Quantity) (float64, error) {
	val64, rc := q.AsInt64()
	var result float64
	if rc {
		result = float64(val64)
	} else {
		valDec := q.AsDec()
		val64, ok := valDec.Unscaled()
		if !ok {
			return -1, errors.New(valDec.String() + " cannot be represented as 64-bit integer")
		}
		// From dec.go: The mathematical value of a Dec equals: unscaled * 10**(-scale)
		result = float64(val64) * math.Pow10(-int(valDec.Scale()))
	}
	return result, nil
}

// GetPodsInNamespace - return all pods in namepsace 'nameSpace' and application 'appName'
func (kc *kubeClient) GetPodsInNamespace(nameSpace string, appName string) ([]v1.Pod, error) {
	listOptions := metaV1.ListOptions{
		LabelSelector: "app=" + appName,
	}
	pods, err := kc.Pods(nameSpace).List(listOptions)
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (kc *kubeClient) getPods(namespace string, uid types.UID) ([]v1.Pod, error) {
	pods, err := kc.Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	appPods := make([]v1.Pod, 0)
	for _, pod := range pods.Items {
		// If a pod belongs to a given RC, it should have an OwnerReference
		// whose UID matches that of the RC
		// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/services/ownerReferences.js#L40
		match := false
		for _, ref := range pod.OwnerReferences {
			if ref.UID == uid && ref.Controller != nil && *ref.Controller {
				match = true
			}
		}
		if match {
			appPods = append(appPods, pod)
		}
	}

	return appPods, nil
}

func (kc *kubeClient) getPodStatus(pods []v1.Pod) (*app.PodStats, error) {
	var starting, running, stopping int
	/*
	 * TODO Logic for pod phases in web console is calculated in the UI:
	 * https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/podDonut.js
	 * https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/filters/resources.js
	 * Should we duplicate the logic here in Go, opt for simpler phases (perhaps just PodPhase), or send Pod as JSON to fabric8-ui
	 * to reuse JS components
	 * const phases = []string{"Running", "Not Ready", "Warning", "Error", "Pulling", "Pending", "Succeeded", "Terminating", "Unknown"}
	 */
	for _, pod := range pods {
		// Terminating pods have a deletionTimeStamp set
		if pod.ObjectMeta.DeletionTimestamp != nil {
			stopping++
		} else if pod.Status.Phase == v1.PodPending {
			// TODO Is this a good approximation of "Starting"?
			starting++
		} else if pod.Status.Phase == v1.PodRunning {
			running++
		} else {
			// TODO Handle other phases
		}
	}

	total := len(pods)
	result := &app.PodStats{
		Starting: &starting,
		Running:  &running,
		Stopping: &stopping,
		Total:    &total,
	}

	return result, nil
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (kc *kubeClient) getResource(url string, allowMissing bool) (map[interface{}]interface{}, error) {
	var body []byte
	fullURL := strings.TrimSuffix(kc.config.ClusterURL, "/") + url
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("Authorization", "Bearer "+kc.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()

	status := resp.StatusCode
	if status == 404 && allowMissing {
		return nil, nil
	} else if status < 200 || status > 300 {
		return nil, fmt.Errorf("Failed to GET url %s due to status code %d", fullURL, status)
	}
	var respType map[interface{}]interface{}
	err = yaml.Unmarshal(b, &respType)
	if err != nil {
		return nil, err
	}
	return respType, nil
}
