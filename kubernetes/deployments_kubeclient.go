package kubernetes

import (
	"bytes"
	"encoding/json"
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
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
	rest "k8s.io/client-go/rest"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/log"
	errs "github.com/pkg/errors"
)

// KubeClientConfig holds configuration data needed to create a new KubeClientInterface
// with kubernetes.NewKubeClient
type KubeClientConfig struct {
	// Provides URLS for all APIs, and also access tokens
	BaseURLProvider
	// Kubernetes namespace in the cluster of type 'user'
	UserNamespace string
	// Timeout used for communicating with Kubernetes and OpenShift API servers,
	// a value of zero indicates no timeout
	Timeout time.Duration
	// Specifies a non-default HTTP transport to use when sending requests to
	// Kubernetes and OpenShift API servers
	Transport http.RoundTripper
	// Provides access to the Kubernetes REST API, uses default implementation if not set
	KubeRESTAPIGetter
	// Provides access to the metrics API, uses default implementation if not set
	MetricsGetter
	// Provides access to the OpenShift REST API, uses default implementation if not set
	OpenShiftRESTAPIGetter
}

// KubeRESTAPIGetter has a method to access the KubeRESTAPI interface
type KubeRESTAPIGetter interface {
	GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error)
}

// OpenShiftRESTAPIGetter has a method to access the OpenShiftRESTAPI interface
type OpenShiftRESTAPIGetter interface {
	GetOpenShiftRESTAPI(config *KubeClientConfig) (OpenShiftRESTAPI, error)
}

// MetricsGetter has a method to access the Metrics interface
type MetricsGetter interface {
	GetMetrics(config *MetricsClientConfig) (Metrics, error)
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
	DeleteDeployment(spaceName string, appName string, envName string) error
	GetEnvironments() ([]*app.SimpleEnvironment, error)
	GetEnvironment(envName string) (*app.SimpleEnvironment, error)
	GetMetricsClient(envNS string) (Metrics, error)
	Close()
}

type kubeClient struct {
	config *KubeClientConfig
	envMap map[string]string
	BaseURLProvider
	KubeRESTAPI
	metricsMap map[string]Metrics
	OpenShiftRESTAPI
	MetricsGetter
}

// KubeRESTAPI collects methods that call out to the Kubernetes API server over the network
type KubeRESTAPI interface {
	corev1.CoreV1Interface
}

type kubeAPIClient struct {
	corev1.CoreV1Interface
	restConfig *rest.Config
}

// OpenShiftRESTAPI collects methods that call out to the OpenShift API server over the network
type OpenShiftRESTAPI interface {
	GetBuildConfigs(namespace string, labelSelector string) (map[string]interface{}, error)
	GetBuilds(namespace string, labelSelector string) (map[string]interface{}, error)
	GetDeploymentConfig(namespace string, name string) (map[string]interface{}, error)
	DeleteDeploymentConfig(namespace string, name string, opts *metaV1.DeleteOptions) error
	GetDeploymentConfigScale(namespace string, name string) (map[string]interface{}, error)
	SetDeploymentConfigScale(namespace string, name string, scale map[string]interface{}) error
	GetRoutes(namespace string, labelSelector string) (map[string]interface{}, error)
	DeleteRoute(namespace string, name string, opts *metaV1.DeleteOptions) error
}

