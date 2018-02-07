package kubernetes

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
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

	errs "github.com/pkg/errors"
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
	// hook to inject build configs for testing
	BuildConfig
}

// KubeRESTAPIGetter has a method to access the KubeRESTAPI interface
type KubeRESTAPIGetter interface {
	GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error)
}

// MetricsGetter has a method to access the Metrics interface
type MetricsGetter interface {
	GetMetrics(config *MetricsClientConfig) (Metrics, error)
}

// BuildConfig will provide build configs for testing
type BuildConfig interface {
	GetBuildConfigs(space string) ([]string, error)
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
	Close()
}

type kubeClient struct {
	config *KubeClientConfig
	envMap map[string]string
	KubeRESTAPI
	Metrics
	BuildConfig
}

// KubeRESTAPI collects methods that call out to the Kubernetes API server over the network
type KubeRESTAPI interface {
	corev1.CoreV1Interface
}

type deployment struct {
	dcUID      types.UID
	appVersion string
	current    *v1.ReplicationController
}

type route struct {
	host string
	path string
	tls  bool
	// Scoring criteria below
	hasAdmitted          bool
	hasAlternateBackends bool
	isCustomHost         bool
}

// Receiver for default implementation of KubeRESTAPIGetter and MetricsGetter
type defaultGetter struct{}

// NewKubeClient creates a KubeClientInterface given a configuration
func NewKubeClient(config *KubeClientConfig) (KubeClientInterface, error) {
	// Use default implementation if no KubernetesGetter is specified
	if config.KubeRESTAPIGetter == nil {
		config.KubeRESTAPIGetter = &defaultGetter{}
	}
	kubeAPI, err := config.GetKubeRESTAPI(config)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Use default implementation if no MetricsGetter is specified
	if config.MetricsGetter == nil {
		config.MetricsGetter = &defaultGetter{}
	}
	// In the absence of a better way to get the user's metrics URL,
	// substitute "api" with "metrics" in user's cluster URL
	metricsURL, err := getMetricsURLFromAPIURL(config.ClusterURL)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Create MetricsClient for talking with Hawkular API
	metricsConfig := &MetricsClientConfig{
		MetricsURL:  metricsURL,
		BearerToken: config.BearerToken,
	}
	metrics, err := config.GetMetrics(metricsConfig)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Get environments from config map
	envMap, err := getEnvironmentsFromConfigMap(kubeAPI, config.UserNamespace)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	kubeClient := &kubeClient{
		config:      config,
		envMap:      envMap,
		KubeRESTAPI: kubeAPI,
		Metrics:     metrics,
		BuildConfig: config.BuildConfig,
	}
	return kubeClient, nil
}

