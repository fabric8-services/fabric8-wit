package main

import (
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
	"github.com/goadesign/goa"
	"github.com/almighty/almighty-core/query/simple"
	"fmt"
)

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
	witRepository models.WorkItemTypeRepository
	ts            transaction.Support
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service, witRepository models.WorkItemTypeRepository, ts transaction.Support) *WorkitemtypeController {
	return &WorkitemtypeController{
		Controller:    service.NewController("WorkitemtypeController"),
		witRepository: witRepository,
		ts:            ts,
	}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	return transaction.Do(c.ts, func() error {
		res, err := c.witRepository.Load(ctx.Context, ctx.Name)
		if err != nil {
			switch err.(type) {
			case models.NotFoundError:
				log.Printf("not found, id=%s", ctx.Name)
				return goa.ErrNotFound(err.Error())
			default:
				return err
			}
		}
		return ctx.OK(res)
	})
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error()))
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error()))
	}
	return transaction.Do(c.ts, func() error {
		result, err := c.witRepository.List(ctx.Context, exp, start, &limit)
		if err != nil {
			return goa.ErrInternal(fmt.Sprintf("Error listing work item types: %s", err.Error()))
		}
		return ctx.OK(result)
	})
}