type openShiftAPIClient struct {
	config     *KubeClientConfig
	httpClient *http.Client
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

// BaseURLProvider provides the BASE URL (minimal path) of several APIs used in Deployments.
// For true multicluster support, every API in this inteface should take an environment namespace name.
// This hasn't been done, because the rest of fabric8 seems to assume the cluster is the same.
// For most uses, the proxy server will hide this issue - but not for metrics/logging and console.
type BaseURLProvider interface {
	GetAPIURL() (*string, error)
	GetMetricsURL(envNS string) (*string, error)
	GetConsoleURL(envNS string) (*string, error)
	GetLoggingURL(envNS string, deploymentName string) (*string, error)

	GetAPIToken() (*string, error)
	GetMetricsToken(envNS string) (*string, error)
}

// ensure kubeClient implements KubeClientInterface
var _ KubeClientInterface = &kubeClient{}
var _ KubeClientInterface = (*kubeClient)(nil)

// Receiver for default implementation of KubeRESTAPIGetter and MetricsGetter
type defaultGetter struct{}

// NewKubeClient creates a KubeClientInterface given a configuration. The returned
// KubeClientInterface must be closed using the Close method, when no longer needed.
func NewKubeClient(config *KubeClientConfig) (KubeClientInterface, error) {
	// Use default implementation if no KubernetesGetter is specified
	if config.KubeRESTAPIGetter == nil {
		config.KubeRESTAPIGetter = &defaultGetter{}
	}
	// Use default implementation if no OpenShiftGetter is specified
	if config.OpenShiftRESTAPIGetter == nil {
		config.OpenShiftRESTAPIGetter = &defaultGetter{}
	}
	kubeAPI, err := config.GetKubeRESTAPI(config)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	osAPI, err := config.GetOpenShiftRESTAPI(config)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Use default implementation if no MetricsGetter is specified
	if config.MetricsGetter == nil {
		config.MetricsGetter = &defaultGetter{}
	}
	// Get environments from config map
	envMap, err := getEnvironmentsFromConfigMap(kubeAPI, config.UserNamespace)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	kubeClient := &kubeClient{
		config:           config,
		envMap:           envMap,
		BaseURLProvider:  config,
		KubeRESTAPI:      kubeAPI,
		OpenShiftRESTAPI: osAPI,
		metricsMap:       make(map[string]Metrics),
		MetricsGetter:    config.MetricsGetter,
	}

	return kubeClient, nil
}

func (*defaultGetter) GetKubeRESTAPI(config *KubeClientConfig) (KubeRESTAPI, error) {
	url, err := config.GetAPIURL()
	if err != nil {
		return nil, err
	}
	token, err := config.GetAPIToken()
	if err != nil {
		return nil, err
	}
	restConfig := &rest.Config{
		Host:        *url,
		BearerToken: *token,
		Timeout:     config.Timeout,
		Transport:   config.Transport,
	}
	coreV1Client, err := corev1.NewForConfig(restConfig)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	client := &kubeAPIClient{
		CoreV1Interface: coreV1Client,
		restConfig:      restConfig,
	}
	return client, nil
}

func (*defaultGetter) GetOpenShiftRESTAPI(config *KubeClientConfig) (OpenShiftRESTAPI, error) {
	// Equivalent to http.DefaultClient with added timeout and transport
	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: config.Transport,
	}
	client := &openShiftAPIClient{
		config:     config,
		httpClient: httpClient,
	}
	return client, nil
}

func (*defaultGetter) GetMetrics(config *MetricsClientConfig) (Metrics, error) {
	return NewMetricsClient(config)
}

func (kc *kubeClient) GetMetricsClient(envNS string) (Metrics, error) {

	if kc.metricsMap[envNS] != nil {
		return kc.metricsMap[envNS], nil
	}

	url, err := kc.GetMetricsURL(envNS)
	if err != nil {
		return nil, err
	}
	token, err := kc.GetMetricsToken(envNS)
	if err != nil {
		return nil, err
	}

	metricsConfig := &MetricsClientConfig{
		MetricsURL:  *url,
		BearerToken: *token,
	}

	metrics, err := kc.GetMetrics(metricsConfig)
	if err != nil {
		return nil, err
	}
	kc.metricsMap[envNS] = metrics
	return metrics, nil
}

// Close releases any resources held by this KubeClientInterface
func (kc *kubeClient) Close() {
	// Metrics client needs to be closed to stop Hawkular go-routine from spinning
	for _, m := range kc.metricsMap {
		m.Close()
	}
}

// GetSpace returns a space matching the provided name, containing all applications that belong to it
func (kc *kubeClient) GetSpace(spaceName string) (*app.SimpleSpace, error) {
	// Get BuildConfigs within the user namespace that have a matching 'space' label
	// This is similar to how pipelines are displayed in fabric8-ui
	// https://github.com/fabric8-ui/fabric8-ui/blob/master/src/app/space/create/pipelines/pipelines.component.ts
	buildconfigs, err := kc.getBuildConfigsForSpace(spaceName)
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

	// Deployment Config name does not always match the application name, look up
	// DC name using available metadata
	dcName, err := kc.getDeploymentConfigNameForApp(envNS, appName, spaceName)
	if err != nil {
		return nil, err
	}

	// Look up the Scale for the DeploymentConfig corresponding to the application name in the provided environment
	scale, err := kc.GetDeploymentConfigScale(envNS, dcName)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	spec, ok := scale["spec"].(map[string]interface{})
	if !ok {
		log.Error(nil, map[string]interface{}{
			"err":              err,
			"space_name":       spaceName,
			"application_name": appName,
			"environment_name": envName,
		}, "invalid deployment config returned from endpoint")
		return nil, errs.New("invalid deployment config returned from endpoint: missing 'spec'")
	}

	replicas, pres := spec["replicas"]
	oldReplicas := 0 // replicas property may be missing from spec if set to 0
	if pres {
		oldReplicasFlt, ok := replicas.(float64)
		if !ok {
			return nil, errs.New("invalid deployment config returned from endpoint: 'replicas' is not a number")
		}
		oldReplicas = int(oldReplicasFlt)
	}
	spec["replicas"] = deployNumber

	err = kc.SetDeploymentConfigScale(envNS, dcName, scale)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	log.Info(nil, map[string]interface{}{
		"space_name":        spaceName,
		"application_name":  appName,
		"environment_name":  envName,
		"old_replica_count": oldReplicas,
		"new_replica_count": deployNumber,
	}, "scaled deployment to %d replicas", deployNumber)

	return &oldReplicas, nil
}