func (*defaultGetter) GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error) {
	restConfig := &rest.Config{
		Host:        config.ClusterURL,
		BearerToken: config.BearerToken,
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return clientset.CoreV1(), nil
}

func (*defaultGetter) GetMetrics(config *MetricsClientConfig) (Metrics, error) {
	return NewMetricsClient(config)
}

// Close releases any resources held by this KubeClientInterface
func (kc *kubeClient) Close() {
	// Metrics client needs to be closed to stop Hawkular go-routine from spinning
	kc.Metrics.Close()
}

// GetSpace returns a space matching the provided name, containing all applications that belong to it
func (kc *kubeClient) GetSpace(spaceName string) (*app.SimpleSpace, error) {
	// Get BuildConfigs within the user namespace that have a matching 'space' label
	// This is similar to how pipelines are displayed in fabric8-ui
	// https://github.com/fabric8-ui/fabric8-ui/blob/master/src/app/space/create/pipelines/pipelines.component.ts
	buildconfigs, err := kc.getBuildConfigs(spaceName)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Get all applications in this space using BuildConfig names
	apps := []*app.SimpleApp{}
	for _, bc := range buildconfigs {
		appn, err := kc.GetApplication(spaceName, bc)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		apps = append(apps, appn)
	}

	result := &app.SimpleSpace{
		Type: "space",
		Attributes: &app.SimpleSpaceAttributes{
			Name:         spaceName,
			Applications: apps,
		},
	}

	return result, nil
}

// GetApplication retrieves an application with the given space and application names, with the status
// of that application's deployment in each environment
func (kc *kubeClient) GetApplication(spaceName string, appName string) (*app.SimpleApp, error) {
	// Get all deployments of this app for each environment in this space
	deployments := []*app.SimpleDeployment{}
	for envName := range kc.envMap {
		deployment, err := kc.GetDeployment(spaceName, appName, envName)
		if err != nil {
			return nil, errs.WithStack(err)
		} else if deployment != nil {
			deployments = append(deployments, deployment)
		}
	}

	result := &app.SimpleApp{
		Type: "application",
		Attributes: &app.SimpleAppAttributes{
			Name:        appName,
			Deployments: deployments,
		},
		ID: appName,
	}
	return result, nil
}

// ScaleDeployment adjusts the desired number of replicas for a specified application, returning the
// previous number of desired replicas
func (kc *kubeClient) ScaleDeployment(spaceName string, appName string, envName string, deployNumber int) (*int, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Look up the Scale for the DeploymentConfig corresponding to the application name in the provided environment
	dcScaleURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s/scale", envNS, appName)
	scale, err := kc.getResource(dcScaleURL, true)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if scale == nil {
		return nil, nil
	}

	spec, ok := scale["spec"].(map[interface{}]interface{})
	if !ok {
		return nil, errs.New("invalid deployment config returned from endpoint: missing 'spec'")
	}

	replicasYaml, pres := spec["replicas"]
	oldReplicas := 0 // replicas property may be missing from spec if set to 0
	if pres {
		oldReplicas, ok = replicasYaml.(int)
		if !ok {
			return nil, errs.New("invalid deployment config returned from endpoint: 'replicas' is not an integer")
		}
	}
	spec["replicas"] = deployNumber

	yamlScale, err := yaml.Marshal(scale)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	_, err = kc.putResource(dcScaleURL, yamlScale)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	return &oldReplicas, nil
}

func (kc *kubeClient) getConsoleURL(envNS string) (*string, error) {
	path := fmt.Sprintf("console/project/%s", envNS)
	// Replace "api" prefix with "console" and append path
	consoleURL, err := modifyURL(kc.config.ClusterURL, "console", path)
	if err != nil {
		return nil, err
	}
	consoleURLStr := consoleURL.String()
	return &consoleURLStr, nil
}

func (kc *kubeClient) getLogURL(envNS string, deploy *deployment) (*string, error) {
	consoleURL, err := kc.getConsoleURL(envNS)
	if err != nil {
		return nil, err
	}
	rcName := deploy.current.Name
	logURL := fmt.Sprintf("%s/browse/rc/%s?tab=logs", *consoleURL, rcName)
	return &logURL, nil
}

func (kc *kubeClient) getApplicationURL(envNS string, deploy *deployment) (*string, error) {
	// Get the best route to the application to show to the user
	routeURL, err := kc.getBestRoute(envNS, deploy)
	if err != nil {
		return nil, err
	}
	var result *string
	if routeURL != nil {
		route := routeURL.String()
		result = &route
	}
	return result, nil
}

// GetDeployment returns information about the current deployment of an application within a
// particular environment. The application must exist within the provided space.
func (kc *kubeClient) GetDeployment(spaceName string, appName string, envName string) (*app.SimpleDeployment, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get all pods created by this deployment
	pods, err := kc.getPods(envNS, deploy.current.UID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Get the status of each pod in the deployment
	podStats, total := kc.getPodStatus(pods)

	// Get related URLs for the deployment
	appURL, err := kc.getApplicationURL(envNS, deploy)
	if err != nil {
		return nil, err
	}
	consoleURL, err := kc.getConsoleURL(envNS)
	if err != nil {
		return nil, err
	}
	logURL, err := kc.getLogURL(envNS, deploy)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	var links *app.GenericLinksForDeployment
	if consoleURL != nil || appURL != nil || logURL != nil {
		links = &app.GenericLinksForDeployment{
			Console:     consoleURL,
			Logs:        logURL,
			Application: appURL,
		}
	}

	verString := string(deploy.appVersion)
	result := &app.SimpleDeployment{
		Type: "deployment",
		Attributes: &app.SimpleDeploymentAttributes{
			Name:     envName,
			Version:  &verString,
			Pods:     podStats,
			PodTotal: &total,
		},
		ID:    envName,
		Links: links,
	}
	return result, nil
}

// GetDeploymentStats returns performance metrics of an application for a period of 1 minute
// beyond the specified start time, which are then aggregated into a single data point.
func (kc *kubeClient) GetDeploymentStats(spaceName string, appName string, envName string,
	startTime time.Time) (*app.SimpleDeploymentStats, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.current.UID)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Gather the statistics we need about the current deployment
	cpuUsage, err := kc.GetCPUMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	memoryUsage, err := kc.GetMemoryMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	netTxUsage, err := kc.GetNetworkSentMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}
	netRxUsage, err := kc.GetNetworkRecvMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}

	result := &app.SimpleDeploymentStats{
		Type: "deploymentstats",
		Attributes: &app.SimpleDeploymentStatsAttributes{
			Cores:  cpuUsage,
			Memory: memoryUsage,
			NetTx:  netTxUsage,
			NetRx:  netRxUsage,
		},
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
		return nil, errs.WithStack(err)
	}

	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.current.UID)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Get CPU, memory and network metrics for pods in deployment
	cpuMetrics, err := kc.GetCPUMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	memoryMetrics, err := kc.GetMemoryMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	netTxMetrics, err := kc.GetNetworkSentMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	netRxMetrics, err := kc.GetNetworkRecvMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}

	// Get the earliest and latest timestamps
	minTime, maxTime := getTimestampEndpoints(cpuMetrics, memoryMetrics)
	result := &app.SimpleDeploymentStatSeries{
		Cores:  cpuMetrics,
		Memory: memoryMetrics,
		NetTx:  netTxMetrics,
		NetRx:  netRxMetrics,
		Start:  minTime,
		End:    maxTime,
	}

	return result, nil
}

