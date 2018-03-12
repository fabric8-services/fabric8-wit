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
func enrichLinkTypeSingle(ctx *workItemLinkContext, single *app.WorkItemLinkTypeSingle) error {
	// Add "links" element
	relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(*single.Data.ID))
	single.Data.Links = &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}

	// // Now include the optional link category data in the work item link type "included" array
	// modelCategory, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, single.Data.Relationships.LinkCategory.Data.ID)
	// if err != nil {
	// 	return err
	// }
	// appCategory := ConvertLinkCategoryFromModel(*modelCategory)
	// single.Included = append(single.Included, appCategory.Data)

	// Now include the system space in the work item link type "included" array
	//
	// NOTE: We always include the system space in order to not break the API
	// now. Technically speaking a work item link type belongs to a space
	// template and not to a space.
	//
	// TODO(kwk): Deprecate this.
	// space, err := ctx.Application.Spaces().Load(ctx.Context, space.SystemSpace)
	// if err != nil {
	// 	return err
	// }
	// spaceData, err := ConvertSpaceFromModel(ctx.Request, *space, IncludeBacklogTotalCount(ctx.Context, ctx.DB))
	// if err != nil {
	// 	return err
	// }
	// spaceSingle := &app.SpaceSingle{
	// 	Data: spaceData,
	// }
	// single.Included = append(single.Included, spaceSingle.Data)

	return nil
}

// enrichLinkTypeList includes related resources in the list's "included" array
func enrichLinkTypeList(ctx *workItemLinkContext, list *app.WorkItemLinkTypeList) error {
	// Add "links" element
	for _, data := range list.Data {
		relatedURL := rest.AbsoluteURL(ctx.Request, ctx.LinkFunc(*data.ID))
		data.Links = &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		}
	}
	// // Build our "set" of distinct category IDs already converted as strings
	// categoryIDMap := map[uuid.UUID]bool{}
	// for _, typeData := range list.Data {
	// 	categoryIDMap[typeData.Relationships.LinkCategory.Data.ID] = true
	// }
	// // Now include the optional link category data in the work item link type "included" array
	// for categoryID := range categoryIDMap {
	// 	modelCategory, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, categoryID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	appCategory := ConvertLinkCategoryFromModel(*modelCategory)
	// 	list.Included = append(list.Included, appCategory.Data)
	// }

	// // Now include the system space in the work item link type "included" array
	// //
	// // NOTE: We always include the system space in order to not break the API
	// // now. Technically speaking a work item link type belongs to a space
	// // template and not to a space.
	// //
	// // TODO(kwk): Deprecate this.
	// space, err := ctx.Application.Spaces().Load(ctx.Context, space.SystemSpace)
	// if err != nil {
	// 	return err
	// }
	// spaceData, err := ConvertSpaceFromModel(ctx.Request, *space, IncludeBacklogTotalCount(ctx.Context, ctx.DB))
	// if err != nil {
	// 	return err
	// }
	// spaceSingle := &app.SpaceSingle{
	// 	Data: spaceData,
	// }
	// list.Included = append(list.Included, spaceSingle.Data)
	return nil
}

// Show runs the show action.
func (c *WorkItemLinkTypeController) Show(ctx *app.ShowWorkItemLinkTypeContext) error {
	// WorkItemLinkTypeController_Show: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLinkType, err := appl.WorkItemLinkTypes().Load(ctx.Context, ctx.WiltID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*modelLinkType, c.config.GetCacheControlWorkItemLinkType, func() error {
			// Convert the created link type entry into a rest representation
			appLinkType := ConvertWorkItemLinkTypeFromModel(ctx.Request, *modelLinkType)

			// Enrich
			hrefFunc := func(obj interface{}) string {
				return fmt.Sprintf(app.WorkItemLinkTypeHref("%v"), obj)
			}
			linkCtx := newWorkItemLinkContext(ctx.Context, ctx.Service, appl, c.db, ctx.Request, ctx.ResponseWriter, hrefFunc, nil)
			err = enrichLinkTypeSingle(linkCtx, &appLinkType)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("Failed to enrich link type: %s", err.Error()))
			}
			return ctx.OK(&appLinkType)
		})
	})
	// WorkItemLinkTypeController_Show: end_implement
}

// ConvertWorkItemLinkTypeFromModel converts a work item link type from model to REST representation
func ConvertWorkItemLinkTypeFromModel(request *http.Request, modelLinkType link.WorkItemLinkType) app.WorkItemLinkTypeSingle {
	linkCategoryRelatedURL := rest.AbsoluteURL(request, app.WorkItemLinkCategoryHref(modelLinkType.LinkCategoryID.String()))

	spaceTemplateRelatedURL := rest.AbsoluteURL(request, app.SpaceTemplateHref(modelLinkType.SpaceTemplateID.String()))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(space.SystemSpace.String()))

	topologyStr := modelLinkType.Topology.String()
	var converted = app.WorkItemLinkTypeSingle{
		Data: &app.WorkItemLinkTypeData{
			Type: link.EndpointWorkItemLinkTypes,
			ID:   &modelLinkType.ID,
			Attributes: &app.WorkItemLinkTypeAttributes{
				Name:        &modelLinkType.Name,
				Description: modelLinkType.Description,
				Version:     &modelLinkType.Version,
				CreatedAt:   &modelLinkType.CreatedAt,
				UpdatedAt:   &modelLinkType.UpdatedAt,
				ForwardName: &modelLinkType.ForwardName,
				ReverseName: &modelLinkType.ReverseName,
				Topology:    &topologyStr,
			},
			Relationships: &app.WorkItemLinkTypeRelationships{
				LinkCategory: &app.RelationWorkItemLinkCategory{
					Data: &app.RelationWorkItemLinkCategoryData{
						Type: link.EndpointWorkItemLinkCategories,
						ID:   modelLinkType.LinkCategoryID,
					},
					Links: &app.GenericLinks{
						Self:    &linkCategoryRelatedURL,
						Related: &linkCategoryRelatedURL,
					},
				},
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

		if attrs.Description != nil {
			modelLinkType.Description = attrs.Description
		}

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

	if rel != nil && rel.LinkCategory != nil && rel.LinkCategory.Data != nil {
		modelLinkType.LinkCategoryID = rel.LinkCategory.Data.ID
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
