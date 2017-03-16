package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
)

// ListWorkItemChildren runs the list action.
func (c *WorkitemController) ListWorkItemChildren(ctx *app.ListWorkItemChildrenWorkitemContext) error {
	// WorkItemChildrenController_List: start_implement

	// Put your logic here
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItemLinks().ListWorkItemChildren(ctx, ctx.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		response := app.WorkItem2List{
			Data: ConvertWorkItems(ctx.RequestData, result),
		}
		return ctx.OK(&response)
	})
}

// WorkItemIncludeChildren adds relationship about children to workitem (include totalCount)
func WorkItemIncludeChildren(request *goa.RequestData, wi *app.WorkItem, wi2 *app.WorkItem2) {
	wi2.Relationships.Children = CreateChildrenRelation(request, wi)
}

// CreateChildrenRelation returns a RelationGeneric object representing the relation for a workitem to child relation
func CreateChildrenRelation(request *goa.RequestData, wi *app.WorkItem) *app.RelationGeneric {
	return &app.RelationGeneric{
		Links: CreateChildrenRelationLinks(request, wi),
	}
}

// CreateChildrenRelationLinks returns a RelationGeneric object representing the links for a workitem to child relation
func CreateChildrenRelationLinks(request *goa.RequestData, wi *app.WorkItem) *app.GenericLinks {
	childrenRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID)) + "/children"
	return &app.GenericLinks{
		Related: &childrenRelated,
	}
}