func (oc *openShiftAPIClient) GetDeploymentConfigScale(namespace string, name string) (map[string]interface{}, error) {
	dcScalePath := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s/scale", namespace, name)
	return oc.getResource(dcScalePath, false)
}

func (oc *openShiftAPIClient) SetDeploymentConfigScale(namespace string, name string, scale map[string]interface{}) error {
	dcScalePath := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s/scale", namespace, name)
	return oc.sendResource(dcScalePath, "PUT", scale)
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

	// Get the quota for all pods in the deployment
	podsQuota, err := kc.getPodsQuota(pods)
	if err != nil {
		return nil, err
	}

	// Get the status of each pod in the deployment
	podStats, total := kc.getPodStatus(pods)

	// Get related URLs for the deployment
	appURL, err := kc.getApplicationURL(envNS, deploy)
	if err != nil {
		return nil, err
	}

	consoleURL, err := kc.GetConsoleURL(envNS)
	if err != nil {
		return nil, err
	}

	logURL, err := kc.GetLoggingURL(envNS, deploy.current.Name)
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
			Name:      envName,
			Version:   &verString,
			Pods:      podStats,
			PodTotal:  &total,
			PodsQuota: podsQuota,
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

	mc, err := kc.GetMetricsClient(envNS)
	if err != nil {
		return nil, err
	}

	// Gather the statistics we need about the current deployment
	cpuUsage, err := mc.GetCPUMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	memoryUsage, err := mc.GetMemoryMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	netTxUsage, err := mc.GetNetworkSentMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}
	netRxUsage, err := mc.GetNetworkRecvMetrics(pods, envNS, startTime)
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

	mc, err := kc.GetMetricsClient(envNS)
	if err != nil {
		return nil, err
	}

	// Get CPU, memory and network metrics for pods in deployment
	cpuMetrics, err := mc.GetCPUMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	memoryMetrics, err := mc.GetMemoryMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	netTxMetrics, err := mc.GetNetworkSentMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	netRxMetrics, err := mc.GetNetworkRecvMetricsRange(pods, envNS, startTime, endTime, limit)
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

func (kc *kubeClient) DeleteDeployment(spaceName string, appName string, envName string) error {
	envNS, err := kc.getEnvironmentNamespace(envName)
	if err != nil {
		return errs.WithStack(err)
	}

	// Deployment Config name does not always match the application name, look up
	// DC name using available metadata
	dcName, err := kc.getDeploymentConfigNameForApp(envNS, appName, spaceName)
	if err != nil {
		return err
	}

	// Delete routes
	err = kc.deleteRoutes(dcName, envNS)
	if err != nil {
		return err
	}
	// Delete services
	err = kc.deleteServices(dcName, envNS)
	if err != nil {
		return err
	}
	// Delete DC (will also delete RCs and pods)
	err = kc.deleteDeploymentConfig(spaceName, dcName, envNS)
	if err != nil {
		return err
	}
	return nil
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

const spaceLabelName = "space"

func (kc *kubeClient) getBuildConfigsForSpace(space string) ([]string, error) {
	// BuildConfigs are OpenShift objects, so access REST API using HTTP directly until
	// there is a Go client for OpenShift

	// BuildConfigs created by fabric8 have a "space" label indicating the space they belong to
	escapedSelector := url.QueryEscape(spaceLabelName + "=" + space)
	result, err := kc.GetBuildConfigs(kc.config.UserNamespace, escapedSelector)
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
		bc, ok := item.(map[string]interface{})
		if !ok {
			return nil, errs.New("malformed build config")
		}
		metadata, ok := bc["metadata"].(map[string]interface{})
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

func (oc *openShiftAPIClient) GetBuildConfigs(namespace string, labelSelector string) (map[string]interface{}, error) {
	bcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/buildconfigs?labelSelector=%s", namespace, labelSelector)
	return oc.getResource(bcURL, false)
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
		return nil, errs.Errorf("unknown or missing provider %s for environments config map in namespace %s", providerLabel, userNamespace)
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
func (oc *openShiftAPIClient) sendResource(path string, method string, reqBody interface{}) error {
	url, err := oc.config.GetAPIURL()
	if err != nil {
		return err
	}
	fullURL := strings.TrimSuffix(*url, "/") + path

	marshalled, err := json.Marshal(reqBody)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not marshall %s request", method)
		return errs.WithStack(err)
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewBuffer(marshalled))
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not create %s request", method)
		return errs.WithStack(err)
	}

	token, err := oc.config.GetAPIToken()
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+*token)

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not perform %s request", method)
		return errs.WithStack(err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"url":           fullURL,
			"request_body":  reqBody,
			"response_body": respBody,
		}, "could not read response from %s request", method)
		return errs.WithStack(err)
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"url":           fullURL,
			"request_body":  reqBody,
			"response_body": respBody,
			"http_status":   status,
		}, "failed to %s request due to HTTP error", method)
		return errs.Errorf("failed to %s url %s: status code %d", method, fullURL, status)
	}
	return nil
}

