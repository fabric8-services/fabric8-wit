package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	errs "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
	WatchEventsInNamespace(nameSpace string) (*cache.FIFO, chan struct{})
	GetDeploymentPodQuota(spaceName string, appName string, envName string) (*app.SimpleDeploymentPodLimitRange, error)
	GetSpaceAndOtherEnvironmentUsage(spaceName string) ([]*app.SpaceAndOtherEnvironmentUsage, error)
	Close()
	KubeAccessControl
}

// KubeRESTAPI collects methods that call out to the Kubernetes API server over the network
type KubeRESTAPI interface {
	corev1.CoreV1Interface
}

type kubeClient struct {
	config *KubeClientConfig
	envMap map[string]string
	BaseURLProvider
	KubeRESTAPI
	metricsMap map[string]Metrics
	rulesMap   map[string]*accessRules
	OpenShiftRESTAPI
	MetricsGetter
}

type kubeAPIClient struct {
	corev1.CoreV1Interface
	restConfig *rest.Config
}

// OpenShiftRESTAPI collects methods that call out to the OpenShift API server over the network
type OpenShiftRESTAPI interface {
	GetBuildConfigs(namespace string, labelSelector string) (map[string]interface{}, error)
	DeleteBuildConfig(namespace string, lables map[string]string) (map[string]interface{}, error)
	GetBuilds(namespace string, labelSelector string) (map[string]interface{}, error)
	GetDeploymentConfig(namespace string, name string) (map[string]interface{}, error)
	DeleteDeploymentConfig(namespace string, name string, opts *metaV1.DeleteOptions) (map[string]interface{}, error)
	GetDeploymentConfigScale(namespace string, name string) (map[string]interface{}, error)
	SetDeploymentConfigScale(namespace string, name string, scale map[string]interface{}) (map[string]interface{}, error)
	GetRoutes(namespace string, labelSelector string) (map[string]interface{}, error)
	DeleteRoute(namespace string, name string, opts *metaV1.DeleteOptions) (map[string]interface{}, error)
	CreateSelfSubjectRulesReview(namespace string, review map[string]interface{}) (map[string]interface{}, error)
}

type openShiftAPIClient struct {
	config     *KubeClientConfig
	httpClient *http.Client
}

type deployment struct {
	dcName     string
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
	GetEnvironmentMapping() map[string]string
	CanDeploy(envType string) bool
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

const limitRangeName = "resource-limits"

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
		return nil, err
	}
	osAPI, err := config.GetOpenShiftRESTAPI(config)
	if err != nil {
		return nil, err
	}
	// Use default implementation if no MetricsGetter is specified
	if config.MetricsGetter == nil {
		config.MetricsGetter = &defaultGetter{}
	}

	envMap := config.GetEnvironmentMapping()
	kubeClient := &kubeClient{
		config:           config,
		envMap:           envMap,
		BaseURLProvider:  config,
		KubeRESTAPI:      kubeAPI,
		OpenShiftRESTAPI: osAPI,
		metricsMap:       make(map[string]Metrics),
		rulesMap:         make(map[string]*accessRules),
		MetricsGetter:    config.MetricsGetter,
	}

	return kubeClient, nil
}

