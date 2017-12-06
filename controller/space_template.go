package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
)

var APISpaceTemplates = "spacetemplates"

// SpaceTemplateController implements the space_template resource.
type SpaceTemplateController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceTemplateController creates a space_template controller.
func NewSpaceTemplateController(service *goa.Service, db application.DB) *SpaceTemplateController {
	return &SpaceTemplateController{Controller: service.NewController("SpaceTemplateController"), db: db}
}

// Show runs the show action.
func (c *SpaceTemplateController) Show(ctx *app.ShowSpaceTemplateContext) error {
	// till we have space templates in place, let this controller redirect
	// user to the typegroups endpoint that returns list of type-groups.
	err := application.Transactional(c.db, func(appl application.Application) error {
		return appl.Spaces().CheckExists(ctx, ctx.SpaceTemplateID.String())
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	typeGroupURL := app.SpaceTemplateHref(ctx.SpaceTemplateID) + "/workitemtypegroups/"
	ctx.ResponseData.Header().Set("Location", typeGroupURL)
	return ctx.TemporaryRedirect()
}
