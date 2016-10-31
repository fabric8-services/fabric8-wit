package main

import (
	"fmt"
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/goadesign/goa"
)

// Workitem2Controller implements the workitem.2 resource.
type Workitem2Controller struct {
	*goa.Controller
	db application.DB
}

// NewWorkitem2Controller creates a workitem.2 controller.
func NewWorkitem2Controller(service *goa.Service, db application.DB) *Workitem2Controller {
	return &Workitem2Controller{Controller: service.NewController("WorkitemController"), db: db}
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *Workitem2Controller) List(ctx *app.ListWorkitem2Context) error {
	// Workitem2Controller_List: start_implement

	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error())))
	}
	var offset int
	var limit int

	if ctx.PageOffset == nil {
		offset = 0
	} else {
		offset = int(*ctx.PageOffset)
	}
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("could not parse page offset: %s", err.Error())))
	}

	if ctx.PageLimit == nil {
		limit = 100
	} else {
		limit = int(*ctx.PageLimit)
	}
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("could not parse page limit: %s", err.Error())))
	}
	if offset < 0 {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("offset must be >= 0, but is: %d", offset)))
	}

	if limit <= 0 {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("limit must be > 0, but is: %d", limit)))
	}

	return application.Transactional(c.db, func(tx application.Application) error {
		result, c, err := tx.WorkItems().List(ctx.Context, exp, &offset, &limit)
		count := int(c)
		if err != nil {
			switch err := err.(type) {
			case models.BadParameterError:
				return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("Error listing work items: %s", err.Error())))
			default:
				log.Printf("Error listing work items: %s", err.Error())
				return ctx.InternalServerError()
			}
		}

		response := app.WorkItemListResponse{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: float64(count)},
			Data:  result,
		}

		// prev link
		if offset > 0 && count > 0 {
			var prevStart int
			// we do have a prev link
			if offset <= count {
				prevStart = offset - limit
			} else {
				// the first range that intersects the end of the useful range
				prevStart = offset - (((offset-count)/limit)+1)*limit
			}
			realLimit := limit
			if prevStart < 0 {
				// need to cut the range to start at 0
				realLimit = limit + prevStart
				prevStart = 0
			}
			prev := fmt.Sprintf("%s?page[offset]=%d,page[limit]=%d", ctx.Request.URL.Path, prevStart, realLimit)
			response.Links.Prev = &prev
		}

		// next link
		nextStart := offset + len(result)
		if nextStart < count {
			// we have a next link
			next := fmt.Sprintf("%s?page[offset]=%d,page[limit]=%d", ctx.Request.URL.Path, nextStart, limit)
			response.Links.Next = &next
		}

		// first link
		var firstEnd int
		if offset > 0 {
			firstEnd = offset % limit // this is where the second page starts
		} else {
			// offset == 0, first == current
			firstEnd = limit
		}
		first := fmt.Sprintf("%s?page[offset]=%d,page[limit]=%d", ctx.Request.URL.Path, 0, firstEnd)
		response.Links.First = &first

		// last link
		var lastStart int
		if offset < count {
			// advance some pages until touching the end of the range
			lastStart = offset + (((count - offset - 1) / limit) * limit)
		} else {
			// retreat at least one page until covering the range
			lastStart = offset - ((((offset - count) / limit) + 1) * limit)
		}
		realLimit := limit
		if lastStart < 0 {
			// need to cut the range to start at 0
			realLimit = limit + lastStart
			lastStart = 0
		}
		last := fmt.Sprintf("%s?page[offset]=%d,page[limit]=%d", ctx.Request.URL.Path, lastStart, realLimit)
		response.Links.Last = &last

		return ctx.OK(&response)
	})

	// Workitem2Controller_List: end_implement
}