func NewOSClient(config *KubeClientConfig) (OpenShiftRESTAPI, error) {
	// Use default implementation if no OpenShiftGetter is specified
	if config.OpenShiftRESTAPIGetter == nil {
		config.OpenShiftRESTAPIGetter = &defaultGetter{}
	}
	osAPI, err := config.GetOpenShiftRESTAPI(config)
	if err != nil {
		return nil, err
	}

	return osAPI, nil
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
		return nil, err
	}

	// Get all applications in this space using BuildConfig names
	apps := []*app.SimpleApp{}
	for _, bc := range buildconfigs {
		appn, err := kc.GetApplication(spaceName, bc)
		if err != nil {
			return nil, err
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
		// Only look for the application in environments where the user can deploy applications
		if kc.CanDeploy(envName) {
			deployment, err := kc.GetDeployment(spaceName, appName, envName)
			if err != nil {
				return nil, err
			} else if deployment != nil {
				deployments = append(deployments, deployment)
			}
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
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
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
		return nil, err
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

	_, err = kc.SetDeploymentConfigScale(envNS, dcName, scale)
	if err != nil {
		return nil, err
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

func (oc *openShiftAPIClient) SetDeploymentConfigScale(namespace string, name string,
	scale map[string]interface{}) (map[string]interface{}, error) {
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
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get all pods created by this deployment
	pods, err := kc.getPods(envNS, deploy.current)
	if err != nil {
		return nil, err
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
		return nil, err
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
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.current)
	if err != nil {
		return nil, err
	}

	mc, err := kc.GetMetricsClient(envNS)
	if err != nil {
		return nil, err
	}

	// Gather the statistics we need about the current deployment
	cpuUsage, err := mc.GetCPUMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
	}
	memoryUsage, err := mc.GetMemoryMetrics(pods, envNS, startTime)
	if err != nil {
		return nil, err
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
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}

	// Get the UID for the current deployment of the app
	deploy, err := kc.getCurrentDeployment(spaceName, appName, envNS)
	if err != nil {
		return nil, err
	} else if deploy == nil || deploy.current == nil {
		return nil, nil
	}

	// Get pods belonging to current deployment
	pods, err := kc.getPods(envNS, deploy.current)
	if err != nil {
		return nil, err
	}

	mc, err := kc.GetMetricsClient(envNS)
	if err != nil {
		return nil, err
	}

	// Get CPU, memory and network metrics for pods in deployment
	cpuMetrics, err := mc.GetCPUMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	memoryMetrics, err := mc.GetMemoryMetricsRange(pods, envNS, startTime, endTime, limit)
	if err != nil {
		return nil, err
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
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return err
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
		log.Error(nil, map[string]interface{}{
			"err":             err,
			"dcName":          dcName,
			"spaceName":       spaceName,
			"applicationName": appName,
			"envName":         envName,
		}, "could not delete routes in deploymentConfig "+dcName)
	}

	// Delete services
	err = kc.deleteServices(dcName, envNS)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":             err,
			"dcName":          dcName,
			"spaceName":       spaceName,
			"applicationName": appName,
			"envName":         envName,
		}, "could not delete services in deploymentConfig "+dcName)
	}

	// Delete DC (will also delete RCs and pods)
	err = kc.deleteDeploymentConfig(spaceName, dcName, envNS)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":             err,
			"dcName":          dcName,
			"spaceName":       spaceName,
			"applicationName": appName,
			"envName":         envName,
		}, "could not delete deploymentConfig "+dcName)
		return err
	}
	return nil
}

// GetEnvironments retrieves information on all environments in the cluster
// for the current user
func (kc *kubeClient) GetEnvironments() ([]*app.SimpleEnvironment, error) {
	envs := []*app.SimpleEnvironment{}
	for envName := range kc.envMap {
		// Only return environments where the user can deploy applications
		if kc.CanDeploy(envName) {
			env, err := kc.GetEnvironment(envName)
			if err != nil {
				return nil, err
			}
			envs = append(envs, env)
		}
	}
	return envs, nil
}

