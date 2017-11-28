package controller

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	inf "gopkg.in/inf.v0"
	yaml "gopkg.in/yaml.v2"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	kubernetes "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	rest "k8s.io/client-go/rest"

	"github.com/fabric8-services/fabric8-wit/app"
)

// KubeClient contains configuration and methods for interacting with Kubernetes cluster
type KubeClient struct {
	config        *rest.Config
	clientset     *kubernetes.Clientset
	userNamespace string
	envMap        map[string]string
}

// NewKubeClient creates a KubeClient given a URL to the Kubernetes cluster, an authorized token to
// access the cluster, and the namespace in that cluster of type 'user'
func NewKubeClient(clusterURL string, kubeToken string, userNamespace string) (*KubeClient, error) {
	config := rest.Config{
		Host:        clusterURL,
		BearerToken: kubeToken,
	}
	clientset, err := kubernetes.NewForConfig(&config)
	if err != nil {
		return nil, err
	}

	kubeClient := new(KubeClient)
	kubeClient.config = &config
	kubeClient.clientset = clientset
	kubeClient.userNamespace = userNamespace

	// Get environments from config map
	envMap, err := kubeClient.getEnvironmentsFromConfigMap()
	if err != nil {
		return nil, err
	}
	kubeClient.envMap = envMap
	return kubeClient, nil
}

// GetSpace returns a space matching the provided name, containing all applications that belong to it
func (kc *KubeClient) GetSpace(spaceName string) (*app.SimpleSpace, error) {
	// Get BuildConfigs within the user namespace that have a matching 'space' label
	// This is similar to how pipelines are displayed in fabric8-ui
	// https://github.com/fabric8-ui/fabric8-ui/blob/master/src/app/space/create/pipelines/pipelines.component.ts
	buildconfigs, err := kc.getBuildConfigs(spaceName)
	if err != nil {
		return nil, err
	}

	// Get all applications in this space using BuildConfig names
	var apps []*app.SimpleApp
	for _, bc := range buildconfigs {
		appn, err := kc.GetApplication(spaceName, bc)
		if err != nil {
			return nil, err
		}
		apps = append(apps, appn)
	}

	result := &app.SimpleSpace{
		Applications: apps, // TODO UUID
	}

	return result, nil
}

// GetApplication retrieves an application with the given space and application names, with the status
// of that application's deployment in each environment
func (kc *KubeClient) GetApplication(spaceName string, appName string) (*app.SimpleApp, error) {
	// Get all deployments of this app for each environment in this space
	var deployments []*app.SimpleDeployment
	for envName := range kc.envMap {
		deployment, err := kc.GetDeployment(spaceName, appName, envName)
		if err != nil {
			return nil, err
		} else if deployment != nil {
			deployments = append(deployments, deployment)
		}
	}

	result := &app.SimpleApp{
		Name:     &appName, // TODO UUID
		Pipeline: deployments,
	}
	return result, nil
}

// GetDeployment returns information about the current deployment of an application within a
// particular environment. The application must exist within the provided space.
func (kc *KubeClient) GetDeployment(spaceName string, appName string, envName string) (*app.SimpleDeployment, error) {
	envNS, pres := kc.envMap[envName]
	if !pres {
		return nil, errors.New("Unknown environment: " + envName)
	}
	// Look up DeploymentConfig corresponding to the application name in the provided environment
	dc, err := kc.getDeploymentConfig(envNS, appName, spaceName)
	if err != nil {
		return nil, err
	} else if len(dc) == 0 {
		return nil, nil
	}
	// Find the current deployment for the DC we just found. This should correspond to the deployment
	// shown in the OpenShift web console's overview page
	rc, err := kc.getCurrentDeployment(envNS, dc)
	if err != nil {
		return nil, err
	} else if len(rc) == 0 {
		return nil, nil
	}
	// Gather the statistics we need about the current deployment
	envStats, err := kc.getDeploymentEnvStats(envNS, rc)
	if err != nil {
		return nil, err
	}

	result := &app.SimpleDeployment{ // TODO UUID, add version
		Name:  &envName,
		Stats: envStats,
	}
	return result, nil
}

