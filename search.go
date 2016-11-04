package main

import (
	"fmt"
	"log"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/models"
	"github.com/goadesign/goa"
)

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
	db application.DB
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service, db application.DB) *SearchController {
	if db == nil {
		panic("db must not be nil")
	}
	return &SearchController{Controller: service.NewController("SearchController"), db: db}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	var offset int
	var limit int

	if ctx.PageOffset == nil {
		offset = 0
	} else {
		offset = int(*ctx.PageOffset)
	}

	if ctx.PageLimit == nil {
		limit = 100
	} else {
		limit = int(*ctx.PageLimit)
	}
	if offset < 0 {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Sprintf("offset must be >= 0, but is: %d", offset)))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		//return transaction.Do(c.ts, func() error {
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, *ctx.Q, &offset, &limit)
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

		response := app.SearchResponse{
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
			prev := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", ctx.Request.URL.Path, *ctx.Q, prevStart, realLimit)
			response.Links.Prev = &prev
		}

		// next link
		nextStart := offset + len(result)
		if nextStart < count {
			// we have a next link
			next := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", ctx.Request.URL.Path, *ctx.Q, nextStart, limit)
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
		first := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", ctx.Request.URL.Path, *ctx.Q, 0, firstEnd)
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
		last := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", ctx.Request.URL.Path, *ctx.Q, lastStart, realLimit)
		response.Links.Last = &last

		return ctx.OK(&response)
	})
}
