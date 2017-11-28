package controller

import (
	"context"
	"encoding/json"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AppsController implements the apps resource.
type AppsController struct {
	*goa.Controller
	AuthURL string
	WitURL  string
}

// NewAppsController creates a apps controller.
func NewAppsController(service *goa.Service, configuration auth.ServiceConfiguration) *AppsController {
	return &AppsController{
		Controller: service.NewController("AppsController"),

		//AuthURL: configuration.GetAuthServiceURL(),
		//AuthURL: "http://localhost:8089/api"
		//AuthURL: "https://auth.prod-preview.openshift.io/api",
		AuthURL: "https://auth.openshift.io/api",

		// TODO
		//WitURL: "http://localhost:8080/api"
		//WitURL: "http://api.prod-preview.openshift.io/api",
		WitURL: "http://api.openshift.io/api",
	}
}

func tostring(item interface{}) string {
	bytes, _ := json.MarshalIndent(item, "", "  ")
	return string(bytes)
}

func (c *AppsController) getAndCheckOsioClient(ctx context.Context) (*OsioClient, error) {

	oc, err := NewOsioClient(ctx, c.WitURL)
	if err != nil {
		return nil, errors.NewUnauthorizedError("osio")
	}
	return oc, nil
}

func (c *AppsController) getSpaceNameFromSpaceID(ctx context.Context, spaceID uuid.UUID) (*string, error) {
	// use WIT API to convert Space UUID to Space name
	oc, err := c.getAndCheckOsioClient(ctx)
	if err != nil {
		return nil, err
	}

	osioSpace, err := oc.GetSpaceByID(spaceID.String(), false)
	if err != nil {
		return nil, errors.NewNotFoundError("osio space", spaceID.String())
	}
	return osioSpace.Attributes.Name, nil
}

func (c *AppsController) getKubeClient(ctx context.Context) (*KubeClient, error) {

	// create Auth API login object
	authClient, err := NewAuthClient(ctx, c.AuthURL)
	if err != nil {
		goa.LogInfo(ctx, "errror creating auth client:"+tostring(err))
		return nil, err
	}

	// get the user definition (for cluster URL)
	authUser, err := authClient.getAuthUser()
	if err != nil {
		goa.LogInfo(ctx, "error getting user info:"+tostring(err))
		return nil, err
	}

	// get the login token for the cluster OpenShift API
	osauth, err := authClient.getAuthToken(*authUser.Data.Attributes.Cluster)
	if err != nil {
		goa.LogInfo(ctx, "error getting openshift credentials:"+tostring(err))
		return nil, err
	}

	kubeURL := *authUser.Data.Attributes.Cluster
	kubeSpace := *authUser.Data.Attributes.Username
	kubeToken := *osauth.AccessToken

	// create the cluster login object
	kc, err := NewKubeClient(kubeURL, kubeToken, kubeSpace)
	if err != nil {
		return nil, err
	}
	return kc, nil
}

func (c *AppsController) getAndCheckKubeClient(ctx context.Context) (*KubeClient, error) {

	kc, err := c.getKubeClient(ctx)
	if err != nil {
		goa.LogInfo(ctx, "didn't actually get a token")
		return nil, errors.NewUnauthorizedError("openshift token")
	}
	return kc, nil
}

// SetDeployment runs the setDeployment action.
func (c *AppsController) SetDeployment(ctx *app.SetDeploymentAppsContext) error {
	// AppsController_SetDeployment: start_implement

	// AppsController_SetDeployment: end_implement
	return ctx.OK([]byte{})
}

// ShowDeploymentStatSeries runs the showDeploymentStatSeries action.
func (c *AppsController) ShowDeploymentStatSeries(ctx *app.ShowDeploymentStatSeriesAppsContext) error {
	// AppsController_ShowDeploymentStatSeries: start_implement

	res := &app.SimpleDeploymentStatSeries{}

	// AppsController_ShowDeploymentStatSeries: end_implement
	return ctx.OK(res)
}

// ShowDeploymentStats runs the showDeploymentStats action.
func (c *AppsController) ShowDeploymentStats(ctx *app.ShowDeploymentStatsAppsContext) error {
	// AppsController_ShowDeploymentStats: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return err
	}

	deploymentStats, err := kc.GetDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName)
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if deploymentStats == nil {
		return errors.NewNotFoundError("deployment", ctx.DeployName)
	}

	res := &app.SimpleDeploymentSingle{
		Data: deploymentStats,
	}

	// AppsController_ShowDeploymentStats: end_implement
	return ctx.OK(res)
}

// ShowEnvironment runs the showEnvironment action.
func (c *AppsController) ShowEnvironment(ctx *app.ShowEnvironmentAppsContext) error {
	// AppsController_ShowEnvironment: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	env, err := kc.GetEnvironment(ctx.EnvName)
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if env == nil {
		return errors.NewNotFoundError("environment", ctx.EnvName)
	}

	res := &app.SimpleEnvironmentSingle{
		Data: env,
	}

	// AppsController_ShowEnvironment: end_implement
	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *AppsController) ShowSpace(ctx *app.ShowSpaceAppsContext) error {
	// AppsController_ShowSpace: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return err
	}

	// get OpenShift space
	space, err := kc.GetSpace(*kubeSpaceName)
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if space == nil {
		return errors.NewNotFoundError("space", *kubeSpaceName)
	}

	res := &app.SimpleSpaceSingle{
		Data: space,
	}

	// AppsController_ShowSpace: end_implement
	return ctx.OK(res)
}

// ShowSpaceApp runs the showSpaceApp action.
func (c *AppsController) ShowSpaceApp(ctx *app.ShowSpaceAppAppsContext) error {
	// AppsController_ShowSpaceApp: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return err
	}

	theapp, err := kc.GetApplication(*kubeSpaceName, ctx.AppName)
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if theapp == nil {
		return errors.NewNotFoundError("application", ctx.AppName)
	}

	res := &app.SimpleApplicationSingle{
		Data: theapp,
	}

	// AppsController_ShowSpaceApp: end_implement
	return ctx.OK(res)
}

// ShowSpaceAppDeployment runs the showSpaceAppDeployment action.
func (c *AppsController) ShowSpaceAppDeployment(ctx *app.ShowSpaceAppDeploymentAppsContext) error {
	// AppsController_ShowSpaceAppDeployment: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return err
	}

	deploymentStats, err := kc.GetDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName)
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if deploymentStats == nil {
		return errors.NewNotFoundError("deployment statistics", ctx.DeployName)
	}

	res := &app.SimpleDeploymentSingle{
		Data: deploymentStats,
	}

	// AppsController_ShowSpaceAppDeployment: end_implement
	return ctx.OK(res)
}

// ShowSpaceEnvironments runs the showSpaceEnvironments action.
func (c *AppsController) ShowSpaceEnvironments(ctx *app.ShowSpaceEnvironmentsAppsContext) error {
	// AppsController_ShowSpaceEnvironments: start_implement

	kc, err := c.getAndCheckKubeClient(ctx)
	if err != nil {
		return err
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if envs == nil {
		return errors.NewNotFoundError("environments", ctx.SpaceID.String())
	}

	res := &app.SimpleEnvironmentList{
		Data: envs,
	}

	// AppsController_ShowSpaceEnvironments: end_implement
	return ctx.OK(res)
}