// GetEnvironment returns information on an environment with the provided name
func (kc *kubeClient) GetEnvironment(envName string) (*app.SimpleEnvironment, error) {
	envNS, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}

	envStats, err := kc.getResourceQuota(envNS)
	if err != nil {
		return nil, err
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
		return nil, err
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

func (oc *openShiftAPIClient) DeleteBuildConfig(namespace string, labels map[string]string) (map[string]interface{}, error) {

	if namespace == "" {
		namespace = oc.config.UserNamespace
	}

	var params []string
	for k, v := range labels {
		params = append(params, k+"="+v)
	}

	// The API server rejects deleting buildconfigs by label, so get all
	// buildconfigs with the label, and delete one-by-one
	bcList, err := oc.GetBuildConfigs(namespace, url.QueryEscape(strings.Join(params[:], ",")))
	if err != nil {
		return nil, err
	}

	kind, ok := bcList["kind"].(string)
	if !ok || (kind != "BuildConfigList" && kind != "List") {
		return nil, errs.New("no buildconfig list returned from endpoint")
	}

	bcs, ok := bcList["items"].([]interface{})
	if !ok {
		return nil, errs.New("no list of buildconfig in response")
	}

	response := make(map[string]interface{})

	for _, bc := range bcs {
		name, err := getName(bc)
		if err != nil {
			return nil, err
		}

		opts := getDeleteOption()
		resourceURI := fmt.Sprintf("/oapi/v1/namespaces/%s/buildconfigs/%s", namespace, name)
		resp, err := oc.sendResource(resourceURI, "DELETE", opts)
		if err != nil {
			return nil, err
		}

		response[name] = resp["status"].(interface{})
	}

	return response, nil
}

func getDeleteOption() *metaV1.DeleteOptions {
	policy := metaV1.DeletePropagationForeground
	opts := &metaV1.DeleteOptions{
		TypeMeta: metaV1.TypeMeta{ // Normally set automatically by k8s client-go
			Kind:       "DeleteOptions",
			APIVersion: "v1",
		},
		PropagationPolicy: &policy,
	}
	return opts
}

func getName(r interface{}) (string, error) {
	b, ok := r.(map[string]interface{})
	if !ok {
		return "", errs.New("BuildConfig is not an object")
	}

	metadata, ok := b["metadata"].(map[string]interface{})
	if !ok {
		return "", errs.New("BuildConfig has no metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", errs.New("BuildConfig name is missing")
	}

	return name, nil
}

// getDeployableEnvironmentNamespace finds a namespace with the corresponding environment name.
// Differs from getEnvironmentNamespace in that the environment must be one where the user can deploy
// applications
func (kc *kubeClient) getDeployableEnvironmentNamespace(envName string) (string, error) {
	envNS, pres := kc.envMap[envName]
	if !pres || !kc.CanDeploy(envName) {
		return "", errs.Errorf("unknown environment: %s", envName)
	}
	return envNS, nil
}

// getEnvironmentNamespace finds a namespace with the corresponding environment name
func (kc *kubeClient) getEnvironmentNamespace(envName string) (string, error) {
	envNS, pres := kc.envMap[envName]
	if !pres {
		return "", errs.Errorf("unknown environment: %s", envName)
	}
	return envNS, nil
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (oc *openShiftAPIClient) sendResource(path string, method string, reqBody interface{}) (map[string]interface{}, error) {
	url, err := oc.config.GetAPIURL()
	if err != nil {
		return nil, err
	}
	fullURL := strings.TrimSuffix(*url, "/") + path

	marshalled, err := json.Marshal(reqBody)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not marshal %s request", method)
		return nil, errs.WithStack(err)
	}

	req, err := http.NewRequest(method, fullURL, bytes.NewBuffer(marshalled))
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not create %s request", method)
		return nil, errs.WithStack(err)
	}

	token, err := oc.config.GetAPIToken()
	if err != nil {
		return nil, err
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
		return nil, errs.WithStack(err)
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"url":          fullURL,
			"request_body": reqBody,
		}, "could not read response from %s request", method)
		return nil, errs.WithStack(err)
	}
	respBody := buf.Bytes()

	status := resp.StatusCode
	if status < http.StatusOK || status > http.StatusPartialContent {
		log.Error(nil, map[string]interface{}{
			"url":           fullURL,
			"request_body":  reqBody,
			"response_body": buf,
			"http_status":   status,
		}, "failed to %s request due to HTTP error", method)

		// If response contains a Kubernetes Status object, create a StatusError
		err = parseErrorFromStatus(respBody)
		if err != nil {
			return nil, convertError(errs.WithStack(err), "failed to %s url %s due to status code %d", method, fullURL, status)
		}
		return nil, errs.Errorf("failed to %s url %s: status code %d", method, fullURL, status)
	}

	var respJSON map[string]interface{}
	err = json.Unmarshal(respBody, &respJSON)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"url":           fullURL,
			"response_body": buf,
			"http_status":   status,
		}, "error unmarshalling JSON response")
		return nil, errs.WithStack(err)
	}
	return respJSON, nil
}

