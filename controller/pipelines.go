package controller

import (
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/errors"
)

// pipeline implements the pipeline resource.
type PipelinesController struct {
	*goa.Controller
	Config *configuration.Registry
	ClientGetter
}

func NewPipelineController(service *goa.Service, config *configuration.Registry) *PipelinesController {
	return &PipelinesController{
		Controller: service.NewController("PipelinesController"),
		Config:     config,
		ClientGetter: &defaultClientGetter{
			config: config,
		},
	}
}

// Delete a pipelines from given space
func (c *PipelinesController) Delete(ctx *app.DeletePipelinesContext) error {

	osioClient, err := c.GetAndCheckOSIOClient(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	k8sSpace, err := osioClient.GetNamespaceByType(ctx, nil, "user")
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to retrieve 'user' namespace"))
	}
	if k8sSpace == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("namespace", "user"))
	}

	osc, err := c.GetOSClient(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	userNS := *k8sSpace.Name
	resp, err := osc.DeleteBuildConfig(userNS, map[string]string{"space": ctx.Space})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        err,
			"space_name": ctx.Space,
		}, "error occurred while deleting pipeline")
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	log.Info(ctx, map[string]interface{}{"response": resp}, "deleted pipelines :")

	return ctx.OK([]byte{})
}