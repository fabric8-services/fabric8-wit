package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jenkinsidler"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/goadesign/goa"
)

// JenkinsController implements the jenkins resource.
type JenkinsController struct {
	*goa.Controller
}

// NewJenkinsController creates a jenkins controller.
func NewJenkinsController(service *goa.Service) *JenkinsController {
	return &JenkinsController{Controller: service.NewController("JenkinsController")}
}

// Start runs the start action.
func (c *JenkinsController) Start(ctx *app.StartJenkinsContext) error {
	// JenkinsController_Start: start_implement

	// Put your logic here
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	idler := jenkinsidler.NewIdler("idler_url")

	//idler.Status()
	// JenkinsController_Start: end_implement
	return nil
}