func (kc *kubeClient) getAndParseDeploymentConfig(namespace string, dcName string, space string) (*deployment, error) {
	result, err := kc.GetDeploymentConfig(namespace, dcName)
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
	metadata, ok := result["metadata"].(map[string]interface{})
	if !ok {
		return nil, errs.Errorf("metadata missing from deployment config %s: %+v", dcName, result)
	}
	// Check the space label is what we expect
	labels, ok := metadata["labels"].(map[string]interface{})
	if !ok {
		return nil, errs.Errorf("labels missing from deployment config %s: %+v", dcName, metadata)
	}
	/* FIXME Not all projects will have the space label defined due to the requirement that
	 * fabric8-maven-plugin is called from the project's POM and not that of its parent.
	 * This requirement is not always satisfied. For now, we work around the issue by logging
	 * a warning and waiving the space label check, if missing.
	 */
	spaceLabel, err := getOptionalStringValue(labels, spaceLabelName)
	if err != nil {
		return nil, err
	}
	if len(spaceLabel) == 0 {
		log.Warn(nil, map[string]interface{}{
			"namespace": namespace,
			"dc_name":   dcName,
			"space":     space,
		}, "space label missing from deployment config")
	} else if spaceLabel != space {
		return nil, errs.Errorf("deployment config %s is part of space %s, expected space %s", dcName, spaceLabel, space)
	}
	// Get UID from deployment config
	uid, ok := metadata["uid"].(string)
	if !ok || len(uid) == 0 {
		return nil, errs.Errorf("malformed metadata in deployment config %s: %+v", dcName, metadata)
	}
	// Read application version from label
	version := labels["version"].(string)
	if !ok || len(version) == 0 {
		return nil, errs.Errorf("version missing from deployment config %s: %+v", dcName, metadata)
	}

	dc := &deployment{
		dcUID:      types.UID(uid),
		appVersion: version,
	}
	return dc, nil
}

func (oc *openShiftAPIClient) GetDeploymentConfig(namespace string, name string) (map[string]interface{}, error) {
	dcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, name)
	return oc.getResource(dcURL, true)
}

const buildConfigLabelName = "openshift.io/build-config.name"
const envServicesAnnotationPrefix = "environment.services.fabric8.io"
const envServicesDeploymentVersions = "deploymentVersions"