// GetEnvironments retrieves information on all environments in the cluster
// for the current user
func (kc *kubeClient) GetEnvironments() ([]*app.SimpleEnvironment, error) {
	envs := []*app.SimpleEnvironment{}
	for envName := range kc.envMap {
		env, err := kc.GetEnvironment(envName)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		envs = append(envs, env)
	}
	return envs, nil
}

// GetEnvironment returns information on an environment with the provided name
func (kc *kubeClient) GetEnvironment(envName string) (*app.SimpleEnvironment, error) {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	envStats, err := kc.getResourceQuota(envNS)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	env := &app.SimpleEnvironment{
		Type: "environment",
		Attributes: &app.SimpleEnvironmentAttributes{
			Name:  &envName,
			Quota: envStats,
		},
	}
	return env, nil
}

func getMetricsURLFromAPIURL(apiURLStr string) (string, error) {
	metricsURL, err := modifyURL(apiURLStr, "metrics", "")
	if err != nil {
		return "", err
	}
	return metricsURL.String(), nil
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

	// hook for testing
	if kc.config.BuildConfig != nil {
		return kc.config.BuildConfig.GetBuildConfigs(space)
	}

	// BuildConfigs are OpenShift objects, so access REST API using HTTP directly until
	// there is a Go client for OpenShift

	// BuildConfigs created by fabric8 have a "space" label indicating the space they belong to
	queryParam := url.QueryEscape("space=" + space)
	bcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/buildconfigs?labelSelector=%s", kc.config.UserNamespace, queryParam)
	result, err := kc.getResource(bcURL, false)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Parse build configs from result
	kind, ok := result["kind"].(string)
	if !ok || kind != "BuildConfigList" {
		return nil, errs.New("no build configs returned from endpoint")
	}
	items, ok := result["items"].([]interface{})
	if !ok {
		return nil, errs.New("malformed response from endpoint")
	}

	// Extract the names of the BuildConfigs from the response
	buildconfigs := []string{}
	for _, item := range items {
		bc, ok := item.(map[interface{}]interface{})
		if !ok {
			return nil, errs.New("malformed build config")
		}
		metadata, ok := bc["metadata"].(map[interface{}]interface{})
		if !ok {
			return nil, errs.New("'metadata' object missing from build config")
		}
		name, ok := metadata["name"].(string)
		if !ok || len(name) == 0 {
			return nil, errs.New("malformed metadata in build config; 'name' is missing or invalid")
		}
		buildconfigs = append(buildconfigs, name)
	}
	return buildconfigs, nil
}

