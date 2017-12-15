package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/auth/authservice"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	witerrors "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AppsController implements the apps resource.
type AppsController struct {
	*goa.Controller
	Config *configuration.Registry
}

// NewAppsController creates a apps controller.
func NewAppsController(service *goa.Service, config *configuration.Registry) *AppsController {
	return &AppsController{
		Controller: service.NewController("AppsController"),
		Config:     config,
	}
}

func tostring(item interface{}) string {
	bytes, _ := json.MarshalIndent(item, "", "  ")
	return string(bytes)
}

func (c *AppsController) getAndCheckOSIOClient(ctx context.Context) *OSIOClient {

	// TODO - grab the WIT URL from the incoming request if possible
	witURL := "localhost"

	// TODO - remove this debug hook before production
	if os.Getenv("OSIO_WIT_URL") != "" {
		witURL = os.Getenv("OSIO_WIT_URL")
	}

	oc := NewOSIOClient(ctx, witURL)

	return oc
}

func (c *AppsController) getSpaceNameFromSpaceID(ctx context.Context, spaceID uuid.UUID) (*string, error) {
	// TODO - add a cache in AppsController
	// use WIT API to convert Space UUID to Space name
	oc := c.getAndCheckOSIOClient(ctx)

	osioSpace, err := oc.GetSpaceByID(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	return osioSpace.Attributes.Name, nil
}

func (c *AppsController) getNamespaceName(ctx context.Context) (*string, error) {

	osioclient := c.getAndCheckOSIOClient(ctx)
	kubeSpaceAttr, err := osioclient.GetNamespaceByType(ctx, nil, "user")
	if err != nil {
		return nil, err
	}
	if kubeSpaceAttr == nil || kubeSpaceAttr.Name == nil {
		return nil, witerrors.NewNotFoundError("namespace", "user")
	}

	return kubeSpaceAttr.Name, nil
}

func (c *AppsController) getUser(authClient authservice.Client, ctx context.Context) (*authservice.User, error) {
	// get the user definition (for cluster URL)
	resp, err := authClient.ShowUser(ctx, authservice.ShowUserPath(), nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET user due to status code " + string(status))
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType authservice.User
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

func (c *AppsController) getToken(authClient authservice.Client, ctx context.Context, forService string) (*authservice.TokenData, error) {

	resp, err := authClient.RetrieveToken(ctx, authservice.RetrieveTokenPath(), forService, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	status := resp.StatusCode
	if status < 200 || status > 300 {
		return nil, errors.New("Failed to GET user due to status code " + string(status))
	}

	respBody, err := ioutil.ReadAll(resp.Body)

	var respType authservice.TokenData
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, err
	}
	return &respType, nil
}

// getKubeClient createa kube client for the appropriate cluster assigned to the current user.
// many different errors are possible, so controllers should call getAndCheckKubeClient() instead
func (c *AppsController) getKubeClient(ctx context.Context) (*kubernetes.KubeClient, error) {

	// create Auth API login object
	authClient, err := auth.CreateClient(ctx, c.Config)
	if err != nil {
		log.Error(ctx, nil, "error accessing Auth server"+tostring(err))
		return nil, err
	}

	authUser, err := c.getUser(*authClient, ctx)
	if err != nil {
		log.Error(ctx, nil, "error accessing Auth server"+tostring(err))
		return nil, err
	}

	if authUser == nil || authUser.Data.Attributes.Cluster == nil {
		log.Error(ctx, nil, "error getting user from Auth server:"+tostring(authUser))
		return nil, errors.New("error getting user from Auth Server: %s" + tostring(authUser))
	}

	// get the login token for the cluster OpenShift API
	osauth, err := c.getToken(*authClient, ctx, *authUser.Data.Attributes.Cluster)
	if err != nil {
		log.Error(ctx, nil, "error getting openshift credentials:"+tostring(err))
		return nil, err
	}

	kubeURL := *authUser.Data.Attributes.Cluster
	kubeToken := *osauth.AccessToken

	kubeNamespaceName, err := c.getNamespaceName(ctx)
	if err != nil {
		return nil, err
	}

	// create the cluster login object
	kc, err := kubernetes.NewKubeClient(kubeURL, kubeToken, *kubeNamespaceName)
	if err != nil {
		return nil, err
	}
	return kc, nil
}

// getAndCheckKubeClient converts all errors Error 401, so errors can be returned from controllers as is
func (c *AppsController) getAndCheckKubeClient(ctx context.Context) (*kubernetes.KubeClient, error) {

	kc, err := c.getKubeClient(ctx)
	if err != nil {
		return nil, witerrors.NewUnauthorizedError("openshift token")
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

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return witerrors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	oldCount, err := kc.ScaleDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName, *ctx.PodCount)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}

	log.Info(ctx, nil, "podcount was ", *oldCount, " will be set to "+strconv.Itoa(*ctx.PodCount))
	return ctx.OK([]byte{})
}

// ShowDeploymentStatSeries runs the showDeploymentStatSeries action.
func (c *AppsController) ShowDeploymentStatSeries(ctx *app.ShowDeploymentStatSeriesAppsContext) error {

	endTime := time.Now()
	startTime := endTime.Add(-8 * time.Hour)
	limit := -1 // No limit

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

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	res := &app.SimpleDeploymentStatSeriesSingle{
		Data: statSeries,
	}

	return ctx.OK(res)
}

func convertToTime(unixMillis int64) time.Time {
	return time.Unix(0, unixMillis*int64(time.Millisecond))
}

// ShowDeploymentStats runs the showDeploymentStats action.
func (c *AppsController) ShowDeploymentStats(ctx *app.ShowDeploymentStatsAppsContext) error {

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	res := &app.SimpleDeploymentStatsSingle{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvironment runs the showEnvironment action.
func (c *AppsController) ShowEnvironment(ctx *app.ShowEnvironmentAppsContext) error {

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	env, err := kc.GetEnvironment(ctx.EnvName)
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if env == nil {
		return witerrors.NewNotFoundError("environment", ctx.EnvName)
	}

	res := &app.SimpleEnvironmentSingle{
		Data: env,
	}

	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *AppsController) ShowSpace(ctx *app.ShowSpaceAppsContext) error {
	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	res := &app.SimpleSpaceSingle{
		Data: space,
	}

	return ctx.OK(res)
}

// ShowSpaceApp runs the showSpaceApp action.
func (c *AppsController) ShowSpaceApp(ctx *app.ShowSpaceAppAppsContext) error {

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	res := &app.SimpleApplicationSingle{
		Data: theapp,
	}

	return ctx.OK(res)
}

// ShowSpaceAppDeployment runs the showSpaceAppDeployment action.
func (c *AppsController) ShowSpaceAppDeployment(ctx *app.ShowSpaceAppDeploymentAppsContext) error {

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	res := &app.SimpleDeploymentSingle{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvAppPods runs the showEnvAppPods action.
func (c *AppsController) ShowEnvAppPods(ctx *app.ShowEnvAppPodsAppsContext) error {

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
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

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return witerrors.NewInternalError(ctx, err)
	}
	if envs == nil {
		return witerrors.NewNotFoundError("environments", ctx.SpaceID.String())
	}

	res := &app.SimpleEnvironmentList{
		Data: envs,
	}

	return ctx.OK(res)
}
