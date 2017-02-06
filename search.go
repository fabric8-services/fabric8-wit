package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/search"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/almighty/almighty-core/space"
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
		offsetValue, err := strconv.Atoi(*ctx.PageOffset)
		if err != nil {
			offset = 0
		} else {
			offset = offsetValue
		}
	}

	if ctx.PageLimit == nil {
		limit = 100
	} else {
		limit = *ctx.PageLimit
	}
	if offset < 0 {
		//jerrors, _ := jsonapi.ErrorToJSONAPIErrors(models.NewBadParameterError(fmt.Sprintf("offset must be >= 0, but is: %d", offset)))
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("offset must be >= 0, but is: %d", offset)))
		return ctx.BadRequest(jerrors)
	}

	// ToDo : Keep URL registeration central somehow.
	hostString := ctx.RequestData.Host
	if hostString == "" {
		hostString = configuration.GetHTTPAddress()
	}
	urlRegexString := fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/list/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForListWI, urlRegexString)
	urlRegexString = fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/board/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForBoardWI, urlRegexString)

	return application.Transactional(c.db, func(appl application.Application) error {
		//return transaction.Do(c.ts, func() error {
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, ctx.Q, &offset, &limit)
		count := int(c)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing work items: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error listing work items: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		response := app.SearchWorkItemList{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx.RequestData, result),
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
			prev := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", buildAbsoluteURL(ctx.RequestData), ctx.Q, prevStart, realLimit)
			response.Links.Prev = &prev
		}

		// next link
		nextStart := offset + len(result)
		if nextStart < count {
			// we have a next link
			next := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", buildAbsoluteURL(ctx.RequestData), ctx.Q, nextStart, limit)
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
		first := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", buildAbsoluteURL(ctx.RequestData), ctx.Q, 0, firstEnd)
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
		last := fmt.Sprintf("%s?q=%s&page[offset]=%d&page[limit]=%d", buildAbsoluteURL(ctx.RequestData), ctx.Q, lastStart, realLimit)
		response.Links.Last = &last

		return ctx.OK(&response)
	})
}

// Users runs the user search action.
func (c *SearchController) Spaces(ctx *app.SpacesSearchContext) error {
	q := ctx.Q
	if q == "" {
		jerror, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Errorf("Empty search query not allowed")))
		return ctx.BadRequest(jerror)
	} else if q == "*" {
		q = "" // Allow empty query if * specified
	}

	var result []*space.Space
	var count int
	var err error

	offset, limit, err := paginationOptions(ctx.PageOffset, ctx.PageLimit)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err))
		return ctx.BadRequest(jerrors)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		var resultCount uint64
		result, resultCount, err = appl.Spaces().Search(ctx, &q, &offset, &limit)
		count = int(resultCount)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing spaces: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Printf("Error listing spaces: %s", err.Error())
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		pagingLinks := paginationLinks(
			buildAbsoluteURL(ctx.RequestData),
			"q="+q,
			offset,
			limit,
			count,
			len(result))

		response := app.SearchSpaceList{
			Links: &pagingLinks,
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  ConvertSpaces(ctx.RequestData, result),
		}

		return ctx.OK(&response)
	})
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
func paginationOptions(pageOffset *string, pageLimit *int) (int, int, error) {
	var offset int
	var limit int
	var err error

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
		err = fmt.Errorf("offset must be >= 0, but is: %d", offset)
	}
	return offset, limit, err
}