func getEnvironmentsFromConfigMap(kube KubeRESTAPI, userNamespace string) (map[string]string, error) {
	// fabric8 creates a ConfigMap in the user namespace with information on environments
	const envConfigMap string = "fabric8-environments"
	const providerLabel string = "fabric8"
	configmap, err := kube.ConfigMaps(userNamespace).Get(envConfigMap, metaV1.GetOptions{})
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Check that config map has the expected label
	if configmap.Labels["provider"] != providerLabel {
		return nil, errs.Errorf("unknown or missing provider %s for environments config map", providerLabel)
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
					return nil, errs.New("malformed environments config map")
				}
				namespace = strings.TrimSpace(tokens[1])
			}
		}
		if len(namespace) == 0 {
			return nil, errs.Errorf("no namespace for environment %s in config map", key)
		}
		envMap[key] = namespace
	}
	return envMap, nil
}

func (kc *kubeClient) getEnvironmentNamespace(envName string) (string, error) {
	envNS, pres := kc.envMap[envName]
	if !pres {
		return "", errs.Errorf("unknown environment: %s", envName)
	}
	return envNS, nil
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (kc *kubeClient) putResource(url string, putBody []byte) (*string, error) {
	fullURL := strings.TrimSuffix(kc.config.ClusterURL, "/") + url
	req, err := http.NewRequest("PUT", fullURL, bytes.NewBuffer(putBody))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("Authorization", "Bearer "+kc.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	status := resp.StatusCode
	if httpStatusFailed(status) {
		return nil, errs.Errorf("failed to PUT url %s: status code %d", fullURL, status)
	}
	bodyStr := string(body)
	return &bodyStr, nil
}

func (kc *kubeClient) getDeploymentConfig(namespace string, appName string, space string) (*deployment, error) {
	dcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, appName)
	result, err := kc.getResource(dcURL, true)
	if err != nil {
		return nil, errs.WithStack(err)
	} else if result == nil {
		return nil, nil
	}

	// Parse deployment config from result
	kind, ok := result["kind"].(string)
	if !ok || kind != "DeploymentConfig" {
		return nil, errs.New("no deployment config returned from endpoint")
	}
	metadata, ok := result["metadata"].(map[interface{}]interface{})
	if !ok {
		return nil, errs.Errorf("metadata missing from deployment config %s", appName)
	}
	// Check the space label is what we expect
	labels, ok := metadata["labels"].(map[interface{}]interface{})
	if !ok {
		return nil, errs.Errorf("labels missing from deployment config %s", appName)
	}
	spaceLabel, ok := labels["space"].(string)
	if !ok || len(spaceLabel) == 0 {
		return nil, errs.Errorf("space label missing from deployment config %s", appName)
	}
	if spaceLabel != space {
		return nil, errs.Errorf("deployment config %s is part of space %s, expected space %s", appName, spaceLabel, space)
	}
	// Get UID from deployment config
	uid, ok := metadata["uid"].(string)
	if !ok || len(uid) == 0 {
		return nil, errs.Errorf("malformed metadata in deployment config %s", appName)
	}
	// Read application version from label
	version := labels["version"].(string)
	if !ok || len(version) == 0 {
		return nil, errs.Errorf("version missing from deployment config %s", appName)
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
		return nil, errs.WithStack(err)
	} else if result == nil {
		return nil, nil
	}
	// Find the current deployment for the DC we just found. This should correspond to the deployment
	// shown in the OpenShift web console's overview page
	rcs, err := kc.getReplicationControllers(namespace, result.dcUID)
	if err != nil {
		return nil, errs.WithStack(err)
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
			newest = rc
		}
	}
	if newest != nil {
		result.current = newest
	}
	return result, nil
}

