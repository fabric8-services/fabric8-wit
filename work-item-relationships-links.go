package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/goadesign/goa"
)

// WorkItemRelationshipsLinksController implements the work-item-relationships-links resource.
type WorkItemRelationshipsLinksController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemRelationshipsLinksController creates a work-item-relationships-links controller.
func NewWorkItemRelationshipsLinksController(service *goa.Service, db application.DB) *WorkItemRelationshipsLinksController {
	if db == nil {
		panic("db must not be nil")
	}
	return &WorkItemRelationshipsLinksController{
		Controller: service.NewController("WorkItemRelationshipsLinksController"),
		db:         db,
	}
}

// Create runs the create action.
func (c *WorkItemRelationshipsLinksController) Create(ctx *app.CreateWorkItemRelationshipsLinksContext) error {
	return createWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID, ctx.Payload)
}

func (c *WorkItemRelationshipsLinksController) Delete(ctx *app.DeleteWorkItemRelationshipsLinksContext) error {
	return deleteWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID, ctx.LinkID)
}

// List runs the list action.
func (c *WorkItemRelationshipsLinksController) List(ctx *app.ListWorkItemRelationshipsLinksContext) error {
	return listWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID)
}

// Show runs the show action.
func (c *WorkItemRelationshipsLinksController) Show(ctx *app.ShowWorkItemRelationshipsLinksContext) error {
	return showWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID, ctx.LinkID)
}

// Update runs the update action.
func (c *WorkItemRelationshipsLinksController) Update(ctx *app.UpdateWorkItemRelationshipsLinksContext) error {
	return updateWorkItemLink(ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID, ctx.Payload)
}
