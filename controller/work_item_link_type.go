package controller

import (
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
)

// WorkItemLinkTypeController implements the work-item-link-type resource.
type WorkItemLinkTypeController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLinkTypeControllerConfiguration
}

// WorkItemLinkTypeControllerConfiguration the configuration for the WorkItemLinkTypeController
type WorkItemLinkTypeControllerConfiguration interface {
	GetCacheControlWorkItemLinkTypes() string
	GetCacheControlWorkItemLinkType() string
}

// NewWorkItemLinkTypeController creates a work-item-link-type controller.
func NewWorkItemLinkTypeController(service *goa.Service, db application.DB, config WorkItemLinkTypeControllerConfiguration) *WorkItemLinkTypeController {
	return &WorkItemLinkTypeController{
		Controller: service.NewController("WorkItemLinkTypeController"),
		db:         db,
		config:     config,
	}
}

// enrichLinkTypeSingle includes related resources in the single's "included" array
// TODO(kwk): Adding of links in this function can be done during conversion.
func enrichLinkTypeSingle(ctx *workItemLinkContext, single *app.WorkItemLinkTypeSingle) error {
	// Add "links" element
	relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(*single.Data.ID))
	single.Data.Links = &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
	return nil
}

// enrichLinkTypeList includes related resources in the list's "included" array
// TODO(kwk): Adding of links in this function can be done during conversion.
func enrichLinkTypeList(ctx *workItemLinkContext, list *app.WorkItemLinkTypeList) error {
	// Add "links" element
	for _, data := range list.Data {
		relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(*data.ID))
		data.Links = &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		}
	}
	return nil
}

// Show runs the show action.
func (c *WorkItemLinkTypeController) Show(ctx *app.ShowWorkItemLinkTypeContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		modelLinkType, err := appl.WorkItemLinkTypes().Load(ctx.Context, ctx.WiltID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*modelLinkType, c.config.GetCacheControlWorkItemLinkType, func() error {
			// Convert the created link type entry into a rest representation
			appLinkType := ConvertWorkItemLinkTypeFromModel(ctx.Request, *modelLinkType)

			// Enrich
			HrefFunc := func(obj interface{}) string {
				return fmt.Sprintf(app.WorkItemLinkTypeHref("%s"), obj)
			}
			linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, HrefFunc, nil)
			err = enrichLinkTypeSingle(linkCtx, &appLinkType)
			if err != nil {
				return goa.ErrInternal("Failed to enrich link type: %s", err.Error())
			}
			return ctx.OK(&appLinkType)
		})
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return nil
}

// ConvertWorkItemLinkTypeFromModel converts a work item link type from model to REST representation
func ConvertWorkItemLinkTypeFromModel(request *http.Request, modelLinkType link.WorkItemLinkType) app.WorkItemLinkTypeSingle {
	spaceTemplateRelatedURL := rest.AbsoluteURL(request, app.SpaceTemplateHref(modelLinkType.SpaceTemplateID.String()))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(space.SystemSpace.String()))

	topologyStr := modelLinkType.Topology.String()
	var converted = app.WorkItemLinkTypeSingle{
		Data: &app.WorkItemLinkTypeData{
			Type: link.EndpointWorkItemLinkTypes,
			ID:   &modelLinkType.ID,
			Attributes: &app.WorkItemLinkTypeAttributes{
				Name:               &modelLinkType.Name,
				Description:        modelLinkType.Description,
				Version:            &modelLinkType.Version,
				CreatedAt:          &modelLinkType.CreatedAt,
				UpdatedAt:          &modelLinkType.UpdatedAt,
				ForwardName:        &modelLinkType.ForwardName,
				ForwardDescription: modelLinkType.ForwardDescription,
				ReverseName:        &modelLinkType.ReverseName,
				ReverseDescription: modelLinkType.ReverseDescription,
				Topology:           &topologyStr,
			},
			Relationships: &app.WorkItemLinkTypeRelationships{
				Space:         app.NewSpaceRelation(space.SystemSpace, spaceRelatedURL),
				SpaceTemplate: app.NewSpaceTemplateRelation(modelLinkType.SpaceTemplateID, spaceTemplateRelatedURL),
			},
		},
	}
	return converted
}

// ConvertWorkItemLinkTypeToModel converts the incoming app representation of a work item link type to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
func ConvertWorkItemLinkTypeToModel(appLinkType app.WorkItemLinkTypeSingle) (*link.WorkItemLinkType, error) {
	modelLinkType := link.WorkItemLinkType{}
	if appLinkType.Data == nil {
		return nil, errors.NewBadParameterError("data", nil).Expected("not <nil>")
	}
	if appLinkType.Data.Attributes == nil {
		return nil, errors.NewBadParameterError("data.attributes", nil).Expected("not <nil>")
	}
	if appLinkType.Data.Relationships == nil {
		return nil, errors.NewBadParameterError("data.relationships", nil).Expected("not <nil>")
	}

	attrs := appLinkType.Data.Attributes
	rel := appLinkType.Data.Relationships

	if appLinkType.Data.ID != nil {
		modelLinkType.ID = *appLinkType.Data.ID
	}

	if attrs != nil {
		// If the name is not nil, it MUST NOT be empty
		if attrs.Name != nil {
			if *attrs.Name == "" {
				return nil, errors.NewBadParameterError("data.attributes.name", *attrs.Name)
			}
			modelLinkType.Name = *attrs.Name
		}

		modelLinkType.Description = attrs.Description
		modelLinkType.ForwardDescription = attrs.ForwardDescription
		modelLinkType.ReverseDescription = attrs.ReverseDescription

		if attrs.Version != nil {
			modelLinkType.Version = *attrs.Version
		}

		// If the forwardName is not nil, it MUST NOT be empty
		if attrs.ForwardName != nil {
			if *attrs.ForwardName == "" {
				return nil, errors.NewBadParameterError("data.attributes.forward_name", *attrs.ForwardName)
			}
			modelLinkType.ForwardName = *attrs.ForwardName
		}

		// If the ReverseName is not nil, it MUST NOT be empty
		if attrs.ReverseName != nil {
			if *attrs.ReverseName == "" {
				return nil, errors.NewBadParameterError("data.attributes.reverse_name", *attrs.ReverseName)
			}
			modelLinkType.ReverseName = *attrs.ReverseName
		}

		if attrs.Topology != nil {
			modelLinkType.Topology = link.Topology(*attrs.Topology)
			if err := modelLinkType.Topology.CheckValid(); err != nil {
				return nil, err
			}
		}
	}

	if rel != nil && rel.SpaceTemplate != nil && rel.SpaceTemplate.Data != nil {
		modelLinkType.SpaceTemplateID = rel.SpaceTemplate.Data.ID
	}

	return &modelLinkType, nil
}

func ConvertLinkTypesFromModels(request *http.Request, modelLinkTypes []link.WorkItemLinkType) (*app.WorkItemLinkTypeList, error) {
	appLinkTypes := app.WorkItemLinkTypeList{}
	appLinkTypes.Data = make([]*app.WorkItemLinkTypeData, len(modelLinkTypes))
	for index, modelLinkType := range modelLinkTypes {
		appLinkType := ConvertWorkItemLinkTypeFromModel(request, modelLinkType)
		appLinkTypes.Data[index] = appLinkType.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	appLinkTypes.Meta = &app.WorkItemLinkTypeListMeta{
		TotalCount: len(modelLinkTypes),
	}
	return &appLinkTypes, nil
}