func (kc *kubeClient) getDeploymentConfigNameForApp(namespace string, appName string, spaceName string) (string, error) {
	// Look up builds with config name matching appName, this will be the one created by the launcher
	labelSelector := fmt.Sprintf("%s=%s,%s=%s", buildConfigLabelName, appName, spaceLabelName, spaceName)
	escapedSelector := url.QueryEscape(labelSelector)

	// Builds are located in the user's namespace of type "user"
	resp, err := kc.GetBuilds(kc.config.UserNamespace, escapedSelector)
	if err != nil {
		return "", err
	}

	// Parse builds from response
	kind, ok := resp["kind"].(string)
	if !ok || kind != "BuildList" {
		return "", errs.New("no builds returned from endpoint")
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		return "", errs.New("malformed response from endpoint")
	}

	// Look for a matching deployment config within latest completed build
	var latestCompletedBuild map[string]interface{}
	var latestCompletionTime time.Time
	for _, item := range items {
		build, ok := item.(map[string]interface{})
		if !ok {
			return "", errs.New("malformed build object")
		}
		status, ok := build["status"].(map[string]interface{})
		if !ok {
			return "", errs.New("status missing from build object")
		}
		phase, ok := status["phase"].(string)
		if ok && phase == "Complete" {
			completionTimeStr, ok := status["completionTimestamp"].(string)
			if ok {
				completionTime, err := time.Parse(time.RFC3339, completionTimeStr)
				if err != nil {
					return "", errs.Wrapf(err, "build completion time uses an invalid date")
				}
				if completionTime.After(latestCompletionTime) {
					latestCompletedBuild = build
					latestCompletionTime = completionTime
				}
			}
		}
	}

	// Fall back to application name, if we can't find a name in the annotations
	result := appName
	if latestCompletedBuild != nil {
		metadata, ok := latestCompletedBuild["metadata"].(map[string]interface{})
		if !ok {
			return "", errs.New("metadata missing from build object")
		}
		annotations, ok := metadata["annotations"].(map[string]interface{})
		if ok {
			envAnnotationName := fmt.Sprintf(envServicesAnnotationPrefix+"/%s", namespace)
			envServices, pres := annotations[envAnnotationName]
			if pres {
				envServicesStr, ok := envServices.(string)
				if !ok {
					log.Warn(nil, map[string]interface{}{
						"namespace":   namespace,
						"appName":     appName,
						"spaceName":   spaceName,
						"envServices": envServicesStr,
					}, "%s annotation does not contain a string", envServicesAnnotationPrefix)
				} else {
					dcName, err := getNameFromEnvServices([]byte(envServicesStr))
					if err != nil {
						log.Warn(nil, map[string]interface{}{
							"err":         err,
							"namespace":   namespace,
							"appName":     appName,
							"spaceName":   spaceName,
							"envServices": envServicesStr,
						}, "failed to determine Deployment Config name")
					} else if len(dcName) > 0 {
						result = dcName
					}
				}
			}
		}
	}
	return result, nil
}

func getNameFromEnvServices(envServices []byte) (string, error) {
	// Parse YAML annotation value
	var envServicesYaml map[interface{}]interface{}
	err := yaml.Unmarshal(envServices, &envServicesYaml)
	if err != nil {
		return "", errs.Wrapf(err, "failed to unmarshal %s YAML", envServicesAnnotationPrefix)
	}

	// Look for deployment versions
	deployVersionsYaml, pres := envServicesYaml[envServicesDeploymentVersions]
	if pres {
		deployVersions, ok := deployVersionsYaml.(map[interface{}]interface{})
		if ok {
			// TODO If there is more than one entry in deploymentVersions, we just
			// take the first one. What scenario could cause this to occur, and
			// could we handle it better?
			for nameYaml := range deployVersions {
				depName, ok := nameYaml.(string)
				if !ok {
					return "", errs.Errorf("%s does not contain a string", envServicesDeploymentVersions)
				}
				return depName, nil
			}
		}
	}
	return "", nil
}

func (oc *openShiftAPIClient) GetBuilds(namespace string, labelSelector string) (map[string]interface{}, error) {
	bcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/builds?labelSelector=%s", namespace, labelSelector)
	return oc.getResource(bcURL, false)
}

func (kc *kubeClient) deleteDeploymentConfig(spaceName string, dcName string, namespace string) error {
	// Check that the deployment config exists and belongs to the expected space
	dc, err := kc.getAndParseDeploymentConfig(namespace, dcName, spaceName)
	if err != nil {
		return err
	} else if dc == nil {
		return errs.Errorf("deployment config %s does not exist in %s", dcName, namespace)
	}

	// Delete all dependent objects and then this DC
	policy := metaV1.DeletePropagationForeground
	opts := &metaV1.DeleteOptions{
		TypeMeta: metaV1.TypeMeta{ // Normally set automatically by k8s client-go
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
		PropagationPolicy: &policy,
	}
	err = kc.DeleteDeploymentConfig(namespace, dcName, opts)
	if err != nil {
		return err
	}
	return nil
}

func (oc *openShiftAPIClient) DeleteDeploymentConfig(namespace string, name string, opts *metaV1.DeleteOptions) error {
	dcPath := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, name)
	// API states this should return a Status object, but it returns the DC instead,
	// just check for no HTTP error
	return oc.sendResource(dcPath, "DELETE", opts)
}