func (kc *kubeClient) getReplicationControllers(namespace string, dcUID types.UID) ([]v1.ReplicationController, error) {
	rcs, err := kc.ReplicationControllers(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, errs.WithStack(err)
	}

	// Current Kubernetes concept used to represent OpenShift Deployments
	rcsForDc := []v1.ReplicationController{}
	for _, rc := range rcs.Items {

		// Use OwnerReferences to map RC to DC that created it
		match := false
		for _, ref := range rc.OwnerReferences {
			if ref.UID == dcUID && ref.Controller != nil && *ref.Controller {
				match = true
				break
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
		return nil, errs.WithStack(err)
	} else if quota == nil {
		return nil, errs.Errorf("no resource quota with name: %s", computeResources)
	}

	// Convert quantities to floating point, as this should provide enough
	// precision in practice
	cpuLimit, err := quantityToFloat64(quota.Status.Hard[v1.ResourceLimitsCPU])
	if err != nil {
		return nil, errs.WithStack(err)
	}
	cpuUsed, err := quantityToFloat64(quota.Status.Used[v1.ResourceLimitsCPU])
	if err != nil {
		return nil, errs.WithStack(err)
	}

	memLimit, err := quantityToFloat64(quota.Status.Hard[v1.ResourceLimitsMemory])
	if err != nil {
		return nil, errs.WithStack(err)
	}

	memUsed, err := quantityToFloat64(quota.Status.Used[v1.ResourceLimitsMemory])
	if err != nil {
		return nil, errs.WithStack(err)
	}
	memUnits := "bytes"

	result := &app.EnvStats{
		Cpucores: &app.EnvStatCores{
			Quota: &cpuLimit,
			Used:  &cpuUsed,
		},
		Memory: &app.EnvStatMemory{
			Quota: &memLimit,
			Used:  &memUsed,
			Units: &memUnits,
		},
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
			return -1, errs.Errorf("%s cannot be represented as a 64-bit integer", valDec.String())
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
		return nil, errs.WithStack(err)
	}
	return pods.Items, nil
}

func (kc *kubeClient) getPods(namespace string, uid types.UID) ([]*v1.Pod, error) {
	pods, err := kc.Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, errs.WithStack(err)
	}

	appPods := []*v1.Pod{}
	for _, pod := range pods.Items {
		// If a pod belongs to a given RC, it should have an OwnerReference
		// whose UID matches that of the RC
		// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/services/ownerReferences.js#L40
		match := false
		for _, ref := range pod.OwnerReferences {
			if ref.UID == uid && ref.Controller != nil && *ref.Controller {
				match = true
				break
			}
		}
		if match {
			appPods = append(appPods, &pod)
		}
	}

	return appPods, nil
}

// Pod status constants
const (
	podRunning     = "Running"
	podNotReady    = "Not Ready"
	podWarning     = "Warning"
	podError       = "Error"
	podPulling     = "Pulling"
	podPending     = "Pending"
	podSucceeded   = "Succeeded"
	podTerminating = "Terminating"
	podUnknown     = "Unknown"
)

func (kc *kubeClient) getPodStatus(pods []*v1.Pod) ([][]string, int) {
	/*
	 * Use the same categorization used by the web console. See:
	 * https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/podDonut.js
	 * https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/filters/resources.js
	 */
	podStatus := make(map[string]int)
	podTotal := 0
	for _, pod := range pods {
		statusKey := podUnknown
		if pod.Status.Phase == v1.PodFailed {
			// Failed pods are not included, see web console:
			// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/directives/podDonut.js#L32
			continue
		} else if pod.DeletionTimestamp != nil {
			// Terminating pods have a deletionTimeStamp set
			statusKey = podTerminating
		} else if warn, severe := isPodWarning(pod); warn {
			// Check for warnings/errors
			if severe {
				statusKey = podError
			} else {
				statusKey = podWarning
			}
		} else if isPullingImage(pod) {
			// One or more containers is waiting on its image to be pulled
			statusKey = podPulling
		} else if pod.Status.Phase == v1.PodRunning && !isPodReady(pod) {
			// Pod is running, but one or more containers is not yet ready
			statusKey = podNotReady
		} else {
			// Use Kubernetes pod phase
			statusKey = string(pod.Status.Phase)
		}
		podStatus[statusKey]++
		podTotal++
	}

	// if there were actually no pods, create a dummy entry
	if podTotal == 0 {
		podStatus[podRunning] = 0
	}

	var result [][]string
	for status, count := range podStatus {
		statusEntry := []string{status, strconv.Itoa(count)}
		result = append(result, statusEntry)
	}

	return result, podTotal
}

func isPodWarning(pod *v1.Pod) (warning, severe bool) {
	const containerTimeout time.Duration = 5 * time.Minute
	const containerCrashLoop string = "CrashLoopBackOff"
	// Consider Unknown phase a warning state
	if pod.Status.Phase == v1.PodUnknown {
		return true, false
	}

	// Check if pod has been in Pending phase for too long
	now := time.Now()
	if pod.Status.Phase == v1.PodPending {
		duration := now.Sub(pod.CreationTimestamp.Time)
		if duration > containerTimeout {
			return true, false
		}
	}

	// Check for warning conditions in pod's containers
	if pod.Status.Phase == v1.PodRunning {
		for _, status := range pod.Status.ContainerStatuses {
			state := status.State
			// Check if the container terminated with non-zero exit status
			if state.Terminated != nil && state.Terminated.ExitCode != 0 {
				// Severe if pod is terminated, indicating container didn't stop cleanly
				return true, pod.DeletionTimestamp != nil
			}
			// Check if the container has been repeatedly crashing
			if state.Waiting != nil && state.Waiting.Reason == containerCrashLoop {
				return true, true
			}
			// Check if the container has not become ready within timeout
			if state.Running != nil && !status.Ready {
				startTime := state.Running.StartedAt.Time
				duration := now.Sub(startTime)
				if duration > containerTimeout {
					return true, false
				}
			}
		}
	}

	return false, false
}

func isPullingImage(pod *v1.Pod) bool {
	const containerCreating string = "ContainerCreating"
	// If pod is pending with a container waiting due to a "ContainerCreating" event,
	// categorize as "Pulling". This may change as more information is made available.
	// See: https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/filters/resources.js#L663
	if pod.Status.Phase == v1.PodPending {
		for _, status := range pod.Status.ContainerStatuses {
			waiting := status.State.Waiting
			if waiting != nil && waiting.Reason == containerCreating {
				return true
			}
		}
	}
	return false
}

func isPodReady(pod *v1.Pod) bool {
	// If all of the pod's containers have a ready status, then the pod is
	// considered ready.
	total := len(pod.Spec.Containers)
	numReady := 0
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			numReady++
		}
	}
	return numReady == total
}

