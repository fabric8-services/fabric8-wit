package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/almighty/almighty-core/account"
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

	offset, limit := paginationOptions(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, ctx.Q, &offset, &limit)
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

		pagingLinks := paginationLinks(
			buildAbsoluteURL(ctx.RequestData),
			"q="+ctx.Q,
			offset,
			limit,
			count,
			len(result))

		response := app.SearchResponse{
			Links: &pagingLinks,
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  result,
		}
		return ctx.OK(&response)
	})

}

// Users runs the user search action.
func (c *SearchController) Users(ctx *app.UsersSearchContext) error {

	q := ctx.Q
	if q == "" {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Errorf("Empty search query not allowed")))
	} else if q == "*" {
		q = "" // Allow empty query if * specified
	}

	var result []account.Identity
	var count int
	var err error

	offset, limit := paginationOptions(ctx.PageOffset, ctx.PageLimit)

	err = application.Transactional(c.db, func(appl application.Application) error {
		result, count, err = appl.Identities().SearchByFullName(ctx, q, offset, limit)
		return err
	})
	if err != nil {
		fmt.Println(err)
		ctx.InternalServerError()
	}

	var users []*app.Users
	for _, ident := range result {
		id := ident.ID.String()
		users = append(users, &app.Users{
			Type: "users",
			ID:   &id,
			Attributes: &app.UserAttributes{
				Fullname: &ident.FullName,
				ImageURL: &ident.ImageURL,
			},
		})
	}

	pagingLinks := paginationLinks(
		buildAbsoluteURL(ctx.RequestData),
		"q="+ctx.Q,
		offset,
		limit,
		count,
		len(result))

	response := app.SearchResponseUsers{
		Data:  users,
		Links: &pagingLinks,
		Meta:  map[string]interface{}{"total-count": count},
	}

	return ctx.OK(&response)
}

func paginationLinks(path, args string, offset, limit, count, length int) app.PagingLinks {
	links := app.PagingLinks{}

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
		prev := fmt.Sprintf("%s?%s&page[offset]=%d&page[limit]=%d", path, args, prevStart, realLimit)
		links.Prev = &prev
	}

	// next link
	nextStart := offset + length
	if nextStart < count {
		// we have a next link
		next := fmt.Sprintf("%s?%s&page[offset]=%d&page[limit]=%d", path, args, nextStart, limit)
		links.Next = &next
	}

	// first link
	var firstEnd int
	if offset > 0 {
		firstEnd = offset % limit // this is where the second page starts
	} else {
		// offset == 0, first == current
		firstEnd = limit
	}
	first := fmt.Sprintf("%s?%s&page[offset]=%d&page[limit]=%d", path, args, 0, firstEnd)
	links.First = &first

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
	last := fmt.Sprintf("%s?%s&page[offset]=%d&page[limit]=%d", path, args, lastStart, realLimit)
	links.Last = &last

	return links
}

// paginationOptions parses and defaults offset/limit numbers
func paginationOptions(pageOffset *string, pageLimit *int) (int, int) {
	var offset int
	var limit int

	if pageOffset == nil {
		offset = 0
	} else {
		offsetValue, err := strconv.Atoi(*pageOffset)
		if err != nil {
			offset = 0
		} else {
			offset = offsetValue
		}
	}

	if pageLimit == nil {
		limit = 100
	} else {
		limit = *pageLimit
	}
	if offset < 0 {
		offset = 0
	}
	if limit > 100 {
		limit = 100
	}
	return offset, limit
}
