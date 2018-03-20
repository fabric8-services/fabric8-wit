package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/auth/authservice"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// DeploymentsController implements the deployments resource.
type DeploymentsController struct {
	*goa.Controller
	Config *configuration.Registry
	ClientGetter
}

// ClientGetter creates an instances of clients used by this controller
type ClientGetter interface {
	GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error)
	GetAndCheckOSIOClient(ctx context.Context) (OpenshiftIOClient, error)
}

// Default implementation of KubeClientGetter and OSIOClientGetter used by NewDeploymentsController
type defaultClientGetter struct {
	config *configuration.Registry
}

// NewDeploymentsController creates a deployments controller.
func NewDeploymentsController(service *goa.Service, config *configuration.Registry) *DeploymentsController {
	return &DeploymentsController{
		Controller: service.NewController("DeploymentsController"),
		Config:     config,
		ClientGetter: &defaultClientGetter{
			config: config,
		},
	}
}

func tostring(item interface{}) string {
	bytes, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

func (*defaultClientGetter) GetAndCheckOSIOClient(ctx context.Context) (OpenshiftIOClient, error) {

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

	// The deployments API communicates with the rest of WIT via the stnadard WIT API.
	// This environment variable is used for local development of the deployments API, to point ot a remote WIT.
	witURLStr := os.Getenv("FABRIC8_WIT_API_URL")
	if witURLStr != "" {
		witurl, err := url.Parse(witURLStr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"FABRIC8_WIT_API_URL": witURLStr,
				"err": err,
			}, "cannot parse FABRIC8_WIT_API_URL: %s", witURLStr)
			return nil, errs.Wrapf(err, "cannot parse FABRIC8_WIT_API_URL: %s", witURLStr)
		}
		host = witurl.Host
		scheme = witurl.Scheme
	}

	oc := NewOSIOClient(ctx, scheme, host)

	return oc, nil
}

// getSpaceNameFromSpaceID() converts an OSIO Space UUID to an OpenShift space name.
// will return an error if the space is not found.
func (c *DeploymentsController) getSpaceNameFromSpaceID(ctx context.Context, spaceID uuid.UUID) (*string, error) {
	// TODO - add a cache in DeploymentsController - but will break if user can change space name
	// use WIT API to convert Space UUID to Space name
	osioclient, err := c.GetAndCheckOSIOClient(ctx)
	if err != nil {
		return nil, err
	}

	osioSpace, err := osioclient.GetSpaceByID(ctx, spaceID)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to convert space UUID %s to space name", spaceID)
	}
	if osioSpace == nil || osioSpace.Attributes == nil || osioSpace.Attributes.Name == nil {
		return nil, errs.Errorf("space UUID %s is not valid space name", spaceID)
	}
	return osioSpace.Attributes.Name, nil
}

func (g *defaultClientGetter) getNamespaceName(ctx context.Context) (*string, error) {

	osioclient, err := g.GetAndCheckOSIOClient(ctx)
	if err != nil {
		return nil, err
	}

	kubeSpaceAttr, err := osioclient.GetNamespaceByType(ctx, nil, "user")
	if err != nil {
		return nil, errs.Wrap(err, "unable to retrieve 'user' namespace")
	}
	if kubeSpaceAttr == nil {
		return nil, errors.NewNotFoundError("namespace", "user")
	}

	return kubeSpaceAttr.Name, nil
}

func getUser(ctx context.Context, authClient authservice.Client) (*authservice.User, error) {
	// get the user definition (for cluster URL)
	resp, err := authClient.ShowUser(ctx, authservice.ShowUserPath(), nil, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to retrieve user from Auth service")
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"response_body": respBody,
			"http_status":   status,
		}, "failed to GET user from auth service due to HTTP error %s", status)
		return nil, errs.Errorf("failed to GET user due to status code %d", status)
	}

	var respType authservice.User
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"response_body": respBody,
		}, "unable to unmarshal user definition from Auth service")
		return nil, errs.Wrapf(err, "unable to unmarshal user definition from Auth service")
	}
	return &respType, nil
}