// GetEnvironments retrieves information on all environments in the cluster
// for the current user
func (kc *KubeClient) GetEnvironments() ([]*app.SimpleEnvironment, error) {
	var envs []*app.SimpleEnvironment
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
func (kc *KubeClient) GetEnvironment(envName string) (*app.SimpleEnvironment, error) {
	envNS, pres := kc.envMap[envName]
	if !pres {
		return nil, errors.New("Unknown environment: " + envName)
	}

	envStats, err := kc.getResourceQuota(envNS)
	if err != nil {
		return nil, err
	}

	env := &app.SimpleEnvironment{
		Name:  &envName, // TODO UUID
		Quota: envStats,
	}
	return env, nil
}

func (kc *KubeClient) getDeploymentEnvStats(envNS string, rc types.UID) (*app.EnvStats, error) {
	// Get all pods created by this deployment
	pods, err := kc.getPods(envNS, rc)
	if err != nil {
		return nil, err
	}
	// Get the status of each pod in the deployment
	podStats, err := kc.getPodStatus(pods)
	if err != nil {
		return nil, err
	}

	result := &app.EnvStats{
		Cpucores: &app.EnvStatCores{},  // TODO
		Memory:   &app.EnvStatMemory{}, // TODO
		Pods:     podStats,
	}
	return result, nil
}

func (kc *KubeClient) getBuildConfigs(space string) ([]string, error) {
	// BuildConfigs are OpenShift objects, so access REST API using HTTP directly until
	// there is a Go client for OpenShift
	const spaceLabel string = "space"
	// BuildConfigs created by fabric8 have a "space" label indicating the space they belong to
	queryParam := url.QueryEscape(spaceLabel + "=" + space)
	bcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/buildconfigs?labelSelector=%s", kc.userNamespace, queryParam)
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
	var buildconfigs []string
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

func (kc *KubeClient) getEnvironmentsFromConfigMap() (map[string]string, error) {
	// fabric8 creates a ConfigMap in the user namespace with information on environments
	const envConfigMap string = "fabric8-environments"
	const providerLabel string = "fabric8"
	configmap, err := kc.clientset.CoreV1().ConfigMaps(kc.userNamespace).Get(envConfigMap, metaV1.GetOptions{})
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

func (kc *KubeClient) getDeploymentConfig(namespace string, appName string, space string) (types.UID, error) {
	dcURL := fmt.Sprintf("/oapi/v1/namespaces/%s/deploymentconfigs/%s", namespace, appName)
	result, err := kc.getResource(dcURL, true)
	if err != nil {
		return "", err
	} else if result == nil {
		return "", nil
	}
	// Parse deployment config from result
	kind, ok := result["kind"].(string)
	if !ok || kind != "DeploymentConfig" {
		return "", errors.New("No deployment config returned from endpoint")
	}
	metadata, ok := result["metadata"].(map[interface{}]interface{})
	if !ok {
		return "", errors.New("Metadata missing from deployment config")
	}
	// Check the space label is what we expect
	labels, ok := metadata["labels"].(map[interface{}]interface{})
	if !ok {
		return "", errors.New("Labels missing from deployment config")
	}
	spaceLabel, ok := labels["space"].(string)
	if !ok || len(spaceLabel) == 0 {
		return "", errors.New("Space label missing from deployment config")
	}
	if spaceLabel != space {
		return "", errors.New("Deployment config " + appName + " is part of space " +
			spaceLabel + ", expected space " + space)
	}
	// Get UID from deployment config
	uid, ok := metadata["uid"].(string)
	if !ok || len(uid) == 0 {
		return "", errors.New("Malformed metadata in deployment config")
	}
	return types.UID(uid), nil
}

func (kc *KubeClient) getCurrentDeployment(namespace string, dcUID types.UID) (types.UID, error) {
	rcs, err := kc.getReplicationControllers(namespace, dcUID)
	if err != nil {
		return "", err
	} else if len(rcs) == 0 {
		return "", nil
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
	// No visible RCs
	if newest == nil {
		return "", nil
	}
	return newest.UID, nil
}

func (kc *KubeClient) getReplicationControllers(namespace string, dcUID types.UID) ([]v1.ReplicationController, error) {
	rcs, err := kc.clientset.CoreV1().ReplicationControllers(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Current Kubernetes concept used to represent OpenShift Deployments
	var rcsForDc []v1.ReplicationController
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

func (kc *KubeClient) getResourceQuota(namespace string) (*app.EnvStats, error) {
	const computeResources string = "compute-resources"
	quota, err := kc.clientset.CoreV1().ResourceQuotas(namespace).Get(computeResources, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	} else if quota == nil {
		return nil, errors.New("No resource quota with name: " + computeResources)
	}

	// TODO Need to figure out how to express these quantities.
	// - Keep fixed point rep
	// -- Send as string value (see Quantity.String)
	// -- Send as string mantissa and int exponent (see Quantity.AsCanonicalBytes)
	// - Convert to floating point
	cpuLimitRes := quota.Status.Hard[v1.ResourceLimitsCPU]
	cpuLimit, err := int64ToInt32(cpuLimitRes.MilliValue())
	if err != nil {
		return nil, err
	}
	cpuUsedRes := quota.Status.Used[v1.ResourceLimitsCPU]
	cpuUsed, err := int64ToInt32(cpuUsedRes.MilliValue())
	if err != nil {
		return nil, err
	}

	cpuStats := &app.EnvStatCores{
		Quota: &cpuLimit,
		Used:  &cpuUsed,
	}

	memLimit, err := quantityToInt32(quota.Status.Hard[v1.ResourceLimitsMemory])
	if err != nil {
		return nil, err
	}

	memUsed, err := quantityToInt32(quota.Status.Used[v1.ResourceLimitsMemory])
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

// FIXME temporary function til we figure out how to express quantities
func quantityToInt32(q resource.Quantity) (int, error) {
	val64, rc := q.AsInt64()
	var val32 int
	var err error
	if rc {
		val32, err = int64ToInt32(val64)
		if err != nil {
			return -1, err
		}
	} else {
		valDec := q.AsDec()
		val32, err = decToInt32(valDec)
		if err != nil {
			return -1, err
		}
	}
	return val32, nil
}

// FIXME temporary function til we figure out how to express quantities
func int64ToInt32(num int64) (int, error) {
	if num > math.MaxInt32 || num < math.MinInt32 {
		return -1, errors.New(string(num) + " cannot be represented as 32-bit integer")
	}
	return int(num), nil
}

// FIXME temporary function til we figure out how to express quantities
func decToInt32(dec *inf.Dec) (int, error) {
	val64, ok := dec.Unscaled()
	if !ok {
		return -1, errors.New(dec.String() + " cannot be represented as 64-bit integer")
	}
	return int64ToInt32(val64)
}

func (kc *KubeClient) getPods(namespace string, uid types.UID) ([]v1.Pod, error) {
	pods, err := kc.clientset.CoreV1().Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var appPods []v1.Pod
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

func (kc *KubeClient) getPodStatus(pods []v1.Pod) (*app.EnvStatPods, error) {
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

	result := &app.EnvStatPods{
		Starting: &starting,
		Running:  &running,
		Stopping: &stopping,
	}

	return result, nil
}

// Derived from: https://github.com/fabric8-services/fabric8-tenant/blob/master/openshift/kube_token.go
func (kc *KubeClient) getResource(url string, allowMissing bool) (map[interface{}]interface{}, error) {
	var body []byte
	fullURL := strings.TrimSuffix(kc.config.Host, "/") + url
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