func (kc *kubeClient) getAndParseDeploymentConfig(namespace string, dcName string, space string) (*deployment, error) {
	result, err := kc.GetDeploymentConfig(namespace, dcName)
	if err != nil {
		return nil, err
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

	/* FIXME The launcher no longer configures POM files to apply a space label as part of the
	 * project's fabric8-maven-plugin resource goal. This results in OpenShift objects that have no
	 * space label.
	 *
	 * Additionally, projects created using the old launcher may have old space label
	 * configuration that is not updated when importing the project with the new launcher.
	 * This results in OpenShift objects having a space label for the wrong space.
	 *
	 * Until OpenShift objects have reliable space labels, we work around the issue by logging
	 * a warning and waiving the space label check.
	 *
	 * See: https://github.com/openshiftio/openshift.io/issues/2360 */
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
		log.Warn(nil, map[string]interface{}{
			"namespace":   namespace,
			"dc_name":     dcName,
			"space_name":  space,
			"space_label": spaceLabel,
		}, "space label on deployment config indicates different space")
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
		dcName:     dcName,
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

	// Fall back to application name, if we can't find a name in the annotations
	result := appName
	// Look for latest build that contains desired annotation
	var latestCreationTime time.Time
	for _, item := range items {
		build, ok := item.(map[string]interface{})
		if !ok {
			return "", errs.New("malformed build object")
		}
		metadata, ok := build["metadata"].(map[string]interface{})
		if !ok {
			return "", errs.New("metadata missing from build object")
		}
		creationTimeStr, ok := metadata["creationTimestamp"].(string)
		if ok {
			creationTime, err := time.Parse(time.RFC3339, creationTimeStr)
			if err != nil {
				return "", errs.Wrapf(err, "build creation time uses an invalid date")
			}
			if creationTime.After(latestCreationTime) {
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
								latestCreationTime = creationTime
							}
						}
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
		return errors.NewNotFoundErrorFromString(fmt.Sprintf("deployment config %s does not exist in %s", dcName, namespace))
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
	// API states this should return a Status object, but it returns the DC instead,
	// just check for no HTTP error
	_, err = kc.DeleteDeploymentConfig(namespace, dcName, opts)
	if err != nil {
		return err
	}
	return nil
}

func (oc *openShiftAPIClient) DeleteDeploymentConfig(namespace string, name string,
	opts *metaV1.DeleteOptions) (map[string]interface{}, error) {
	dcPath := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, name)
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
		return nil, err
	} else if result == nil {
		return nil, nil
	}
	// Find the current deployment for the DC we just found. This should correspond to the deployment
	// shown in the OpenShift web console's overview page
	rcs, err := kc.getReplicationControllers(namespace, result)
	if err != nil {
		return nil, err
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

func (kc *kubeClient) getReplicationControllers(namespace string, deploy *deployment) ([]v1.ReplicationController, error) {
	rcs, err := kc.ReplicationControllers(namespace).List(metaV1.ListOptions{})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":               err,
			"namespace":         namespace,
			"deployment_config": deploy.dcName,
		}, "failed to list replication controllers")
		return nil, convertError(errs.WithStack(err), "failed to list replication controllers in %s", namespace)
	}

	// Current Kubernetes concept used to represent OpenShift Deployments
	rcsForDc := []v1.ReplicationController{}
	for _, rc := range rcs.Items {

		// Use OwnerReferences to map RC to DC that created it
		match := false
		for _, ref := range rc.OwnerReferences {
			if ref.UID == deploy.dcUID && ref.Controller != nil && *ref.Controller {
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
	// Get both resource quotas in one API call
	const computeResources string = "compute-resources"
	const objectCounts string = "object-counts"
	quotas, err := kc.ResourceQuotas(namespace).List(metaV1.ListOptions{})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":       err,
			"namespace": namespace,
		}, "failed to list resource quotas")
		return nil, convertError(errs.WithStack(err), "failed to list resource quotas from %s",
			namespace)
	}

	var computeQuota, objectQuota *v1.ResourceQuota
	for idx := range quotas.Items {
		quota := &quotas.Items[idx]
		if quota.Name == computeResources {
			computeQuota = quota
		} else if quota.Name == objectCounts {
			objectQuota = quota
		}
	}

	if computeQuota == nil {
		log.Error(nil, map[string]interface{}{
			"namespace":  namespace,
			"quota_name": computeResources,
		}, "resource quota not found")
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("resource quota '%s' not found in %s",
			computeResources, namespace))
	} else if objectQuota == nil {
		log.Error(nil, map[string]interface{}{
			"namespace":  namespace,
			"quota_name": objectCounts,
		}, "resource quota not found")
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("resource quota '%s' not found in %s",
			objectCounts, namespace))
	}

	result := &app.EnvStats{}

	// Collect compute-based resource usage and quotas
	cpuQuota, err := getEnvStatQuota(computeQuota, v1.ResourceLimitsCPU)
	if err != nil {
		return nil, err
	} else if cpuQuota == nil {
		log.Error(nil, map[string]interface{}{
			"namespace":     namespace,
			"quota_name":    computeResources,
			"resource_name": v1.ResourceLimitsCPU,
		}, "CPU resource not found in quota")
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("CPU missing from resource quota in %s",
			namespace))
	}
	result.Cpucores = cpuQuota

	memQuota, err := getEnvStatQuota(computeQuota, v1.ResourceLimitsMemory)
	if err != nil {
		return nil, err
	} else if memQuota == nil {
		log.Error(nil, map[string]interface{}{
			"namespace":     namespace,
			"quota_name":    computeResources,
			"resource_name": v1.ResourceLimitsMemory,
		}, "memory resource not found in quota")
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("memory missing from resource quota in %s",
			namespace))
	}
	result.Memory = memQuota

	// Get object-based resource usage and quotas where they exist
	objStats, err := getEnvStatQuota(objectQuota, v1.ResourcePods)
	if err != nil {
		return nil, err
	}
	result.Pods = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourceReplicationControllers)
	if err != nil {
		return nil, err
	}
	result.ReplicationControllers = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourceQuotas)
	if err != nil {
		return nil, err
	}
	result.ResourceQuotas = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourceServices)
	if err != nil {
		return nil, err
	}
	result.Services = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourceSecrets)
	if err != nil {
		return nil, err
	}
	result.Secrets = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourceConfigMaps)
	if err != nil {
		return nil, err
	}
	result.ConfigMaps = objStats
	objStats, err = getEnvStatQuota(objectQuota, v1.ResourcePersistentVolumeClaims)
	if err != nil {
		return nil, err
	}
	result.PersistentVolumeClaims = objStats
	// OpenShift-specific object type
	const resourceImageStreams v1.ResourceName = "openshift.io/imagestreams"
	objStats, err = getEnvStatQuota(objectQuota, resourceImageStreams)
	if err != nil {
		return nil, err
	}
	result.ImageStreams = objStats

	return result, nil
}