const deploymentPhaseAnnotation string = "openshift.io/deployment.phase"
const deploymentVersionAnnotation string = "openshift.io/deployment-config.latest-version"

func (kc *kubeClient) getCurrentDeployment(space string, appName string, namespace string) (*deployment, error) {
	// Deployment Config name does not always match the application name, look up
	// DC name using available metadata
	dcName, err := kc.getDeploymentConfigNameForApp(namespace, appName, space)
	if err != nil {
		return nil, err
	}

	// Look up DeploymentConfig corresponding to the application name in the provided environment
	result, err := kc.getAndParseDeploymentConfig(namespace, dcName, space)
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
	// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/controllers/overview.js#L679
	candidates := make(map[string]*v1.ReplicationController)
	// Also consider most recent successful deployment, even if scaled down (not visible)
	var active *v1.ReplicationController
	for idx := range rcs {
		rc := &rcs[idx]
		phase := rc.Annotations[deploymentPhaseAnnotation]
		if phase == "Complete" && (active == nil ||
			active.CreationTimestamp.Before(rc.CreationTimestamp)) {
			active = rc
		}
		if isReplicationControllerVisible(rc) {
			candidates[rc.Name] = rc
		}
	}
	if active != nil {
		candidates[active.Name] = active
	}
	// For final comparison use deployment version annotation instead of creation timestamp
	current, err := getMostRecentByDeploymentVersion(candidates)
	if err != nil {
		return nil, err
	}
	result.current = current
	return result, nil
}

func isReplicationControllerVisible(rc *v1.ReplicationController) bool {
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
	return visible
}

func getMostRecentByDeploymentVersion(rcs map[string]*v1.ReplicationController) (*v1.ReplicationController, error) {
	var result *v1.ReplicationController
	var newestVersion *int64

	for _, rc := range rcs {
		var version *int64
		versionStr, pres := rc.Annotations[deploymentVersionAnnotation]
		if pres {
			versionNum, err := strconv.ParseInt(versionStr, 10, 64)
			if err != nil {
				return nil, errs.Wrapf(err, "deployment version for %s is not a valid integer", rc.Name)
			}
			version = &versionNum
		}

		// Take first RC unconditionally
		if result == nil {
			result = rc
			newestVersion = version
		} else if newestVersion == nil {
			// Prioritize RC with version over those without
			if version != nil {
				result = rc
				newestVersion = version
			} else {
				// Have neither current version nor newest version so far
				// Compare RC names lexicographically as done by web console:
				// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/services/deployments.js#L393
				if rc.Name > result.Name {
					result = rc
				}
			}
		} else if version != nil {
			// Both current RC and newest RC have versions, so compare as integers
			if *version > *newestVersion {
				result = rc
				newestVersion = version
			}
		}
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

func (kc *kubeClient) getPods(namespace string, uid types.UID) ([]*v1.Pod, error) {
	pods, err := kc.Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, errs.WithStack(err)
	}

	appPods := []*v1.Pod{}
	for idx, pod := range pods.Items {
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
			appPods = append(appPods, &pods.Items[idx])
		}
	}

	return appPods, nil
}

