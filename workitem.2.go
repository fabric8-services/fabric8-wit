package main

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/goadesign/goa"
)

// Workitem2Controller implements the workitem.2 resource.
type Workitem2Controller struct {
	*goa.Controller
	wiRepository models.WorkItemRepository
	ts           transaction.Support
}

// NewWorkitem2Controller creates a workitem.2 controller.
func NewWorkitem2Controller(service *goa.Service, wiRepository models.WorkItemRepository, ts transaction.Support) *Workitem2Controller {
	return &Workitem2Controller{Controller: service.NewController("WorkitemController"), wiRepository: wiRepository, ts: ts}
}

// List runs the list action.
// Prev and Next links will be present only when there actually IS a next or previous page.
// Last will always be present. Total Item count needs to be computed from the "Last" link.
func (c *Workitem2Controller) List(ctx *app.ListWorkitem2Context) error {
	// Workitem2Controller_List: start_implement

	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error()))
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error()))
	}
	return transaction.Do(c.ts, func() error {
		result, c, err := c.wiRepository.List(ctx.Context, exp, start, &limit)
		count := int(c)
		if err != nil {
			return goa.ErrInternal(fmt.Sprintf("Error listing work items: %s", err.Error()))
		}
		var offset int
		if start != nil {
			offset = *start
		}
		response := app.WorkItemListResponse{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: float64(count)},
			Data:  result,
		}

		if offset > 0 {
			prevStart := offset - limit
			if prevStart < 0 {
				prevStart = 0
			}
			prev := fmt.Sprintf("%s?page=%d,%d", ctx.Request.URL.Path, prevStart, offset-prevStart)
			response.Links.Prev = &prev
		}
		nextStart := offset + len(result)
		if nextStart < count {
			if nextStart+limit >= count {
				// next is the last page
				next := fmt.Sprintf("%s?page=%d,%d", ctx.Request.URL.Path, nextStart, count-nextStart)
				response.Links.Next = &next
				response.Links.Last = &next
			} else {
				next := fmt.Sprintf("%s?page=%d,%d", ctx.Request.URL.Path, nextStart, limit)
				response.Links.Next = &next
				lastStart := offset + ((int(count)-offset-1)/limit)*limit
				last := fmt.Sprintf("%s?page=%d,%d", ctx.Request.URL.Path, lastStart, count-lastStart)
				response.Links.Last = &last
			}
		} else {
			// there is no next page
			last := fmt.Sprintf("%s?page=%d,%d", ctx.Request.URL.Path, offset, len(result))
			response.Links.Last = &last
		}

		return ctx.OK(&response)
	})

	// Workitem2Controller_List: end_implement
}
