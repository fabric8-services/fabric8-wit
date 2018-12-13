package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/kubernetes"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
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

func (g *defaultClientGetter) GetAndCheckOSIOClient(ctx context.Context) (OpenshiftIOClient, error) {

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
				"err":                 err,
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

// GetKubeClient creates a kube client for the appropriate cluster assigned to the current user
func (g *defaultClientGetter) GetKubeClient(ctx context.Context) (kubernetes.KubeClientInterface, error) {

	kubeNamespaceName, err := g.getNamespaceName(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve namespace name")
		return nil, errs.Wrap(err, "could not retrieve namespace name")
	}

	osioclient, err := g.GetAndCheckOSIOClient(ctx)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not create OSIO client")
		return nil, err
	}

	baseURLProvider, err := NewURLProvider(ctx, g.config, osioclient)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "could not retrieve tenant data")
		return nil, errs.Wrap(err, "could not retrieve tenant data")
	}

	/* Timeout used per HTTP request to Kubernetes/OpenShift API servers.
	 * Communication with Hawkular currently uses a hard-coded 30 second
	 * timeout per request, and does not use this parameter. */
	// create the cluster API client
	kubeConfig := &kubernetes.KubeClientConfig{
		BaseURLProvider: baseURLProvider,
		UserNamespace:   *kubeNamespaceName,
		Timeout:         g.config.GetDeploymentsHTTPTimeoutSeconds(),
	}
	kc, err := kubernetes.NewKubeClient(kubeConfig)
	if err != nil {
		url, _ := baseURLProvider.GetAPIURL()
		log.Error(ctx, map[string]interface{}{
			"err":            err,
			"user_namespace": *kubeNamespaceName,
			"cluster":        *url,
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
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("osio space", ctx.SpaceID.String()))
	}

	_ /*oldCount*/, err = kc.ScaleDeployment(*kubeSpaceName, ctx.AppName, ctx.DeployName, *ctx.PodCount)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "error scaling deployment %s", ctx.DeployName))
	}

	return ctx.OK([]byte{})
}

// DeleteDeployment runs the deleteDeployment action.
func (c *DeploymentsController) DeleteDeployment(ctx *app.DeleteDeploymentDeploymentsContext) error {
	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
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
		return jsonapi.JSONErrorResponse(ctx, err)
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
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("end", *ctx.End))
	}

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	statSeries, err := kc.GetDeploymentStatSeries(*kubeSpaceName, ctx.AppName, ctx.DeployName,
		startTime, endTime, limit)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	} else if statSeries == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("deployment", ctx.DeployName))
	}

	res := &app.SimpleDeploymentStatSeriesSingle{
		Data: statSeries,
	}

	return ctx.OK(res)
}

// ShowDeploymentPodLimitRange runs the showDeploymentPodLimitRange action.
func (c *DeploymentsController) ShowDeploymentPodLimitRange(ctx *app.ShowDeploymentPodLimitRangeDeploymentsContext) error {
	// Inputs : spaceId, appName, deployName
	kc, err := c.GetKubeClient(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	spaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	quotas, err := kc.GetDeploymentPodQuota(*spaceName, ctx.AppName, ctx.DeployName)

	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	res := &app.SimpleDeploymentPodLimitRangeSingle{
		Data: quotas,
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
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("osio space", ctx.SpaceID.String()))
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
		return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err,
			"could not retrieve deployment statistics for deployment '%s' in space '%s'", ctx.DeployName, *kubeSpaceName))
	}
	if deploymentStats == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("deployment", ctx.DeployName))
	}

	deploymentStats.ID = ctx.DeployName

	res := &app.SimpleDeploymentStatsSingle{
		Data: deploymentStats,
	}

	return ctx.OK(res)
}

// ShowSpace runs the showSpace action.
func (c *DeploymentsController) ShowSpace(ctx *app.ShowSpaceDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil || kubeSpaceName == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("osio space", ctx.SpaceID.String()))
	}

	// get OpenShift space
	space, err := kc.GetSpace(*kubeSpaceName)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "could not retrieve space %s", *kubeSpaceName))
	}
	if space == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("openshift space", *kubeSpaceName))
	}

	// Kubernetes doesn't know about space ID, so add it here
	space.ID = ctx.SpaceID

	res := &app.SimpleSpaceSingle{
		Data: space,
	}

	return ctx.OK(res)
}

// ShowSpaceEnvironments runs the showSpaceEnvironments action.
// FIXME Remove this method once showSpaceEnvironments API is removed.
func (c *DeploymentsController) ShowSpaceEnvironments(ctx *app.ShowSpaceEnvironmentsDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error retrieving environments"))
	}
	if envs == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("environments", ctx.SpaceID.String()))
	}

	res := &app.SimpleEnvironmentList{
		Data: envs,
	}

	return ctx.OK(res)
}