func (kc *kubeClient) getPodsQuota(pods []*v1.Pod) (*app.PodsQuota, error) {
	cores := float64(0)
	memory := float64(0)

	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			cpu, err := quantityToFloat64(*container.Resources.Limits.Cpu())
			if err != nil {
				return nil, errs.WithStack(err)
			}
			mem, err := quantityToFloat64(*container.Resources.Limits.Memory())
			if err != nil {
				return nil, errs.WithStack(err)
			}
			cores += cpu
			memory += mem
		}
	}

	result := &app.PodsQuota{
		Cpucores: &cores,
		Memory:   &memory,
	}

	return result, nil
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

	result := [][]string{}
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
		return nil, errs.Errorf("no pod template for current deployment in namespace %s", namespace)
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
	result, err := kc.GetRoutes(namespace, "")
	if err != nil {
		return errs.WithStack(err)
	}

	items, err := getRoutesFromRouteList(result)
	if err != nil {
		return err
	}
	for _, item := range items {
		routeItem, ok := item.(map[string]interface{})
		if !ok {
			log.Error(nil, map[string]interface{}{
				"err":       err,
				"namespace": namespace,
				"response":  result,
			}, "route object invalid")
			return errs.Errorf("invalid route object returned from %s", namespace)
		}

		// Parse route from result
		spec, ok := routeItem["spec"].(map[string]interface{})
		if !ok {
			log.Error(nil, map[string]interface{}{
				"err":       err,
				"namespace": namespace,
				"response":  result,
			}, "spec missing from route")
			return errs.Errorf("spec missing from route returned from %s", namespace)
		}
		// Determine which service this route points to
		to, ok := spec["to"].(map[string]interface{})
		if !ok {
			log.Error(nil, map[string]interface{}{
				"err":       err,
				"namespace": namespace,
				"response":  result,
			}, "route has no destination")
			return errs.Errorf("no destination in route returned from %s", namespace)
		}
		toName, ok := to["name"].(string)
		if !ok || len(toName) == 0 {
			log.Error(nil, map[string]interface{}{
				"err":       err,
				"namespace": namespace,
				"response":  result,
			}, "service name missing or invalid for route")
			return errs.Errorf("service name missing or invalid for route returned from %s", namespace)
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
				backend, ok := altBackends[idx].(map[string]interface{})
				if !ok {
					log.Error(nil, map[string]interface{}{
						"err":       err,
						"namespace": namespace,
						"response":  result,
					}, "malformed alternative backend")
					return errs.Errorf("malformed alternative backend in route returned from %s", namespace)
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
			status, ok := routeItem["status"].(map[string]interface{})
			if !ok {
				log.Error(nil, map[string]interface{}{
					"err":       err,
					"namespace": namespace,
					"response":  result,
				}, "status missing from route")
				return errs.Errorf("status missing from route returned from %s", namespace)
			}
			ingresses, ok := status["ingress"].([]interface{})
			if !ok {
				log.Error(nil, map[string]interface{}{
					"err":       err,
					"namespace": namespace,
					"response":  result,
				}, "no ingress array listed in route")
				return errs.Errorf("no ingress array listed in route returned from %s", namespace)
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
					log.Error(nil, map[string]interface{}{
						"err":       err,
						"namespace": namespace,
						"response":  result,
					}, "hostname missing from ingress")
					return errs.Errorf("hostname missing from ingress in route returned from %s", namespace)
				}
			} else {
				// Fall back to optional host in spec
				hostname, err = getOptionalStringValue(spec, "host")
				if err != nil {
					log.Error(nil, map[string]interface{}{
						"err":       err,
						"namespace": namespace,
						"response":  result,
					}, "invalid hostname in route spec")
					return errs.Wrapf(err, "invalid hostname in route spec returned from %s", namespace)
				}
			}

			// Check for optional path
			path, err := getOptionalStringValue(spec, "path")
			if err != nil {
				return err
			}

			// Determine whether route uses TLS
			// see: https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/filters/resources.js#L193
			isTLS := false
			tls, ok := spec["tls"].(map[string]interface{})
			if ok {
				tlsTerm, ok := tls["termination"].(string)
				if ok && len(tlsTerm) > 0 {
					isTLS = true
				}
			}

			// Check if this route uses a custom hostname
			customHost := true
			metadata, ok := routeItem["metadata"].(map[string]interface{})
			if ok {
				annotations, ok := metadata["annotations"].(map[string]interface{})
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

func (oc *openShiftAPIClient) GetRoutes(namespace string, labelSelector string) (map[string]interface{}, error) {
	var routeURL string
	if len(labelSelector) > 0 {
		routeURL = fmt.Sprintf("/oapi/v1/namespaces/%s/routes?labelSelector=%s", namespace, labelSelector)
	} else {
		routeURL = fmt.Sprintf("/oapi/v1/namespaces/%s/routes", namespace)
	}
	return oc.getResource(routeURL, false)
}

func getRoutesFromRouteList(list map[string]interface{}) ([]interface{}, error) {
	// Parse list of routes
	kind, ok := list["kind"].(string)
	if !ok || kind != "RouteList" {
		return nil, errs.New("No route list returned from endpoint")
	}
	items, ok := list["items"].([]interface{})
	if !ok {
		return nil, errs.New("No list of routes in response")
	}
	return items, nil
}

func getOptionalStringValue(respData map[string]interface{}, paramName string) (string, error) {
	val, pres := respData[paramName]
	if !pres {
		return "", nil
	}
	strVal, ok := val.(string)
	if !ok {
		return "", errs.Errorf("property %s is not a string", paramName)
	}
	return strVal, nil
}

func findOldestAdmittedIngress(ingresses []interface{}) (ingress map[string]interface{}, err error) {
	var oldestAdmittedIngress map[string]interface{}
	var oldestIngressTime time.Time
	for idx := range ingresses {
		ingress, ok := ingresses[idx].(map[string]interface{})
		if !ok {
			return nil, errs.New("bad ingress found in route")
		}
		// Check for oldest admitted ingress
		conditions, ok := ingress["conditions"].([]interface{})
		if ok {
			for condIdx := range conditions {
				condition, ok := conditions[condIdx].(map[string]interface{})
				if !ok {
					return nil, errs.New("bad condition for ingress")
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
						return nil, errs.New("missing last transition time from ingress condition")
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

func (kc *kubeClient) deleteServices(appName string, envNS string) error {
	// Delete all dependent objects before deleting the service
	policy := metaV1.DeletePropagationForeground
	delOpts := &metaV1.DeleteOptions{
		PropagationPolicy: &policy,
	}
	// Delete all services in namespace with matching 'app' label
	listOpts := metaV1.ListOptions{
		LabelSelector: "app=" + appName,
	}
	// The API server rejects deleting services by label, so get all
	// services with the label, and delete one-by-one
	services, err := kc.Services(envNS).List(listOpts)
	if err != nil {
		return errs.WithStack(err)
	}
	for _, service := range services.Items {
		err = kc.Services(envNS).Delete(service.Name, delOpts)
		if err != nil {
			return errs.WithStack(err)
		}
	}
	return nil
}

func (kc *kubeClient) deleteRoutes(appName string, envNS string) error {
	// Delete all routes in namespace with matching 'app' label
	escapedSelector := url.QueryEscape("app=" + appName)

	// Delete all dependent objects before deleting the route
	policy := metaV1.DeletePropagationForeground
	opts := &metaV1.DeleteOptions{
		TypeMeta: metaV1.TypeMeta{ // Normally set automatically by k8s client-go
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
		PropagationPolicy: &policy,
	}

	// The API server rejects deleting services by label, so get all
	// services with the label, and delete one-by-one
	routeList, err := kc.GetRoutes(envNS, escapedSelector)
	if err != nil {
		return err
	}
	routeItems, err := getRoutesFromRouteList(routeList)
	if err != nil {
		return err
	}
	for _, routeItem := range routeItems {
		route, ok := routeItem.(map[string]interface{})
		if !ok {
			return errs.New("Route is not an object")
		}
		metadata, ok := route["metadata"].(map[string]interface{})
		if !ok {
			return errs.New("Route has no metadata")
		}
		name, ok := metadata["name"].(string)
		if !ok {
			return errs.New("Route name is missing")
		}

		// API states this should return a Status object, but it returns the route instead,
		// just check for no HTTP error
		err := kc.DeleteRoute(envNS, name, opts)
		if err != nil {
			return err
		}
	}
	return nil
}

func (oc *openShiftAPIClient) DeleteRoute(namespace string, name string, opts *metaV1.DeleteOptions) error {
	routesPath := fmt.Sprintf("/oapi/v1/namespaces/%s/routes/%s", namespace, name)
	// API states this should return a Status object, but it returns the route instead,
	// just check for no HTTP error
	return oc.sendResource(routesPath, "DELETE", opts)
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (oc *openShiftAPIClient) getResource(path string, allowMissing bool) (map[string]interface{}, error) {

	url, err := oc.config.GetAPIURL()
	if err != nil {
		return nil, err
	}
	var body []byte
	fullURL := strings.TrimSuffix(*url, "/") + path
	req, err := http.NewRequest("GET", fullURL, bytes.NewReader(body))
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
			"url": fullURL,
		}, "error creating HTTP GET request")
		return nil, errs.WithStack(err)
	}

	token, err := oc.config.GetAPIToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+*token)

	resp, err := oc.httpClient.Do(req)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
			"url": fullURL,
		}, "error during HTTP request")
		return nil, errs.WithStack(err)
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()

	status := resp.StatusCode
	if status == http.StatusNotFound && allowMissing {
		return nil, nil
	} else if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"url":           fullURL,
			"response_body": buf,
			"http_status":   status,
		}, "error returned from HTTP request")
		return nil, errs.Errorf("failed to GET url %s due to status code %d", fullURL, status)
	}
	var respType map[string]interface{}
	err = json.Unmarshal(b, &respType)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"url":           fullURL,
			"response_body": buf,
			"http_status":   status,
		}, "error unmarshalling JSON response")
		return nil, errs.WithStack(err)
	}
	return respType, nil
}