func getTokenData(ctx context.Context, authClient authservice.Client, forService string) (*authservice.TokenData, error) {

	resp, err := authClient.RetrieveToken(ctx, authservice.RetrieveTokenPath(), forService, nil)
	if err != nil {
		return nil, errs.Wrapf(err, "unable to retrieve Auth token for '%s' service", forService)
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)

	status := resp.StatusCode
	if status != http.StatusOK {
		log.Error(nil, map[string]interface{}{
			"err":          err,
			"request_path": authservice.ShowUserPath(),
			"for_service":  forService,
			"http_status":  status,
		}, "failed to GET token from auth service due to HTTP error %s", status)
		return nil, errs.Errorf("failed to GET Auth token for '%s' service due to status code %d", forService, status)
	}

	var respType authservice.TokenData
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":           err,
			"request_path":  authservice.ShowUserPath(),
			"for_service":   forService,
			"http_status":   status,
			"response_body": respBody,
		}, "unable to unmarshal Auth token")
		return nil, errs.Wrapf(err, "unable to unmarshal Auth token for '%s' service from Auth service", forService)
	}
	return &respType, nil
}

// GetKubeClient creates a kube client for the appropriate cluster assigned to the current user
func (g *defaultClientGetter) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {

	// create Auth API client
	authClient, err := auth.CreateClient(ctx, g.config)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Auth server")
		return nil, errs.Wrap(err, "error creating Auth client")
	}

	authUser, err := getUser(ctx, *authClient)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error accessing Auth server")
		return nil, errs.Wrap(err, "error retrieving user definition from Auth client")
	}

	if authUser == nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error retrieving user from Auth server")
		return nil, errs.New("error getting user from Auth Server")
	}

	if authUser.Data.Attributes.Cluster == nil {
		log.Error(ctx, map[string]interface{}{
			"err":     err,
			"user_id": *authUser.Data.Attributes.UserID,
		}, "error retrieving user cluster from Auth server")
		return nil, errs.Errorf("error getting user cluster from Auth Server: %s", tostring(authUser))
	}

	// get the openshift/kubernetes auth info for the cluster OpenShift API
	osauth, err := getTokenData(ctx, *authClient, *authUser.Data.Attributes.Cluster)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":     err,
			"user_id": *authUser.Data.Attributes.UserID,
			"cluster": *authUser.Data.Attributes.Cluster,
		}, "error getting openshift credentials for user from Auth server")
		return nil, errs.Wrap(err, "error getting openshift credentials")
	}

	kubeURL := *authUser.Data.Attributes.Cluster
	kubeToken := *osauth.AccessToken

	kubeNamespaceName, err := g.getNamespaceName(ctx)
	if err != nil {
		return nil, errs.Wrap(err, "could not retrieve namespace name")
	}

	/* Timeout used per HTTP request to Kubernetes/OpenShift API servers.
	 * Communication with Hawkular currently uses a hard-coded 30 second
	 * timeout per request, and does not use this parameter. */
	// TODO replace with configurable timeout
	const timeout time.Duration = 30 * time.Second

	// create the cluster API client
	kubeConfig := &kubernetes.KubeClientConfig{
		ClusterURL:    kubeURL,
		BearerToken:   kubeToken,
		UserNamespace: *kubeNamespaceName,
		Timeout:       timeout,
	}
	kc, err := kubernetes.NewKubeClient(kubeConfig)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":     err,
			"user_id": *authUser.Data.Attributes.UserID,
			"cluster": *authUser.Data.Attributes.Cluster,
		}, "could not create Kubernetes client object")
		return nil, errs.Wrap(err, "could not create Kubernetes client object")
	}
	return kc, nil
}

// SetDeployment runs the setDeployment action.
func (c *DeploymentsController) SetDeployment(ctx *app.SetDeploymentDeploymentsContext) error {

	// we double check podcount here, because in the future we might have different query parameters
	// (for setting different Pod switches) and PodCount might become optional
	if ctx.PodCount == nil {
		return errors.NewBadParameterError("podCount", "missing")
	}

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return errors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	_ /*oldCount*/, err = kc.ScaleDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName, *ctx.PodCount)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "error scaling deployment %s", ctx.DeployName))
	}

	return ctx.OK([]byte{})
}

// DeleteDeployment runs the deleteDeployment action.
func (c *DeploymentsController) DeleteDeployment(ctx *app.DeleteDeploymentDeploymentsContext) error {
	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	err = kc.DeleteDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        err,
			"space_name": *kubeSpaceName,
		}, "error deleting deployment")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	return ctx.OK([]byte{})
}