// ShowEnvironmentsBySpace runs the showEnvironmentsBySpace action.
func (c *DeploymentsController) ShowEnvironmentsBySpace(ctx *app.ShowEnvironmentsBySpaceDeploymentsContext) error {

	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	kubeSpaceName, err := c.getSpaceNameFromSpaceID(ctx, ctx.SpaceID)
	if err != nil || kubeSpaceName == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("osio space", ctx.SpaceID.String()))
	}

	usage, err := kc.GetSpaceAndOtherEnvironmentUsage(*kubeSpaceName)

	// Model the response
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error retrieving environments"))
	}

	res := &app.SpaceAndOtherEnvironmentUsageList{
		Data: usage,
	}

	return ctx.OK(res)
}

// ShowAllEnvironments runs the showAllEnvironments action.
func (c *DeploymentsController) ShowAllEnvironments(ctx *app.ShowAllEnvironmentsDeploymentsContext) error {
	kc, err := c.GetKubeClient(ctx)
	defer cleanup(kc)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	envs, err := kc.GetEnvironments()
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "error retrieving environments"))
	}
	if envs == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundErrorFromString("no environments found"))
	}

	res := &app.SimpleEnvironmentList{
		Data: envs,
	}

	return ctx.OK(res)
}

// WatchEnvironmentEvents runs the watchEnvironmentEvents action.
func (c *DeploymentsController) WatchEnvironmentEvents(ctx *app.WatchEnvironmentEventsDeploymentsContext) error {
	c.WatchEnvironmentEventsWSHandler(ctx).ServeHTTP(ctx.ResponseWriter, ctx.Request)
	return nil
}

// WatchEnvironmentEventsWSHandler establishes a websocket connection to run the watchEnvironmentEvents action.
func (c *DeploymentsController) WatchEnvironmentEventsWSHandler(ctx *app.WatchEnvironmentEventsDeploymentsContext) websocket.Handler {
	return func(ws *websocket.Conn) {
		defer ws.Close()

		kc, err := c.GetKubeClient(ctx)
		defer cleanup(kc)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "error accessing Auth server")

			sendWebsocketJSON(ctx, ws, map[string]interface{}{"error": "unable to access auth server"})
			return
		}

		store, stopWs := kc.WatchEventsInNamespace(ctx.EnvName)
		defer close(stopWs)

		go func() {
			for {
				var m string
				err := websocket.Message.Receive(ws, &m)
				if err != nil {
					if err != io.EOF {
						log.Error(ctx, map[string]interface{}{
							"err": err,
						}, "error reading from websocket")
					}
					store.Close()
					return
				}
			}
		}()

		for {
			item, err := store.Pop(cache.PopProcessFunc(func(item interface{}) error {
				return nil
			}))

			event, ok := item.(*v1.Event)
			if !ok {
				sendWebsocketJSON(ctx, ws, map[string]interface{}{"error": "Kubernetes event was an unexpected type"})
				return
			}

			if err != nil {
				if err != cache.FIFOClosedError {
					log.Error(ctx, map[string]interface{}{
						"err": err,
					}, "error receiving events")

					sendWebsocketJSON(ctx, ws, map[string]interface{}{"error": "unable to access Kubernetes events"})
				}
				return
			}

			eventItem := transformItem(event)
			if err != nil {
				sendWebsocketJSON(ctx, ws, map[string]interface{}{"error": "unable to parse Kubernetes event"})
			} else {
				err = websocket.JSON.Send(ws, eventItem)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"err": err,
					}, "error sending events")
					return
				}
			}
		}
	}
}

//DeploymentsEvent is the transformed Kubernetes v1.Event item
type DeploymentsEvent struct {
	// The object that this event is about.
	InvolvedObject v1.ObjectReference `json:"involvedObject" protobuf:"bytes,2,opt,name=involvedObject"`

	// This should be a short, machine understandable string that gives the reason
	// for the transition into the object's current status.
	// TODO: provide exact specification for format.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`

	// A human-readable description of the status of this operation.
	// TODO: decide on maximum length.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// The number of times this event has occurred.
	// +optional
	Count int32 `json:"count,omitempty" protobuf:"varint,8,opt,name=count"`

	// Type of this event (Normal, Warning), new types could be added in the future
	// +optional
	Type string `json:"type,omitempty" protobuf:"bytes,9,opt,name=type"`

	// CreationTimestamp is a timestamp representing the server time when this object was
	// created. It is not guaranteed to be set in happens-before order across separate operations.
	// Clients may not set this value. It is represented in RFC3339 form and is in UTC.
	//
	// Populated by the system.
	// Read-only.
	// Null for lists.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	CreationTimestamp metaV1.Time `json:"creationTimestamp,omitempty" protobuf:"bytes,8,opt,name=creationTimestamp"`
}

func transformItem(event *v1.Event) *DeploymentsEvent {
	transformedItem := &DeploymentsEvent{
		InvolvedObject:    event.InvolvedObject,
		Reason:            event.Reason,
		Message:           event.Message,
		Count:             event.Count,
		Type:              event.Type,
		CreationTimestamp: event.ObjectMeta.CreationTimestamp,
	}
	return transformedItem
}

func sendWebsocketJSON(ctx *app.WatchEnvironmentEventsDeploymentsContext, ws *websocket.Conn, item interface{}) {
	err := websocket.JSON.Send(ws, item)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "error sending websocket message")
	}
}

func cleanup(kc kubernetes.KubeClientInterface) {
	if kc != nil {
		kc.Close()
	}
}
