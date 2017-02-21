package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

// WorkItemChildrenController implements the work-item-children resource.
type WorkItemChildrenController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemChildrenController creates a work-item-children controller.
func NewWorkItemChildrenController(service *goa.Service, db application.DB) *WorkItemChildrenController {
	return &WorkItemChildrenController{Controller: service.NewController("WorkItemChildrenController"), db: db}
}

// List runs the list action.
func (c *WorkItemChildrenController) List(ctx *app.ListWorkItemChildrenContext) error {
	// WorkItemChildrenController_List: start_implement

	// Put your logic here
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.WorkItems().ListChildren(ctx, ctx.ID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		response := app.WorkItem2List{
			Links: &app.PagingLinks{},
			Data:  ConvertWorkItems(ctx.RequestData, result),
		}
		return ctx.OK(&response)
	})
}