func getEnvStatQuota(quota *v1.ResourceQuota, resourceName v1.ResourceName) (*app.EnvStatQuota, error) {
	var result *app.EnvStatQuota
	used, limit, err := getResourceUsageAndLimit(quota, resourceName)
	if err != nil {
		return nil, err
	} else if used != nil && limit != nil {
		result = &app.EnvStatQuota{
			Quota: limit,
			Used:  used,
		}
	}
	return result, nil
}

func getResourceUsageAndLimit(quota *v1.ResourceQuota, resourceName v1.ResourceName) (used, limit *float64, err error) {
	// Return nil if no resource by that name is present
	quantity, pres := quota.Status.Hard[resourceName]
	if pres {
		// Convert quantities to floating point, as this should provide enough
		// precision in practice
		limitVal, err := quantityToFloat64(quantity)
		if err != nil {
			return nil, nil, err
		}
		limit = &limitVal
	}

	// Do the same for usage
	quantity, pres = quota.Status.Used[resourceName]
	if pres {
		usedVal, err := quantityToFloat64(quantity)
		if err != nil {
			return nil, nil, err
		}
		used = &usedVal
	}

	return used, limit, nil
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

func (kc *kubeClient) getPods(namespace string, rc *v1.ReplicationController) ([]*v1.Pod, error) {
	pods, err := kc.Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":                    err,
			"namespace":              namespace,
			"replication_controller": rc.Name,
		}, "failed to list pods")
		return nil, convertError(errs.WithStack(err), "failed to list pods in %s", namespace)
	}

	appPods := []*v1.Pod{}
	for idx, pod := range pods.Items {
		// If a pod belongs to a given RC, it should have an OwnerReference
		// whose UID matches that of the RC
		// https://github.com/openshift/origin-web-console/blob/v3.7.0/app/scripts/services/ownerReferences.js#L40
		match := false
		for _, ref := range pod.OwnerReferences {
			if ref.UID == rc.UID && ref.Controller != nil && *ref.Controller {
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
				return nil, err
			}
			mem, err := quantityToFloat64(*container.Resources.Limits.Memory())
			if err != nil {
				return nil, err
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

func (kc *kubeClient) GetDeploymentPodQuota(spaceName string, appName string, envName string) (*app.SimpleDeploymentPodLimitRange, error) {
	// Find namespace for environment name
	namespace, err := kc.getDeployableEnvironmentNamespace(envName)
	if err != nil {
		return nil, err
	}
	dcName, err := kc.getDeploymentConfigNameForApp(namespace, appName, spaceName)
	if err != nil {
		return nil, errs.Errorf("could not retrieve deployment with the given namespace %s, app name %s and space name %s", namespace, appName, spaceName)
	}

	deploymentConfig, err := kc.GetDeploymentConfig(namespace, dcName)
	if err != nil {
		return nil, errs.Errorf("could not retrieve deployment config with name %s for namespace %s", dcName, namespace)
	} else if deploymentConfig == nil {
		return nil, errors.NewNotFoundErrorFromString(fmt.Sprintf("no deployment config found named %s in %s", dcName, namespace))
	}

	spec, ok := deploymentConfig["spec"].(map[string]interface{})
	if !ok {
		return nil, errs.Errorf("spec is missing from deployment config %s: %+v", dcName, spec)
	}

	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, errs.Errorf("template is missing from deployment config %s: %+v", dcName, template)
	}

	innerSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil, errs.Errorf("inner spec is missing from deployment config %s: %+v", dcName, innerSpec)
	}

	// This should be checked, to see if maps[string]interface is appropriate type for iterable arr
	containers, ok := innerSpec["containers"].([]interface{})
	if !ok {
		return nil, errs.Errorf("containers is missing from deployment config %s: %+v", dcName, containers)
	}

	numContainersMissingCPU := float64(0)
	numContainersMissingMem := float64(0)
	podCPULimit := float64(0)
	podMemLimit := float64(0)

	for _, containerItem := range containers {
		container, ok := containerItem.(map[string]interface{})
		if !ok {
			return nil, errs.Errorf("containers array contains invalid container: %v", containerItem)
		}
		resourcesItem, pres := container["resources"]
		if !pres {
			numContainersMissingCPU++
			numContainersMissingMem++
			continue
		}
		resources, ok := resourcesItem.(map[string]interface{})
		if !ok {
			return nil, errs.Errorf("resources spec in pod template is invalid: %v", resourcesItem)
		}
		if len(resources) == 0 {
			numContainersMissingCPU++
			numContainersMissingMem++
			continue
		}

		limits, ok := resources["limits"].(map[string]interface{})
		if !ok {
			return nil, errs.Errorf("limits is missing from deployment config %s: %+v", dcName, resources)
		}

		cpuLimit, err := getOptionalStringValue(limits, "cpucores")
		if err != nil {
			return nil, err
		}
		if len(cpuLimit) == 0 {
			numContainersMissingCPU++
		} else {
			cpuQuantity, err := resource.ParseQuantity(cpuLimit)
			if err != nil {
				return nil, errs.Errorf("could not parse cpu quantity for %+v of %s ", container, dcName)
			}

			cpuValue, err := quantityToFloat64(cpuQuantity)
			if err != nil {
				return nil, errs.Errorf("could not convert cpu quantity %+v to float64 value", cpuQuantity)
			}
			podCPULimit += cpuValue
		}

		memLimit, err := getOptionalStringValue(limits, "memory")
		if err != nil {
			return nil, err
		}
		if len(memLimit) == 0 {
			numContainersMissingMem++
		} else {
			memoryQuantity, err := resource.ParseQuantity(memLimit)
			if err != nil {
				return nil, errs.Errorf("could not parse memory quantity for %+v of %s ", container, dcName)
			}

			memoryValue, err := quantityToFloat64(memoryQuantity)
			if err != nil {
				return nil, errs.Errorf("could not convert memory quantity %+v to float64 value", memoryQuantity)
			}

			podMemLimit += memoryValue
		}
	}

	if numContainersMissingCPU > 0 || numContainersMissingMem > 0 {
		// Look up default resource limits using LimitRanges API
		limitRange, err := kc.LimitRanges(namespace).Get(limitRangeName, metaV1.GetOptions{})
		if err != nil {
			log.Error(nil, map[string]interface{}{
				"err":              err,
				"namespace":        namespace,
				"limit_range_name": limitRangeName,
			}, "failed to get limit range")
			return nil, convertError(errs.WithStack(err), "failed to get limit range %s in %s", limitRangeName, namespace)
		}

		var containerCPULimit, containerMemLimit *resource.Quantity
		for _, limit := range limitRange.Spec.Limits {
			if limit.Type == "Container" {
				cpuQty, pres := limit.Default[v1.ResourceCPU]
				if pres {
					containerCPULimit = &cpuQty
				}
				memQty, pres := limit.Default[v1.ResourceMemory]
				if pres {
					containerMemLimit = &memQty
				}
			}
		}

		if containerCPULimit == nil || containerMemLimit == nil {
			log.Error(nil, map[string]interface{}{
				"limit_range": limitRange,
				"namespace":   namespace,
			}, "CPU or memory container limit missing from LimitRange")
			return nil, errs.Errorf("CPU or memory container limit missing from LimitRange for namespace %s", namespace)
		}

		defaultCPULimit, err := quantityToFloat64(*containerCPULimit)
		if err != nil {
			return nil, errs.Errorf("could not convert cpu quantity %+v to float64 value", *containerCPULimit)
		}
		defaultMemLimit, err := quantityToFloat64(*containerMemLimit)
		if err != nil {
			return nil, errs.Errorf("could not convert memory quantity %+v to float64 value", *containerMemLimit)
		}

		// Apply default limit for each container that didn't specify CPU/memory limits
		podCPULimit += defaultCPULimit * numContainersMissingCPU
		podMemLimit += defaultMemLimit * numContainersMissingMem
	}

	return &app.SimpleDeploymentPodLimitRange{
		Limits: &app.PodsQuota{
			Cpucores: &podCPULimit,
			Memory:   &podMemLimit,
		},
	}, nil
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

func (kc *kubeClient) getMatchingServices(namespace string, deploy *deployment) (routesByService map[string][]*route, err error) {
	services, err := kc.Services(namespace).List(metaV1.ListOptions{})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":               err,
			"namespace":         namespace,
			"deployment_config": deploy.dcName,
		}, "failed to list services")
		return nil, convertError(errs.WithStack(err), "failed to list services in %s", namespace)
	}
	// Check if each service's selector matches labels in deployment's pod template
	template := deploy.current.Spec.Template
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
		return err
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

