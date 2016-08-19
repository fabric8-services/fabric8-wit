package main

import (
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
	witRepository models.WorkItemTypeRepository
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service, witRepository models.WorkItemTypeRepository) *WorkitemtypeController {
	return &WorkitemtypeController{Controller: service.NewController("WorkitemtypeController"), witRepository: witRepository}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	res, err := c.witRepository.Load(ctx.Context, ctx.ID)
	if err != nil {
		switch err.(type) {
		case models.NotFoundError:
			log.Printf("not found, id=%s", ctx.ID)
			return goa.ErrNotFound(err.Error())
		default:
			return err
		}
	}
	return ctx.OK(res)
}
