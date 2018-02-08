package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/kubernetesV1"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	witerrors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AppsController implements the apps resource.
type AppsController struct {
	*goa.Controller
	Config *configuration.Registry
	KubeClientGetterV1
}

// KubeClientGetterV1 creates an instance of KubeClientInterface
type KubeClientGetterV1 interface {
	GetKubeClientV1(ctx context.Context) (kubernetesV1.KubeClientInterface, error)
}

// Default implementation of KubeClientGetter used by NewAppsController
type defaultKubeClientGetterV1 struct {
	config *configuration.Registry
}

// NewAppsController creates a apps controller.
func NewAppsController(service *goa.Service, config *configuration.Registry) *AppsController {
	return &AppsController{
		Controller: service.NewController("AppsController"),
		Config:     config,
		KubeClientGetterV1: &defaultKubeClientGetterV1{
			config: config,
		},
	}
}

func getAndCheckOSIOClientV1(ctx context.Context) *OSIOClientV1 {

	// defaults
	host := "localhost"
	scheme := "https"

	req := goa.ContextRequest(ctx)
	if req != nil {
		// Note - it's probably more efficient to force a loopback host, and only use the port number here
		// (on some systems using a non-loopback interface forces a network stack traverse)
		host = req.Host
		scheme = req.URL.Scheme
	}

	if os.Getenv("FABRIC8_WIT_API_URL") != "" {
		witurl, err := url.Parse(os.Getenv("FABRIC8_WIT_API_URL"))
		if err != nil {
			log.Warn(ctx, nil, "Cannot parse FABRIC8_WIT_API_URL; assuming localhost")
		}
		host = witurl.Host
		scheme = witurl.Scheme
	}

	oc := NewOSIOClientV1(ctx, scheme, host)

	return oc
}

func (c *AppsController) getSpaceNameFromSpaceID(ctx context.Context, spaceID uuid.UUID) (*string, error) {
	// TODO - add a cache in AppsController - but will break if user can change space name
	// use WIT API to convert Space UUID to Space name
	osioclient := getAndCheckOSIOClientV1(ctx)

	osioSpace, err := osioclient.GetSpaceByID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	return osioSpace.Attributes.Name, nil
}

func getNamespaceNameV1(ctx context.Context) (*string, error) {

	osioclient := getAndCheckOSIOClientV1(ctx)
	kubeSpaceAttr, err := osioclient.GetNamespaceByType(ctx, nil, "user")
	if err != nil {
		return nil, err
	}
	if kubeSpaceAttr == nil || kubeSpaceAttr.Name == nil {
		return nil, witerrors.NewNotFoundError("namespace", "user")
	}

	return kubeSpaceAttr.Name, nil
}

func getUserV1(authClient authservice.Client, ctx context.Context) (*authservice.User, error) {
	// get the user definition (for cluster URL)
	resp, err := authClient.ShowUser(ctx, authservice.ShowUserPath(), nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		return nil, fmt.Errorf("Failed to GET user due to status code %d", status)
	}

	var respType authservice.User
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

func getTokenDataV1(authClient authservice.Client, ctx context.Context, forService string) (*authservice.TokenData, error) {

	resp, err := authClient.RetrieveToken(ctx, authservice.RetrieveTokenPath(), forService, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		return nil, errors.New("Failed to GET user due to status code " + string(status))
	}

	var respType authservice.TokenData
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

// GetKubeClient creates a kube client for the appropriate cluster assigned to the current user
func (g *defaultKubeClientGetterV1) GetKubeClientV1(ctx context.Context) (kubernetesV1.KubeClientInterface, error) {

	// create Auth API client
	authClient, err := auth.CreateClient(ctx, g.config)
	if err != nil {
		log.Error(ctx, nil, "error accessing Auth server"+tostring(err))
		return nil, err
	}

	authUser, err := getUserV1(*authClient, ctx)
	if err != nil {
		log.Error(ctx, nil, "error accessing Auth server"+tostring(err))
		return nil, err
	}

	if authUser == nil || authUser.Data.Attributes.Cluster == nil {
		log.Error(ctx, nil, "error getting user from Auth server:"+tostring(authUser))
		return nil, fmt.Errorf("error getting user from Auth Server: %s", tostring(authUser))
	}

	// get the openshift/kubernetes auth info for the cluster OpenShift API
	osauth, err := getTokenDataV1(*authClient, ctx, *authUser.Data.Attributes.Cluster)
	if err != nil {
		log.Error(ctx, nil, "error getting openshift credentials:"+tostring(err))
		return nil, err
	}

	kubeURL := *authUser.Data.Attributes.Cluster
	kubeToken := *osauth.AccessToken

	kubeNamespaceName, err := getNamespaceNameV1(ctx)
	if err != nil {
		return nil, err
	}

	// create the cluster API client
	kubeConfig := &kubernetesV1.KubeClientConfig{
		ClusterURL:    kubeURL,
		BearerToken:   kubeToken,
		UserNamespace: *kubeNamespaceName,
	}
	kc, err := kubernetesV1.NewKubeClient(kubeConfig)
	if err != nil {
		return nil, err
	}
	return kc, nil
}

// SetDeployment runs the setDeployment action.
func (c *AppsController) SetDeployment(ctx *app.SetDeploymentAppsContext) error {

	// we double check podcount here, because in the future we might have different query parameters
	// (for setting different Pod switches) and PodCount might become optional
	if ctx.PodCount == nil {
		return witerrors.NewBadParameterError("podCount", "missing")
	}

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	oldCount, err := kc.ScaleDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName, *ctx.PodCount)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}

	log.Info(ctx, nil, "podcount was %d; will be set to %d", *oldCount, *ctx.PodCount)
	return ctx.OK([]byte{})
}

