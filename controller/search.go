package controller

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/search"
	"github.com/almighty/almighty-core/space"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

type searchConfiguration interface {
	GetHTTPAddress() string
}

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
	db            application.DB
	configuration searchConfiguration
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service, db application.DB, configuration searchConfiguration) *SearchController {
	if db == nil {
		panic("db must not be nil")
	}
	return &SearchController{Controller: service.NewController("SearchController"), db: db, configuration: configuration}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	var offset int
	var limit int

	offset, limit = computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	// ToDo : Keep URL registeration central somehow.
	hostString := ctx.RequestData.Host
	if hostString == "" {
		hostString = c.configuration.GetHTTPAddress()
	}
	urlRegexString := fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/list/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForListWI, urlRegexString)
	urlRegexString = fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/board/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForBoardWI, urlRegexString)

	return application.Transactional(c.db, func(appl application.Application) error {
		//return transaction.Do(c.ts, func() error {
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, ctx.Q, &offset, &limit, ctx.SpaceID)
		count := int(c)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("Error listing work items: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Error(ctx, map[string]interface{}{
					"err": err,
				}, "unable to list the work items")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		response := app.SearchWorkItemList{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx.RequestData, result),
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, "q="+ctx.Q)
		return ctx.OK(&response)
	})
}

// Spaces runs the space search action.
func (c *SearchController) Spaces(ctx *app.SpacesSearchContext) error {
	q := ctx.Q
	if q == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(fmt.Errorf("Empty search query not allowed")))
	} else if q == "*" {
		q = "" // Allow empty query if * specified
	}

	var result []space.Space
	var count int
	var err error

	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		var resultCount uint64
		result, resultCount, err = appl.Spaces().Search(ctx, &q, &offset, &limit)
		count = int(resultCount)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(fmt.Sprintf("Error listing spaces: %s", err.Error())))
			default:
				log.Error(ctx, map[string]interface{}{
					"query":  q,
					"offset": offset,
					"limit":  limit,
					"err":    err,
				}, "unable to list spaces")
				return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
			}
		}

		spaceData, err := ConvertSpacesFromModel(ctx.Context, c.db, ctx.RequestData, result)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		response := app.SearchSpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  spaceData,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, "q="+q)

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
		result, count, err = appl.Identities().Search(ctx, q, offset, limit)
		return err
	})
	if err != nil {
		fmt.Println(err)
		ctx.InternalServerError()
	}

	var users []*app.Users
	for i := range result {
		ident := result[i]
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
