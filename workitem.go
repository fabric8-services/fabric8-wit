package main

import (
	"log"

	"github.com/goadesign/goa"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	wiRepository *models.WorkItemRepository
	ts           models.TransactionSupport
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, wiRepository *models.WorkItemRepository, ts models.TransactionSupport) *WorkitemController {
	ctrl := WorkitemController{Controller: service.NewController("WorkitemController"), wiRepository: wiRepository, ts: ts}
	if ctrl.wiRepository == nil {
		panic("nil work item repository")
	}
	return &ctrl
}

func (c *WorkitemController) doWithTransaction(todo func() error) error {
	if err := c.ts.Begin(); err != nil {
		return err
	}
	if err := todo(); err != nil {
		c.ts.Rollback()
		return err
	}
	return c.ts.Commit()
}

// Show runs the show action.
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	return c.doWithTransaction(func() error {
		wi, err := c.wiRepository.Load(ctx.ID)
		if err == nil {
			return ctx.OK(wi)
		} else {
			switch err.(type) {
			case models.NotFoundError:
				log.Printf("not found, id=%s", ctx.ID)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
		}
	})
}

// Create runs the create action.
func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
	return c.doWithTransaction(func() error {
		wi, err := c.wiRepository.Create(ctx.Payload.Type, ctx.Payload.Name, ctx.Payload.Fields)

		if err == nil {
			ctx.ResponseData.Header().Set("Location", app.WorkitemHref(wi.ID))
			return ctx.Created(wi)
		} else {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
	})
}

// Delete runs the delete action.
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	return c.doWithTransaction(func() error {
		err := c.wiRepository.Delete(ctx.ID)
		if err == nil {
			return ctx.OK([]byte{})
		} else {
			switch err.(type) {
			case models.NotFoundError:
				return goa.ErrNotFound(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
	})
}

// Update runs the update action.
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	return c.doWithTransaction(func() error {

		toSave := app.WorkItem{
			ID:      ctx.Payload.ID,
			Name:    ctx.Payload.Name,
			Type:    ctx.Payload.Type,
			Version: ctx.Payload.Version,
			Fields:  ctx.Payload.Fields,
		}
		wi, err := c.wiRepository.Save(toSave)

		if err == nil {
			return ctx.OK(wi)
		} else {
			switch err := err.(type) {
			case models.BadParameterError, models.ConversionError:
				return goa.ErrBadRequest(err.Error())
			default:
				return goa.ErrInternal(err.Error())
			}
		}
	})
}
