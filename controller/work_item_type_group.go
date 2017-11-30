package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
)

// WorkItemTypeGroupController implements the work_item_type_group resource.
type WorkItemTypeGroupController struct {
	*goa.Controller
	db application.DB
}

const APIWorkItemTypeGroups = "workitemtypegroups"

// NewWorkItemTypeGroupController creates a work_item_type_group controller.
func NewWorkItemTypeGroupController(service *goa.Service, db application.DB) *WorkItemTypeGroupController {
	return &WorkItemTypeGroupController{
		Controller: service.NewController("WorkItemTypeGroupController"),
		db:         db,
	}
}

// Show runs the list action.
func (c *WorkItemTypeGroupController) Show(ctx *app.ShowWorkItemTypeGroupContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		return appl.Spaces().CheckExists(ctx, ctx.SpaceTemplateID.String())
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	// TODO(kwk): Replace with loading from DB once type groups are persistently
	// stored in there.
	for _, group := range workitem.TypeGroups() {
		if group.ID == ctx.GroupID {
			return ctx.OK(&app.WorkItemTypeGroupSingle{
				Data: ConvertTypeGroup(ctx.Request, group),
			})
		}
	}
	return jsonapi.JSONErrorResponse(ctx, errors.NewNotFoundError("type group", ctx.GroupID.String()))
}

// List runs the list action.
func (c *WorkItemTypeGroupController) List(ctx *app.ListWorkItemTypeGroupContext) error {
	err := application.Transactional(c.db, func(appl application.Application) error {
		return appl.Spaces().CheckExists(ctx, ctx.SpaceTemplateID.String())
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	typeGroups := workitem.TypeGroups()
	res := &app.WorkItemTypeGroupList{
		Data: make([]*app.WorkItemTypeGroupData, len(typeGroups)),
		Links: &app.WorkItemTypeGroupLinks{
			Self: rest.AbsoluteURL(ctx.Request, app.SpaceTemplateHref(space.SystemSpace)) + "/" + APIWorkItemTypeGroups,
		},
	}
	for i, group := range typeGroups {
		res.Data[i] = ConvertTypeGroup(ctx.Request, group)
	}
	return ctx.OK(res)
}

// ConvertTypeGroup converts WorkitemTypeGroup model to a response resource
// object for jsonapi.org specification
func ConvertTypeGroup(request *http.Request, tg workitem.WorkItemTypeGroup) *app.WorkItemTypeGroupData {

	workitemtypes := "workitemtypes"
	// TODO(kwk): Replace system space once we have space templates
	// defaultWorkItemTypeRelatedURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(space.SystemSpace, tg.DefaultType))
	workItemTypeGroupRelatedURL := rest.AbsoluteURL(request, app.WorkItemTypeGroupHref(space.SystemSpace, tg.ID))
	defaultIDStr := tg.DefaultType.String()
	createdAt := tg.CreatedAt.UTC()
	updatedAt := tg.UpdatedAt.UTC()

	res := &app.WorkItemTypeGroupData{
		ID:   &tg.ID,
		Type: APIWorkItemTypeGroups,
		Links: &app.GenericLinks{
			Related: &workItemTypeGroupRelatedURL,
		},
		Attributes: &app.WorkItemTypeGroupAttributes{
			Bucket:    tg.Bucket.String(),
			Name:      tg.Name,
			Icon:      tg.Icon,
			CreatedAt: &createdAt,
			UpdatedAt: &updatedAt,
		},
		Relationships: &app.WorkItemTypeGroupRelationships{
			DefaultType: &app.RelationGeneric{
				Data: &app.GenericData{
					ID:   &defaultIDStr,
					Type: &workitemtypes,
					// Links: &app.GenericLinks{
					// 	Related: &defaultWorkItemTypeRelatedURL,
					// },
				},
			},
			TypeList: &app.RelationGenericList{
				Data: make([]*app.GenericData, len(tg.TypeList)),
			},
		},
	}

	for i, witID := range tg.TypeList {
		// witRelatedURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(space.SystemSpace, witID))
		idStr := witID.String()
		res.Relationships.TypeList.Data[i] = &app.GenericData{
			ID:   &idStr,
			Type: &workitemtypes,
			// Links: &app.GenericLinks{
			// Related: &witRelatedURL,
			// },
		}
	}
	return res
}