func (kc *kubeClient) getBestRoute(namespace string, dc *deployment) (*url.URL, error) {
	serviceMap, err := kc.getMatchingServices(namespace, dc)
	if err != nil {
		return nil, err
	}
	// Get routes and associate to services using spec.to.name
	err = kc.getRoutesByService(namespace, serviceMap)
	if err != nil {
		return nil, err
	}

	// Find route with highest score according to heuristics from web-console
	var bestRoute *route
	bestScore := -1
	for _, routes := range serviceMap {
		for _, route := range routes {
			score := scoreRoute(route)
			if score > bestScore {
				bestScore = score
				bestRoute = route
			}
		}
	}

	// Construct URL from best route
	var result *url.URL
	if bestRoute != nil {
		scheme := "http"
		if bestRoute.tls {
			scheme = "https"
		}
		result = &url.URL{
			Scheme: scheme,
			Host:   bestRoute.host,
			Path:   bestRoute.path,
		}
	}

	return result, nil
}

func (kc *kubeClient) getMatchingServices(namespace string, dc *deployment) (routesByService map[string][]*route, err error) {
	services, err := kc.Services(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}
	// Check if each service's selector matches labels in deployment's pod template
	template := dc.current.Spec.Template
	if template == nil {
		return nil, errors.New("No pod template for current deployment")
	}
	routesByService = make(map[string][]*route)
	for _, service := range services.Items {
		selector := service.Spec.Selector
		match := true
		// Treat empty selector as not matching
		if len(selector) == 0 {
			match = false
		}
		for key := range selector {
			if selector[key] != template.Labels[key] {
				match = false
				break
			}
		}
		// If all selector labels match those in the pod template, add service key to map.
		// Routes will be added later by the getRoutesByService method.
		if match {
			routesByService[service.Name] = make([]*route, 0)
		}
	}
	return routesByService, nil
}

