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
		return createWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, ctx.Payload)
	})
}

// List runs the list action.
func (c *WorkItemRelationshipsLinksController) List(ctx *app.ListWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		return listWorkItemLink(newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref), ctx, &ctx.ID)
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
