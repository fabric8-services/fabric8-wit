package controller

import (
	"fmt"

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
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, ctx.Q, &offset, &limit)
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