func (kc *kubeClient) getRoutesByService(namespace string, routesByService map[string][]*route) error {
	routeURL := fmt.Sprintf("/oapi/v1/namespaces/%s/routes", namespace)
	result, err := kc.getResource(routeURL, false)
	if err != nil {
		return err
	}

	// Parse list of routes
	kind, ok := result["kind"].(string)
	if !ok || kind != "RouteList" {
		return errors.New("No route list returned from endpoint")
	}
	items, ok := result["items"].([]interface{})
	if !ok {
		return errors.New("No list of routes in response")
	}

	for _, item := range items {
		routeItem, ok := item.(map[interface{}]interface{})
		if !ok {
			return errors.New("Route object invalid")
		}

		// Parse route from result
		spec, ok := routeItem["spec"].(map[interface{}]interface{})
		if !ok {
			return errors.New("Spec missing from route")
		}
		// Determine which service this route points to
		to, ok := spec["to"].(map[interface{}]interface{})
		if !ok {
			return errors.New("Route has no destination")
		}
		toName, ok := to["name"].(string)
		if !ok || len(toName) == 0 {
			return errors.New("Service name missing or invalid for route")
		}

		var matchingServices []string
		// Check if this route is for a service we're interested in
		_, pres := routesByService[toName]
		if pres {
			matchingServices = append(matchingServices, toName)
		}

		// Also check alternate backends for services
		altBackends, ok := spec["alternateBackends"].([]interface{})
		if ok {
			for idx := range altBackends {
				backend, ok := altBackends[idx].(map[interface{}]interface{})
				if !ok {
					return errors.New("Malformed alternative backend")
				}
				// Check if this alternate backend is a service we want a route for
				backendKind, err := getOptionalStringValue(backend, "kind")
				if err != nil {
					return err
				}
				if backendKind == "Service" {
					backendName, ok := backend["name"].(string)
					if ok && len(backendName) > 0 {
						_, pres := routesByService[backendName]
						if pres {
							matchingServices = append(matchingServices, backendName)
						}
					}
				}
			}
		}
		if len(matchingServices) > 0 {
			// Get ingress points
			status, ok := routeItem["status"].(map[interface{}]interface{})
			if !ok {
				return errors.New("Status missing from route")
			}
			ingresses, ok := status["ingress"].([]interface{})
			if !ok {
				return errors.New("No ingress array listed in route")
			}

			// Prefer ingress with oldest lastTransitionTime that is marked as admitted
			oldestAdmittedIngress, err := findOldestAdmittedIngress(ingresses)
			if err != nil {
				return err
			}

			// Use hostname from oldest admitted ingress if possible
			var hostname string
			if oldestAdmittedIngress != nil {
				hostname, ok = oldestAdmittedIngress["host"].(string)
				if !ok {
					return errors.New("Hostname missing from ingress")
				}
			} else {
				// Fall back to optional host in spec
				hostname, _ = spec["host"].(string)
			}

			// Check for optional path
			path, err := getOptionalStringValue(spec, "path")
			if err != nil {
				return err
			}

			// Determine whether route uses TLS
			// see: https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/filters/resources.js#L193
			isTLS := false
			tls, ok := spec["tls"].(map[interface{}]interface{})
			if ok {
				tlsTerm, ok := tls["termination"].(string)
				if ok && len(tlsTerm) > 0 {
					isTLS = true
				}
			}

			// Check if this route uses a custom hostname
			customHost := true
			metadata, ok := routeItem["metadata"].(map[interface{}]interface{})
			if ok {
				annotations, ok := metadata["annotations"].(map[interface{}]interface{})
				if ok {
					hostGenerated, err := getOptionalStringValue(annotations, "openshift.io/host.generated")
					if err != nil {
						return err
					}
					if hostGenerated == "true" {
						customHost = false
					}
				}
			}
			route := &route{
				host:                 hostname,
				path:                 path,
				tls:                  isTLS,
				hasAdmitted:          oldestAdmittedIngress != nil,
				hasAlternateBackends: len(altBackends) > 0,
				isCustomHost:         customHost,
			}
			// TODO check wildcard policy? (see above link)
			// Associate this route with any services whoses routes we're looking for
			for _, serviceName := range matchingServices {
				routesByService[serviceName] = append(routesByService[serviceName], route)
			}
		}
	}
	return nil
}

