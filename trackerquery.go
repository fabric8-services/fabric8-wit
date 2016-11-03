package main

import (
	"fmt"
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/goadesign/goa"
)

// TrackerqueryController implements the trackerquery resource.
type TrackerqueryController struct {
	*goa.Controller
	db        application.DB
	scheduler *remoteworkitem.Scheduler
}

// NewTrackerqueryController creates a trackerquery controller.
func NewTrackerqueryController(service *goa.Service, db application.DB, scheduler *remoteworkitem.Scheduler) *TrackerqueryController {
	return &TrackerqueryController{Controller: service.NewController("TrackerqueryController"), db: db, scheduler: scheduler}
}

// Create runs the create action.
func (c *TrackerqueryController) Create(ctx *app.CreateTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		tq, err := appl.TrackerQueries().Create(ctx.Context, ctx.Payload.Query, ctx.Payload.Schedule, ctx.Payload.TrackerID)
		if err != nil {
			switch err := err.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerqueryHref(tq.ID))
		return ctx.Created(tq)
	})
	c.scheduler.ScheduleAllQueries()
	return result
}

// Show runs the show action.
func (c *TrackerqueryController) Show(ctx *app.ShowTrackerqueryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		tq, err := appl.TrackerQueries().Load(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case remoteworkitem.NotFoundError:
				log.Printf("not found, id=%s", ctx.ID)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
		}
		return ctx.OK(tq)
	})
}

// Update runs the update action.
func (c *TrackerqueryController) Update(ctx *app.UpdateTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {

		toSave := app.TrackerQuery{
			ID:        ctx.ID,
			Query:     ctx.Payload.Query,
			Schedule:  ctx.Payload.Schedule,
			TrackerID: ctx.Payload.TrackerID,
		}
		tq, err := appl.TrackerQueries().Save(ctx.Context, toSave)

		if err != nil {
			switch err := err.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		return ctx.OK(tq)
	})
	c.scheduler.ScheduleAllQueries()
	return result
}

// Delete runs the delete action.
func (c *TrackerqueryController) Delete(ctx *app.DeleteTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.TrackerQueries().Delete(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case remoteworkitem.NotFoundError:
				return goa.ErrNotFound(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		return ctx.OK([]byte{})
	})
	c.scheduler.ScheduleAllQueries()
	return result
}

// List runs the list action.
func (c *TrackerqueryController) List(ctx *app.ListTrackerqueryContext) error {
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error()))
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.TrackerQueries().List(ctx.Context, exp, start, &limit)
		if err != nil {
			return goa.ErrInternal(fmt.Sprintf("Error listing tracker queries: %s", err.Error()))
		}
		return ctx.OK(result)
	})

}
