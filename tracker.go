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

// TrackerController implements the tracker resource.
type TrackerController struct {
	*goa.Controller
	db        application.DB
	scheduler *remoteworkitem.Scheduler
}

// NewTrackerController creates a tracker controller.
func NewTrackerController(service *goa.Service, db application.DB, scheduler *remoteworkitem.Scheduler) *TrackerController {
	return &TrackerController{Controller: service.NewController("TrackerController"), db: db, scheduler: scheduler}
}

// Create runs the create action.
func (c *TrackerController) Create(ctx *app.CreateTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		t, err := appl.Trackers().Create(ctx.Context, ctx.Payload.URL, ctx.Payload.Type)
		if err != nil {
			switch err := err.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerHref(t.ID))
		return ctx.Created(t)
	})
	c.scheduler.ScheduleAllQueries()
	return result
}

// Delete runs the delete action.
func (c *TrackerController) Delete(ctx *app.DeleteTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Trackers().Delete(ctx.Context, ctx.ID)
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

// Show runs the show action.
func (c *TrackerController) Show(ctx *app.ShowTrackerContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		t, err := appl.Trackers().Load(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case remoteworkitem.NotFoundError:
				log.Printf("not found, id=%s", ctx.ID)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
		}
		return ctx.OK(t)
	})
}

// List runs the list action.
func (c *TrackerController) List(ctx *app.ListTrackerContext) error {
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error()))
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.Trackers().List(ctx.Context, exp, start, &limit)
		if err != nil {
			return goa.ErrInternal(fmt.Sprintf("Error listing trackers: %s", err.Error()))
		}
		return ctx.OK(result)
	})

}

// Update runs the update action.
func (c *TrackerController) Update(ctx *app.UpdateTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {

		toSave := app.Tracker{
			ID:   ctx.ID,
			URL:  ctx.Payload.URL,
			Type: ctx.Payload.Type,
		}
		t, err := appl.Trackers().Save(ctx.Context, toSave)

		if err != nil {
			switch err := err.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		return ctx.OK(t)
	})
	c.scheduler.ScheduleAllQueries()
	return result
}
