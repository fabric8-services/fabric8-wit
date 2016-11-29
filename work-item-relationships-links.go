package main

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/workitem/link"
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

func parseWorkItemIDToUint64(wiIDStr string) (uint64, error) {
	wiID, err := strconv.ParseUint(wiIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid work item ID \"%s\": %s", wiIDStr, err.Error())
	}
	return wiID, nil
}

// Create runs the create action.
func (c *WorkItemRelationshipsLinksController) Create(ctx *app.CreateWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Check that current work item does indeed exist
		if _, err := appl.WorkItems().Load(ctx.Context, ctx.ID); err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Check that the source ID of the link is the same as the current work
		// item ID.
		src, _ := getSrcTgt(ctx.Payload.Data)
		if src != nil && *src != ctx.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("data.relationships.source.data.id is \"%s\" but must be \"%s\"", ctx.Payload.Data.Relationships.Source.Data.ID, ctx.ID)))
			return ctx.BadRequest(jerrors)
		}
		// If no source is specified we pre-fill the source field of the payload
		// with the current work item ID from the URL. This is for convenience.
		if src == nil {
			if ctx.Payload.Data.Relationships == nil {
				ctx.Payload.Data.Relationships = &app.WorkItemLinkRelationships{}
			}
			if ctx.Payload.Data.Relationships.Source == nil {
				ctx.Payload.Data.Relationships.Source = &app.RelationWorkItem{}
			}
			if ctx.Payload.Data.Relationships.Source.Data == nil {
				ctx.Payload.Data.Relationships.Source.Data = &app.RelationWorkItemData{}
			}
			ctx.Payload.Data.Relationships.Source.Data.ID = ctx.ID
			ctx.Payload.Data.Relationships.Source.Data.Type = link.EndpointWorkItems
		}
		return createWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.Payload)
	})
}

func (c *WorkItemRelationshipsLinksController) Delete(ctx *app.DeleteWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Check work item link exists
		wil, err := appl.WorkItemLinks().Load(ctx.Context, ctx.LinkID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Only allow deletion if the current work item is at the source of the link
		src, _ := getSrcTgt(wil.Data)
		if *src != ctx.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest("Current work item is not at source of work item link"))
			return ctx.BadRequest(jerrors)
		}
		return deleteWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.LinkID)
	})
}

// List runs the list action.
func (c *WorkItemRelationshipsLinksController) List(ctx *app.ListWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return listWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, &ctx.ID)
	})
}

// Show runs the show action.
func (c *WorkItemRelationshipsLinksController) Show(ctx *app.ShowWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Check work item link exists
		wil, err := appl.WorkItemLinks().Load(ctx.Context, ctx.LinkID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Only allow showing if the current work item is at the source or at the target of the link
		src, tgt := getSrcTgt(wil.Data)
		if *src != ctx.ID && *tgt != ctx.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest("Current work item is not at source nor target of the work item link to show"))
			return ctx.BadRequest(jerrors)
		}
		return showWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.LinkID)
	})
}

// Update runs the update action.
func (c *WorkItemRelationshipsLinksController) Update(ctx *app.UpdateWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		// Check work item link exists
		wil, err := appl.WorkItemLinks().Load(ctx.Context, ctx.LinkID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Only allow updating if the current work item is at the source of the current link
		src, _ := getSrcTgt(wil.Data)
		if *src != ctx.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest("Current work item is not at source of the existing work item link to update"))
			return ctx.BadRequest(jerrors)
		}
		// Only allow updating if the current work item is also at the source of the new link in the payload
		src, _ = getSrcTgt(ctx.Payload.Data)
		if src != nil && *src != ctx.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest("Current work item is not at source of the new work item link update payload"))
			return ctx.BadRequest(jerrors)
		}
		return updateWorkItemLink(appl, ctx.Context, c.db, ctx.ResponseData, ctx, ctx.Payload)
	})
}

func getSrcTgt(wilData *app.WorkItemLinkData) (*string, *string) {
	var src, tgt *string
	if wilData != nil && wilData.Relationships != nil {
		if wilData.Relationships.Source != nil && wilData.Relationships.Source.Data != nil {
			src = &wilData.Relationships.Source.Data.ID
		}
		if wilData.Relationships.Target != nil && wilData.Relationships.Target.Data != nil {
			tgt = &wilData.Relationships.Target.Data.ID
		}
	}
	return src, tgt
}
