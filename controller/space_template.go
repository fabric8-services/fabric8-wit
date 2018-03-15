package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/goadesign/goa"
)

// APISpaceTemplates is the URL a) the URL portion in /api/spacetemplates and b)
// the "type" string in a JSON API resource object.
var APISpaceTemplates = "spacetemplates"

// SpaceTemplateController implements the space_template resource.
type SpaceTemplateController struct {
	*goa.Controller
	db     application.DB
	config SpaceTemplateControllerConfiguration
}

// SpaceTemplateControllerConfiguration the configuration for the SpaceTemplateController
type SpaceTemplateControllerConfiguration interface {
	GetCacheControlSpaceTemplates() string
}

// NewSpaceTemplateController creates a space_template controller.
func NewSpaceTemplateController(service *goa.Service, db application.DB, config SpaceTemplateControllerConfiguration) *SpaceTemplateController {
	return &SpaceTemplateController{Controller: service.NewController("SpaceTemplateController"), db: db, config: config}
}

// List runs the list action.
func (c *SpaceTemplateController) List(ctx *app.ListSpaceTemplateContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		templates, err := appl.SpaceTemplates().List(ctx)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to list space templates")
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		err = ctx.ConditionalEntities(templates, c.config.GetCacheControlSpaceTemplates, func() error {
			res := &app.SpaceTemplateList{}
			res.Data = ConvertSpaceTemplates(appl, ctx.Request, templates)
			return ctx.OK(res)
		})
		return err
	})
}

// Show runs the show action.
func (c *SpaceTemplateController) Show(ctx *app.ShowSpaceTemplateContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		st, err := appl.SpaceTemplates().Load(ctx, ctx.SpaceTemplateID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":               err,
				"space_template_id": ctx.SpaceTemplateID,
			}, "failed to load space template")
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*st, c.config.GetCacheControlSpaceTemplates, func() error {
			res := &app.SpaceTemplateSingle{}
			res.Data = ConvertSpaceTemplate(appl, ctx.Request, *st)
			return ctx.OK(res)
		})
	})
}

// SpaceTemplateConvertFunc is a open ended function to add additional links/data/relations to a space template during
// convertion from internal to API
type SpaceTemplateConvertFunc func(application.Application, *http.Request, *spacetemplate.SpaceTemplate, *app.SpaceTemplate) error

// ConvertSpaceTemplates converts between internal and external REST representation
func ConvertSpaceTemplates(appl application.Application, request *http.Request, spaceTemplates []spacetemplate.SpaceTemplate, additional ...SpaceTemplateConvertFunc) []*app.SpaceTemplate {
	var is = []*app.SpaceTemplate{}
	for _, i := range spaceTemplates {
		is = append(is, ConvertSpaceTemplate(appl, request, i, additional...))
	}
	return is
}

// ConvertSpaceTemplate converts between internal and external REST representation
func ConvertSpaceTemplate(appl application.Application, request *http.Request, st spacetemplate.SpaceTemplate, additional ...SpaceTemplateConvertFunc) *app.SpaceTemplate {

	// template := base64.StdEncoding.EncodeToString([]byte(st.Template))
	i := &app.SpaceTemplate{
		Type: APISpaceTemplates,
		ID:   &st.ID,
		Attributes: &app.SpaceTemplateAttributes{
			Name:           &st.Name,
			CreatedAt:      &st.CreatedAt,
			UpdatedAt:      &st.UpdatedAt,
			Version:        &st.Version,
			Description:    st.Description,
			IsBaseTemplate: ptr.Bool(st.ID == spacetemplate.SystemBaseTemplateID),
			// Template:    &template,
		},
		Relationships: &app.SpaceTemplateRelationships{
			Workitemtypes: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: ptr.String(rest.AbsoluteURL(request, app.SpaceTemplateHref(st.ID)+"/workitemtypes")),
				},
			},
			Workitemlinktypes: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: ptr.String(rest.AbsoluteURL(request, app.SpaceTemplateHref(st.ID)+"/workitemlinktypes")),
				},
			},
			Workitemtypegroups: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: ptr.String(rest.AbsoluteURL(request, app.SpaceTemplateHref(st.ID)+"/workitemtypegroups")),
				},
			},
		},
		Links: &app.GenericLinks{
			Self: ptr.String(rest.AbsoluteURL(request, app.SpaceTemplateHref(st.ID))),
		},
	}

	for _, add := range additional {
		add(appl, request, &st, i)
	}
	return i
}
