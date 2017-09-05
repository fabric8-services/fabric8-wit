package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
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
			Data: ConvertLabel(appl, ctx.RequestData, lbl),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.LabelHref(ctx.SpaceID, res.Data.ID)))
		return ctx.Created(res)
	})
}

// ConvertLabel converts from internal to external REST representation
func ConvertLabel(appl application.Application, request *goa.RequestData, lbl label.Label) *app.Label {
	labelType := label.APIStringTypeLabels
	spaceID := lbl.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request.Request, app.LabelHref(spaceID, lbl.ID))
	spaceRelatedURL := rest.AbsoluteURL(request.Request, app.SpaceHref(spaceID))
	l := &app.Label{
		Type: labelType,
		ID:   &lbl.ID,
		Attributes: &app.LabelAttributes{
			Color:     &lbl.Color,
			Name:      &lbl.Name,
			CreatedAt: &lbl.CreatedAt,
			UpdatedAt: &lbl.UpdatedAt,
			Version:   &lbl.Version,
		},
		Relationships: &app.LabelRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &space.SpaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
		},

		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}
	return l
}

// List runs the list action.
func (c *LabelController) List(ctx *app.ListLabelContext) error {
	// LabelController_List: start_implement

	// Put your logic here

	// LabelController_List: end_implement
	res := &app.LabelList{}
	return ctx.OK(res)
}

// ConvertLabelsSimple converts a array of Label IDs into a Generic Reletionship List
func ConvertLabelsSimple(request *http.Request, labelIDs []interface{}) []*app.GenericData {
	ops := []*app.GenericData{}
	for _, labelID := range labelIDs {
		ops = append(ops, ConvertLabelSimple(request, labelID))
	}
	return ops
}

// ConvertLabelSimple converts a Label ID into a Generic Reletionship
func ConvertLabelSimple(request *http.Request, labelID interface{}) *app.GenericData {
	t := label.APIStringTypeLabels
	i := fmt.Sprint(labelID)
	return &app.GenericData{
		Type: &t,
		ID:   &i,
	}
}