func (kc *kubeClient) deleteServices(appLabel string, envNS string) error {
	// Delete all dependent objects before deleting the service
	policy := metaV1.DeletePropagationForeground
	delOpts := &metaV1.DeleteOptions{
		PropagationPolicy: &policy,
	}
	// Delete all services in namespace with matching 'app' label
	listOpts := metaV1.ListOptions{
		LabelSelector: "app=" + appLabel,
	}
	// The API server rejects deleting services by label, so get all
	// services with the label, and delete one-by-one
	services, err := kc.Services(envNS).List(listOpts)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":       err,
			"namespace": envNS,
			"app_label": appLabel,
		}, "failed to list services")
		return convertError(errs.WithStack(err), "failed to list services in %s", envNS)
	}
	for _, service := range services.Items {
		err = kc.Services(envNS).Delete(service.Name, delOpts)
		if err != nil {
			log.Error(nil, map[string]interface{}{
				"err":          err,
				"namespace":    envNS,
				"app_label":    appLabel,
				"service_name": service.Name,
			}, "failed to delete service")
			return convertError(errs.WithStack(err), "failed to delete service '%s' in %s",
				service.Name, envNS)
		}
	}
	return nil
}

func (kc *kubeClient) deleteRoutes(appLabel string, envNS string) error {
	// Delete all routes in namespace with matching 'app' label
	escapedSelector := url.QueryEscape("app=" + appLabel)

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
		_, err := kc.DeleteRoute(envNS, name, opts)
		if err != nil {
			return err
		}
	}
	return nil
}