// ShowDeploymentStatSeries runs the showDeploymentStatSeries action.
func (c *AppsController) ShowDeploymentStatSeries(ctx *app.ShowDeploymentStatSeriesAppsContext) error {

	endTime := time.Now()
	startTime := endTime.Add(-8 * time.Hour) // default: start time is 8 hours before end time
	limit := -1                              // default: No limit

	if ctx.Limit != nil {
		limit = *ctx.Limit
	}

	if ctx.Start != nil {
		startTime = convertToTime(int64(*ctx.Start))
	}

	if ctx.End != nil {
		endTime = convertToTime(int64(*ctx.End))
	}

	if endTime.Before(startTime) {
		return witerrors.NewBadParameterError("end", *ctx.End)
	}

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return err
	}

	statSeries, err := kc.GetDeploymentStatSeries(*kubeSpaceName, ctx.AppName, ctx.DeployName,
		startTime, endTime, limit)
	if err != nil {
		return err
	} else if statSeries == nil {
		return witerrors.NewNotFoundError("deployment", ctx.DeployName)
	}

	res := &app.SimpleDeploymentStatSeriesV1Single{
		Data: statSeries,
	}

	return ctx.OK(res)
}

// ShowDeploymentStats runs the showDeploymentStats action.
func (c *AppsController) ShowDeploymentStats(ctx *app.ShowDeploymentStatsAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	var startTime time.Time
	if ctx.Start != nil {
		startTime = convertToTime(int64(*ctx.Start))
	} else {
		// If a start time was not supplied, default to one minute ago
		startTime = time.Now().Add(-1 * time.Minute)
	}

	deploymentStats, err := kc.GetDeploymentStats(*kubeSpaceName, ctx.AppName, ctx.DeployName, startTime)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if deploymentStats == nil {
		return witerrors.NewNotFoundError("deployment", ctx.DeployName)
	}

	res := &app.SimpleDeploymentStatsV1Single{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvironment runs the showEnvironment action.
func (c *AppsController) ShowEnvironment(ctx *app.ShowEnvironmentAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	env, err := kc.GetEnvironment(ctx.EnvName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if env == nil {
		return witerrors.NewNotFoundError("environment", ctx.EnvName)
	}

	res := &app.SimpleEnvironmentV1Single{
		Data: env,
	}

	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *AppsController) ShowSpace(ctx *app.ShowSpaceAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	// get OpenShift space
	space, err := kc.GetSpace(*kubeSpaceName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if space == nil {
		return witerrors.NewNotFoundError("space", *kubeSpaceName)
	}

	res := &app.SimpleSpaceV1Single{
		Data: space,
	}

	return ctx.OK(res)
}

// ShowSpaceApp runs the showSpaceApp action.
func (c *AppsController) ShowSpaceApp(ctx *app.ShowSpaceAppAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	theapp, err := kc.GetApplication(*kubeSpaceName, ctx.AppName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if theapp == nil {
		return witerrors.NewNotFoundError("application", ctx.AppName)
	}

	res := &app.SimpleApplicationV1Single{
		Data: theapp,
	}

	return ctx.OK(res)
}

// ShowSpaceAppDeployment runs the showSpaceAppDeployment action.
func (c *AppsController) ShowSpaceAppDeployment(ctx *app.ShowSpaceAppDeploymentAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	deploymentStats, err := kc.GetDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if deploymentStats == nil {
		return witerrors.NewNotFoundError("deployment statistics", ctx.DeployName)
	}

	res := &app.SimpleDeploymentV1Single{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvAppPods runs the showEnvAppPods action.
func (c *AppsController) ShowEnvAppPods(ctx *app.ShowEnvAppPodsAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	pods, err := kc.GetPodsInNamespace(ctx.EnvName, ctx.AppName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if pods == nil || len(pods) == 0 {
		return witerrors.NewNotFoundError("pods", ctx.AppName)
	}
	jsonresp := "{\"pods\":" + tostring(pods) + "}\n"

	return ctx.OK([]byte(jsonresp))
}

// ShowSpaceEnvironments runs the showSpaceEnvironments action.
func (c *AppsController) ShowSpaceEnvironments(ctx *app.ShowSpaceEnvironmentsAppsContext) error {

	kc, err := c.GetKubeClientV1(ctx)
	defer cleanupV1(kc)
	if err != nil {
		return witerrors.NewUnauthorizedError("openshift token")
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if envs == nil {
		return witerrors.NewNotFoundError("environments", ctx.SpaceID.String())
	}

	res := &app.SimpleEnvironmentV1List{
		Data: envs,
	}

	return ctx.OK(res)
}

func cleanupV1(kc kubernetesV1.KubeClientInterface) {
	if kc != nil {
		kc.Close()
	}
}