func getOptionalStringValue(respData map[interface{}]interface{}, paramName string) (string, error) {
	val, pres := respData[paramName]
	if !pres {
		return "", nil
	}
	strVal, ok := val.(string)
	if !ok {
		return "", errors.New("Property " + paramName + " is not a string")
	}
	return strVal, nil
}

func findOldestAdmittedIngress(ingresses []interface{}) (ingress map[interface{}]interface{}, err error) {
	var oldestAdmittedIngress map[interface{}]interface{}
	var oldestIngressTime time.Time
	for idx := range ingresses {
		ingress, ok := ingresses[idx].(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("Bad ingress found in route")
		}
		// Check for oldest admitted ingress
		conditions, ok := ingress["conditions"].([]interface{})
		if ok {
			for condIdx := range conditions {
				condition, ok := conditions[condIdx].(map[interface{}]interface{})
				if !ok {
					return nil, errors.New("Bad condition for ingress")
				}
				condType, err := getOptionalStringValue(condition, "type")
				if err != nil {
					return nil, err
				}
				condStatus, err := getOptionalStringValue(condition, "status")
				if err != nil {
					return nil, err
				}
				if condType == "Admitted" && condStatus == "True" {
					lastTransitionStr, ok := condition["lastTransitionTime"].(string)
					if !ok {
						return nil, errors.New("Missing last transition time from ingress condition")
					}
					lastTransition, err := time.Parse(time.RFC3339, lastTransitionStr)
					if err != nil {
						return nil, err
					}
					if oldestAdmittedIngress == nil || lastTransition.Before(oldestIngressTime) {
						oldestAdmittedIngress = ingress
						oldestIngressTime = lastTransition
					}
				}
			}
		}
	}
	return oldestAdmittedIngress, nil
}

func scoreRoute(route *route) int {
	// See: https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/services/routes.js#L106
	score := 0
	if route.hasAdmitted {
		score += 11
	}
	if route.hasAlternateBackends {
		score += 5
	}
	if route.isCustomHost {
		score += 3
	}
	if route.tls {
		score++
	}
	return score
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (kc *kubeClient) getResource(url string, allowMissing bool) (map[interface{}]interface{}, error) {
	var body []byte
	fullURL := strings.TrimSuffix(kc.config.ClusterURL, "/") + url
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("Authorization", "Bearer "+kc.config.BearerToken)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()

	status := resp.StatusCode
	if status == http.StatusNotFound && allowMissing {
		return nil, nil
	} else if httpStatusFailed(status) {
		return nil, errs.Errorf("failed to GET url %s due to status code %d", fullURL, status)
	}
	var respType map[interface{}]interface{}
	err = yaml.Unmarshal(b, &respType)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return respType, nil
}

func httpStatusFailed(status int) bool {
	// if status is not between 200-299 then it's an error
	return status < http.StatusOK || status >= http.StatusMultipleChoices
}
