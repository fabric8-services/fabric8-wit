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
		lbl := label.Label{
			SpaceID:         ctx.SpaceID,
			Name:            ctx.Payload.Data.Attributes.Name,
			TextColor:       *ctx.Payload.Data.Attributes.TextColor,
			BackgroundColor: *ctx.Payload.Data.Attributes.BackgroundColor,
		}
		err = appl.Labels().Create(ctx, &lbl)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.LabelSingle{
			Data: ConvertLabel(appl, ctx.Request, lbl),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.LabelHref(ctx.SpaceID, res.Data.ID)))
		return ctx.Created(res)
	})
}

// ConvertLabel converts from internal to external REST representation
func ConvertLabel(appl application.Application, request *http.Request, lbl label.Label) *app.Label {
	labelType := label.APIStringTypeLabels
	spaceID := lbl.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.LabelHref(spaceID, lbl.ID))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	l := &app.Label{
		Type: labelType,
		ID:   &lbl.ID,
		Attributes: &app.LabelAttributes{
			TextColor:       &lbl.TextColor,
			BackgroundColor: &lbl.BackgroundColor,
			Name:            lbl.Name,
			CreatedAt:       &lbl.CreatedAt,
			UpdatedAt:       &lbl.UpdatedAt,
			Version:         &lbl.Version,
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
	return application.Transactional(c.db, func(appl application.Application) error {
		labels, err := appl.Labels().List(ctx, ctx.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.LabelList{}
		res.Data = ConvertLabels(appl, ctx.Request, labels)
		return ctx.OK(res)
	})
}

// ConvertLabels from internal to external REST representation
func ConvertLabels(appl application.Application, request *http.Request, labels []label.Label) []*app.Label {
	var ls = []*app.Label{}
	for _, i := range labels {
		ls = append(ls, ConvertLabel(appl, request, i))
	}
	return ls
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
