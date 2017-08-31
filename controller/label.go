package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
)

// LabelController implements the label resource.
type LabelController struct {
	*goa.Controller
	db     application.DB
	config LabelControllerConfiguration
}

// LabelControllerConfiguration the configuration for the LabelController
type LabelControllerConfiguration interface {
	GetCacheControlLabels() string
	GetCacheControlLabel() string
}

// NewLabelController creates a label controller.
func NewLabelController(service *goa.Service, db application.DB, config LabelControllerConfiguration) *LabelController {
	return &LabelController{
		Controller: service.NewController("LabelController"),
		db:         db,
		config:     config}
}

// Show retrieve a single label
func (c *LabelController) Show(ctx *app.ShowLabelContext) error {
	return nil
}

// Create runs the create action.
func (c *LabelController) Create(ctx *app.CreateLabelContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		lbl := label.Label{SpaceID: ctx.SpaceID, Name: *ctx.Payload.Data.Attributes.Name, Color: *ctx.Payload.Data.Attributes.Color}
		err = appl.Labels().Create(ctx, &lbl)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.LabelSingle{
		//		Data: ConvertLabel(appl, ctx.RequestData, lbl),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.LabelHref(ctx.SpaceID, res.Data.ID)))
		return ctx.Created(res)
	})

}

// List runs the list action.
func (c *LabelController) List(ctx *app.ListLabelContext) error {
	// LabelController_List: start_implement

	// Put your logic here

	// LabelController_List: end_implement
	res := &app.LabelList{}
	return ctx.OK(res)
}
