package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/transaction"
	"github.com/goadesign/goa"
)

// TrackerqueryController implements the trackerquery resource.
type TrackerqueryController struct {
	*goa.Controller
	tqRepository models.TrackerQueryRepository
	ts           transaction.Support
	scheduler    *remoteworkitem.Scheduler
}

// NewTrackerqueryController creates a trackerquery controller.
func NewTrackerqueryController(service *goa.Service, tqRepository models.TrackerQueryRepository, ts transaction.Support, scheduler *remoteworkitem.Scheduler) *TrackerqueryController {
	return &TrackerqueryController{Controller: service.NewController("TrackerqueryController"), tqRepository: tqRepository, ts: ts, scheduler: scheduler}
}

// Create runs the create action.
func (c *TrackerqueryController) Create(ctx *app.CreateTrackerqueryContext) error {
	result := transaction.Do(c.ts, func() error {
		tq, err := c.tqRepository.Create(ctx.Context, ctx.Payload.Query, ctx.Payload.Schedule, uint64(ctx.Payload.Tracker))
		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
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

// Delete runs the delete action.
func (c *TrackerqueryController) Delete(ctx *app.DeleteTrackerqueryContext) error {
	// TrackerqueryController_Delete: start_implement

	// Put your logic here

	// TrackerqueryController_Delete: end_implement
	return nil
}

// List runs the list action.
func (c *TrackerqueryController) List(ctx *app.ListTrackerqueryContext) error {
	// TrackerqueryController_List: start_implement

	// Put your logic here

	// TrackerqueryController_List: end_implement
	res := app.TrackerQueryCollection{}
	return ctx.OK(res)
}

// Show runs the show action.
func (c *TrackerqueryController) Show(ctx *app.ShowTrackerqueryContext) error {
	// TrackerqueryController_Show: start_implement

	// Put your logic here

	// TrackerqueryController_Show: end_implement
	res := &app.TrackerQuery{}
	return ctx.OK(res)
}

// Update runs the update action.
func (c *TrackerqueryController) Update(ctx *app.UpdateTrackerqueryContext) error {
	result := transaction.Do(c.ts, func() error {

		toSave := app.TrackerQuery{
			ID:       ctx.ID,
			Query:    ctx.Payload.Query,
			Schedule: ctx.Payload.Schedule,
		}
		tq, err := c.tqRepository.Save(ctx.Context, toSave)

		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
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
