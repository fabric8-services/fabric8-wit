package main

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
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
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
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
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
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
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				log.LogError(ctx, map[string]interface{}{
					"ID": ctx.ID,
				}, "Tracker controller not found")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
			}
		}
		return ctx.OK(t)
	})
}

// List runs the list action.
func (c *TrackerController) List(ctx *app.ListTrackerContext) error {
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.Trackers().List(ctx.Context, exp, start, &limit)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(fmt.Sprintf("Error listing trackers: %s", err.Error())))
			return ctx.InternalServerError(jerrors)
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
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		return ctx.OK(t)
	})
	c.scheduler.ScheduleAllQueries()
	return result
}