// ShowDeploymentStatSeries runs the showDeploymentStatSeries action.
func (c *DeploymentsController) ShowDeploymentStatSeries(ctx *app.ShowDeploymentStatSeriesDeploymentsContext) error {

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
		return errors.NewBadParameterError("end", *ctx.End)
	}

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
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
		return errors.NewNotFoundError("deployment", ctx.DeployName)
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
func (c *DeploymentsController) ShowDeploymentStats(ctx *app.ShowDeploymentStatsDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return errors.NewNotFoundError("osio space", ctx.SpaceID.String())
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
		return errors.NewInternalError(ctx, errs.Wrapf(err, "could not retrieve deployment statistics for %s", ctx.DeployName))
	}
	if deploymentStats == nil {
		return errors.NewNotFoundError("deployment", ctx.DeployName)
	}

	deploymentStats.ID = ctx.DeployName

	res := &app.SimpleDeploymentStatsSingle{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvironment runs the showEnvironment action.
func (c *DeploymentsController) ShowEnvironment(ctx *app.ShowEnvironmentDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	env, err := kc.GetEnvironment(ctx.EnvName)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "could not retrieve environment %s", ctx.EnvName))
	}
	if env == nil {
		return errors.NewNotFoundError("environment", ctx.EnvName)
	}

	env.ID = *env.Attributes.Name

	res := &app.SimpleEnvironmentSingle{
		Data: env,
	}

	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *DeploymentsController) ShowSpace(ctx *app.ShowSpaceDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil || kubeSpaceName == nil {
		return errors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	// get OpenShift space
	space, err := kc.GetSpace(*kubeSpaceName)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "could not retrieve space %s", *kubeSpaceName))
	}
	if space == nil {
		return errors.NewNotFoundError("openshift space", *kubeSpaceName)
	}

	// Kubernetes doesn't know about space ID, so add it here
	space.ID = ctx.SpaceID

	res := &app.SimpleSpaceSingle{
		Data: space,
	}

	return ctx.OK(res)
}

// ShowSpaceApp runs the showSpaceApp action.
func (c *DeploymentsController) ShowSpaceApp(ctx *app.ShowSpaceAppDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return errors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	theapp, err := kc.GetApplication(*kubeSpaceName, ctx.AppName)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "could not retrieve application %s", ctx.AppName))
	}
	if theapp == nil {
		return errors.NewNotFoundError("application", ctx.AppName)
	}

	theapp.ID = theapp.Attributes.Name

	res := &app.SimpleApplicationSingle{
		Data: theapp,
	}

	return ctx.OK(res)
}

// ShowSpaceAppDeployment runs the showSpaceAppDeployment action.
func (c *DeploymentsController) ShowSpaceAppDeployment(ctx *app.ShowSpaceAppDeploymentDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return errors.NewNotFoundError("osio space", ctx.SpaceID.String())
	}

	deploymentStats, err := kc.GetDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "error retrieving deployment %s", ctx.DeployName))
	}
	if deploymentStats == nil {
		return errors.NewNotFoundError("deployment statistics", ctx.DeployName)
	}

	deploymentStats.ID = deploymentStats.Attributes.Name

	res := &app.SimpleDeploymentSingle{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowEnvAppPods runs the showEnvAppPods action.
func (c *DeploymentsController) ShowEnvAppPods(ctx *app.ShowEnvAppPodsDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	pods, err := kc.GetPodsInNamespace(ctx.EnvName, ctx.AppName)
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrapf(err, "error retrieving pods from namespace %s/%s", ctx.EnvName, ctx.AppName))
	}
	if pods == nil || len(pods) == 0 {
		return errors.NewNotFoundError("pods", ctx.AppName)
	}

	jsonresp := fmt.Sprintf("{\"data\":{\"attributes\":{\"environment\":\"%s\",\"application\":\"%s\",\"pods\":%s}}}", ctx.EnvName, ctx.AppName, tostring(pods))

	return ctx.OK([]byte(jsonresp))
}

// ShowSpaceEnvironments runs the showSpaceEnvironments action.
func (c *DeploymentsController) ShowSpaceEnvironments(ctx *app.ShowSpaceEnvironmentsDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return errors.NewUnauthorizedError("openshift token")
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return errors.NewInternalError(ctx, errs.Wrap(err, "error retrieving environments"))
	}
	if envs == nil {
		return errors.NewNotFoundError("environments", ctx.SpaceID.String())
	}

	res := &app.SimpleEnvironmentList{
		Data: envs,
	}

	return ctx.OK(res)
}

func cleanup(kc kubernetes.KubeClientInterface) {
	if kc != nil {
		kc.Close()
	}
}