func (oc *openShiftAPIClient) DeleteRoute(namespace string, name string,
	opts *metaV1.DeleteOptions) (map[string]interface{}, error) {
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
	} else if status < http.StatusOK || status > http.StatusPartialContent {
		log.Error(nil, map[string]interface{}{
			"url":           fullURL,
			"response_body": buf,
			"http_status":   status,
		}, "error returned from HTTP request")

		// If response contains a Kubernetes Status object, create a StatusError
		err = parseErrorFromStatus(b)
		if err != nil {
			return nil, convertError(errs.WithStack(err), "failed to GET url %s due to status code %d", fullURL, status)
		}
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

func parseErrorFromStatus(body []byte) error {
	// Try unmarshalling as Status resource
	var kubeStatus metaV1.Status
	err := json.Unmarshal(body, &kubeStatus)
	if err != nil || kubeStatus.Kind != "Status" {
		return nil
	}
	return kubeErrors.FromObject(&kubeStatus)
}

// convertError converts a Kubernetes API error into an error suitable to be
// passed to jsonapi.ErrorToJSONAPIError. The format and args arguments are used
// to construct an error message, in a similar fashion to fmt.Sprintf.
func convertError(err error, format string, args ...interface{}) error {
	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}

	cause := errs.Cause(err)
	if statusError, ok := cause.(*kubeErrors.StatusError); ok {
		message = fmt.Sprintf("%s: %s", message, statusError.Error())
		// Pass through certain HTTP statuses to our API response
		if kubeErrors.IsBadRequest(statusError) {
			return errors.NewBadParameterErrorFromString(message)
		} else if kubeErrors.IsNotFound(statusError) {
			return errors.NewNotFoundErrorFromString(message)
		}
	}
	return errors.NewInternalError(nil /* unused */, errs.Wrap(err, message))
}

