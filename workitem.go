package main

import (
	"log"

	"github.com/goadesign/goa"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	wiRepository models.WorkItemRepository
	ts           transaction.Support
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, wiRepository models.WorkItemRepository, ts transaction.Support) *WorkitemController {
	ctrl := WorkitemController{Controller: service.NewController("WorkitemController"), wiRepository: wiRepository, ts: ts}
	if ctrl.wiRepository == nil {
		panic("nil work item repository")
	}
	return &ctrl
}

// Show runs the show action.
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	return transaction.Do(c.ts, func() error {
		wi, err := c.wiRepository.Load(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				log.Printf("not found, id=%s", ctx.ID)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
		}
		return ctx.OK(wi)
	})
}

// Create runs the create action.
func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
	return transaction.Do(c.ts, func() error {
		wi, err := c.wiRepository.Create(ctx.Context, ctx.Payload.Type, ctx.Payload.Name, ctx.Payload.Fields)

		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi.ID))
		return ctx.Created(wi)
	})
}

// Delete runs the delete action.
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	return transaction.Do(c.ts, func() error {
		err := c.wiRepository.Delete(ctx.Context, ctx.ID)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				return goa.ErrNotFound(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		return ctx.OK([]byte{})
	})
}

// Update runs the update action.
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	return transaction.Do(c.ts, func() error {

		toSave := app.WorkItem{
			ID:      ctx.Payload.ID,
			Name:    ctx.Payload.Name,
			Type:    ctx.Payload.Type,
			Version: ctx.Payload.Version,
			Fields:  ctx.Payload.Fields,
		}
		wi, err := c.wiRepository.Save(ctx.Context, toSave)

		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
		return ctx.OK(wi)
	})
}
