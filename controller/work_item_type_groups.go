package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
)

// WorkItemTypeGroupsController implements the work_item_type_groups resource.
type WorkItemTypeGroupsController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemTypeGroupsController creates a work_item_type_groups controller.
func NewWorkItemTypeGroupsController(service *goa.Service, db application.DB) *WorkItemTypeGroupsController {
	return &WorkItemTypeGroupsController{
		Controller: service.NewController("WorkItemTypeGroupsController"),
		db:         db,
	}
}

// List runs the list action.
func (c *WorkItemTypeGroupsController) List(ctx *app.ListWorkItemTypeGroupsContext) error {
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
