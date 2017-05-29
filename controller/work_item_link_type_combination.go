package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// WorkItemLinkTypeCombinationController implements the work_item_link_type_combination resource.
type WorkItemLinkTypeCombinationController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLinkTypeControllerConfiguration
}

// NewWorkItemLinkTypeCombinationController creates a work_item_link_type_combination controller.
func NewWorkItemLinkTypeCombinationController(service *goa.Service, db application.DB, config WorkItemLinkTypeControllerConfiguration) *WorkItemLinkTypeCombinationController {
	return &WorkItemLinkTypeCombinationController{
		Controller: service.NewController("WorkItemLinkTypeCombinationController"),
		db:         db,
		config:     config,
	}
}

// Create runs the create action.
func (c *WorkItemLinkTypeCombinationController) Create(ctx *app.CreateWorkItemLinkTypeCombinationContext) error {
	m, err := ConvertWorkItemLinkTypeCominationToModel(*ctx.Payload.Data)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		_, err = appl.WorkItemLinkTypeCombinations().Create(ctx, &m)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		convertedToApp, err := ConvertWorkItemLinkTypeCombinationFromModel(appl, ctx.RequestData, m)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.WorkItemLinkTypeCombinationSingle{
			Data: convertedToApp,
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.WorkItemLinkTypeCombinationHref(m.SpaceID, res.Data.ID)))
		return ctx.Created(res)
	})
}

// Show runs the show action.
func (c *WorkItemLinkTypeCombinationController) Show(ctx *app.ShowWorkItemLinkTypeCombinationContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		m, err := appl.WorkItemLinkTypeCombinations().Load(ctx, ctx.WiltcID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntity(*m, c.config.GetCacheControlWorkItemLinkTypeCombinations, func() error {
			convertedToApp, err := ConvertWorkItemLinkTypeCombinationFromModel(appl, ctx.RequestData, *m)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			res := &app.WorkItemLinkTypeCombinationSingle{
				Data: convertedToApp,
			}
			return ctx.OK(res)
		})
	})
}

// ConvertWorkItemLinkTypeCombinationsFromModel converts between internal and external REST representation
func ConvertWorkItemLinkTypeCombinationsFromModel(appl application.Application, request *goa.RequestData, combis []link.WorkItemLinkTypeCombination, additional ...WorkItemLinkTypeCombinationConvertFunc) ([]*app.WorkItemLinkTypeCombinationData, error) {
	var res = []*app.WorkItemLinkTypeCombinationData{}
	for _, i := range combis {
		converted, err := ConvertWorkItemLinkTypeCombinationFromModel(appl, request, i, additional...)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		res = append(res, converted)
	}
	return res, nil
}

// WorkItemLinkTypeCombinationConvertFunc is a open ended function to add additional links/data/relations to a type combination during
// convertion from internal to API
type WorkItemLinkTypeCombinationConvertFunc func(application.Application, *goa.RequestData, *link.WorkItemLinkTypeCombination, *app.WorkItemLinkTypeCombinationData) error

func ConvertWorkItemLinkTypeCominationToModel(a app.WorkItemLinkTypeCombinationData) (link.WorkItemLinkTypeCombination, error) {
	m := link.WorkItemLinkTypeCombination{}
	if err := a.Validate(); err != nil {
		return m, errs.Wrap(err, "failed to validate work item link type combination")
	}
	if a.ID != nil {
		m.ID = *a.ID
	}
	if a.Attributes != nil {
		if a.Attributes.Version != nil {
			m.Version = *a.Attributes.Version
		}
		if a.Attributes.CreatedAt != nil {
			m.CreatedAt = *a.Attributes.CreatedAt
		}
		if a.Attributes.UpdatedAt != nil {
			m.UpdatedAt = *a.Attributes.UpdatedAt
		}
	}
	rel := a.Relationships
	if rel != nil {
		if rel.Space != nil && rel.Space.Data != nil && rel.Space.Data.ID != nil {
			m.SpaceID = *rel.Space.Data.ID
		}
		if rel.SourceType != nil && rel.SourceType.Data != nil {
			m.SourceTypeID = rel.SourceType.Data.ID
		}
		if rel.TargetType != nil && rel.TargetType.Data != nil {
			m.TargetTypeID = rel.TargetType.Data.ID
		}
		if rel.LinkType != nil && rel.LinkType.Data != nil {
			m.LinkTypeID = rel.LinkType.Data.ID
		}
	}
	return m, nil
}

// ConvertWorkItemLinkTypeCombinationFromModel converts a type combination from a
// model representation to the app representation.
func ConvertWorkItemLinkTypeCombinationFromModel(appl application.Application, request *goa.RequestData, m link.WorkItemLinkTypeCombination, additional ...WorkItemLinkTypeCombinationConvertFunc) (*app.WorkItemLinkTypeCombinationData, error) {
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(m.SpaceID.String()))
	witTargetSelfURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(m.SpaceID.String(), m.TargetTypeID.String()))
	witSourceSelfURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(m.SpaceID.String(), m.SourceTypeID.String()))
	parentSelfURL := rest.AbsoluteURL(request, app.WorkItemLinkTypeCombinationHref(m.SpaceID.String(), m.SourceTypeID.String()))
	a := &app.WorkItemLinkTypeCombinationData{
		Type: link.EndpointWorkItemLinkTypeCombinations,
		ID:   &m.ID,
		Links: &app.GenericLinks{
			Self: &parentSelfURL,
		},
		Attributes: &app.WorkItemLinkTypeCombinationAttributes{
			CreatedAt: &m.CreatedAt,
			UpdatedAt: &m.UpdatedAt,
			Version:   &m.Version,
		},
		Relationships: &app.WorkItemLinkTypeCombinationRelationships{
			Space: app.NewSpaceRelation(m.SpaceID, spaceSelfURL),
			LinkType: &app.RelationWorkItemLinkType{
				Data: &app.RelationWorkItemLinkTypeData{
					Type: link.EndpointWorkItemLinkTypes,
					ID:   m.LinkTypeID,
				},
			},
			SourceType: &app.RelationWorkItemType{
				Data: &app.RelationWorkItemTypeData{
					Type: link.EndpointWorkItemTypes,
					ID:   m.SourceTypeID,
				},
				Links: &app.GenericLinks{
					Self: &witSourceSelfURL,
				},
			},
			TargetType: &app.RelationWorkItemType{
				Data: &app.RelationWorkItemTypeData{
					Type: link.EndpointWorkItemTypes,
					ID:   m.TargetTypeID,
				},
				Links: &app.GenericLinks{
					Self: &witTargetSelfURL,
				},
			},
		},
	}
	for _, add := range additional {
		add(appl, request, &m, a)
	}
	if err := a.Validate(); err != nil {
		return nil, errs.Wrap(err, "failed to validate work item link type combination")
	}
	return a, nil
}