func (kc *kubeClient) WatchEventsInNamespace(nameSpace string) (*cache.FIFO, chan struct{}) {
	eventsLW := &cache.ListWatch{
		ListFunc: func(options metaV1.ListOptions) (runtime.Object, error) {
			return kc.Events(nameSpace).List(options)
		},
		WatchFunc: func(options metaV1.ListOptions) (watch.Interface, error) {
			return kc.Events(nameSpace).Watch(options)
		},
	}

	store := cache.NewFIFO(cache.MetaNamespaceKeyFunc)
	ref := cache.NewReflector(eventsLW, &v1.Event{}, store, 0)
	stopCh := make(chan struct{})

	ref.RunUntil(stopCh)

	return store, stopCh
}

func (kc *kubeClient) GetSpaceAndOtherEnvironmentUsage(spaceName string) ([]*app.SpaceAndOtherEnvironmentUsage, error) {

	space, err := kc.GetSpace(spaceName)
	if err != nil {
		return nil, err
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return nil, err
	}

	result := make([]*app.SpaceAndOtherEnvironmentUsage, 0, len(envs))
	envMap := calculateEnvMap(space)
	for _, env := range envs {
		if env.Attributes.Name != nil && env.Attributes.Quota != nil {
			envName := *env.Attributes.Name
			otherUsage := *env.Attributes.Quota

			spaceUsage, pres := envMap[envName]
			if pres {
				// Subtract usage by this space from total environment usage to determine
				// usage by all other spaces combined
				*otherUsage.Cpucores.Used -= *spaceUsage.Cpucores
				*otherUsage.Memory.Used -= *spaceUsage.Memory
			} else {
				cpuUsage := float64(0)
				memUsage := float64(0)
				spaceUsage = &app.SpaceEnvironmentUsageQuota{
					Cpucores: &cpuUsage,
					Memory:   &memUsage,
				}
			}

			usage := &app.SpaceAndOtherEnvironmentUsage{
				Attributes: &app.SpaceAndOtherEnvironmentUsageAttributes{
					Name:       &envName,
					SpaceUsage: spaceUsage,
					OtherUsage: &otherUsage,
				},
				ID:   envName,
				Type: "environment",
			}

			result = append(result, usage)
		}
	}

	return result, nil
}

func calculateEnvMap(space *app.SimpleSpace) map[string]*app.SpaceEnvironmentUsageQuota {
	envMap := make(map[string]*app.SpaceEnvironmentUsageQuota)
	for _, appl := range space.Attributes.Applications {
		for _, dep := range appl.Attributes.Deployments {
			if value, ok := envMap[dep.Attributes.Name]; ok {
				*value.Cpucores += *dep.Attributes.PodsQuota.Cpucores
				*value.Memory += *dep.Attributes.PodsQuota.Memory
			} else {
				cpucores := *dep.Attributes.PodsQuota.Cpucores
				memory := *dep.Attributes.PodsQuota.Memory
				envMap[dep.Attributes.Name] = &app.SpaceEnvironmentUsageQuota{
					Cpucores: &cpucores,
					Memory:   &memory,
				}
			}
		}
	}

	return envMap
}
