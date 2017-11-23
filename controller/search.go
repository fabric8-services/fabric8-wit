package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type searchConfiguration interface {
	GetHTTPAddress() string
	auth.ServiceConfiguration
}

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
	db            application.DB
	configuration searchConfiguration
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service, db application.DB, configuration searchConfiguration) *SearchController {
	return &SearchController{Controller: service.NewController("SearchController"), db: db, configuration: configuration}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {

	var offset int
	var limit int

	offset, limit = computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	// TODO: Keep URL registeration central somehow.
	hostString := ctx.Request.Host
	if hostString == "" {
		hostString = c.configuration.GetHTTPAddress()
	}
	urlRegexString := fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/list/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForListWI, urlRegexString)
	urlRegexString = fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/board/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForBoardWI, urlRegexString)

	if ctx.FilterExpression != nil {
		return application.Transactional(c.db, func(appl application.Application) error {
			result, cnt, err := appl.SearchItems().Filter(ctx.Context, *ctx.FilterExpression, ctx.FilterParentexists, &offset, &limit)
			count := int(cnt)
			if err != nil {
				cause := errs.Cause(err)
				switch cause.(type) {
				case errors.BadParameterError:
					jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx,
						goa.ErrBadRequest(fmt.Sprintf("error listing work items for expression '%s': %s", *ctx.FilterExpression, err)))
					return ctx.BadRequest(jerrors)
				default:
					log.Error(ctx, map[string]interface{}{
						"err":               err,
						"filter_expression": *ctx.FilterExpression,
					}, "unable to list the work items")
					jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(fmt.Sprintf("unable to list the work items: %s", err)))
					return ctx.InternalServerError(jerrors)
				}
			}

			hasChildren := workItemIncludeHasChildren(ctx, appl)
			includeParent := includeParentWorkItem(ctx, appl)
			response := app.SearchWorkItemList{
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
				Data:  ConvertWorkItems(ctx.Request, result, hasChildren, includeParent),
			}
			c.enrichWorkItemList(ctx, &response) // append parentWI in response
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count, "filter[expression]="+*ctx.FilterExpression)
			return ctx.OK(&response)
		})

	}
	return application.Transactional(c.db, func(appl application.Application) error {
		if ctx.Q == nil || *ctx.Q == "" {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx,
				goa.ErrBadRequest("empty search query not allowed"))
			return ctx.BadRequest(jerrors)
		}

		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, *ctx.Q, &offset, &limit, ctx.SpaceID)
		count := int(c)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				log.Error(ctx, map[string]interface{}{
					"err":        err,
					"expression": *ctx.Q,
				}, "unable to list the work items")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest(fmt.Sprintf("error listing work items for expression: %s: %s", *ctx.Q, err)))
				return ctx.BadRequest(jerrors)
			default:
				log.Error(ctx, map[string]interface{}{
					"err":        err,
					"expression": *ctx.Q,
				}, "unable to list the work items")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(fmt.Sprintf("unable to list the work items expression: %s: %s", *ctx.Q, err)))
				return ctx.InternalServerError(jerrors)
			}
		}

		response := app.SearchWorkItemList{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx.Request, result),
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count, "q="+*ctx.Q)
		return ctx.OK(&response)
	})
}

// Spaces runs the space search action.
func (c *SearchController) Spaces(ctx *app.SpacesSearchContext) error {
	q := ctx.Q
	if q == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("empty search query not allowed"))
	} else if q == "*" {
		q = "" // Allow empty query if * specified
	}

	var result []space.Space
	var count int
	var err error

	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		var resultCount uint64
		result, resultCount, err = appl.Spaces().Search(ctx, &q, &offset, &limit)
		count = int(resultCount)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				log.Error(ctx, map[string]interface{}{
					"query":  q,
					"offset": offset,
					"limit":  limit,
					"err":    err,
				}, "unable to list spaces")
				return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(fmt.Sprintf("error listing spaces for expression: %s: %s", q, err)))
			default:
				log.Error(ctx, map[string]interface{}{
					"query":  q,
					"offset": offset,
					"limit":  limit,
					"err":    err,
				}, "unable to list spaces")
				return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(fmt.Sprintf("unable to list spaces for expression: %s: %s", q, err)))
			}
		}

		spaceData, err := ConvertSpacesFromModel(ctx.Request, result, IncludeBacklogTotalCount(ctx.Context, c.db))
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		response := app.SearchSpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  spaceData,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count, "q="+q)

		return ctx.OK(&response)
	})
}

// Users runs the user search action.
func (c *SearchController) Users(ctx *app.UsersSearchContext) error {
	return proxy.RouteHTTP(ctx, c.configuration.GetAuthShortServiceHostName())
}

// Iterate over the WI list and read parent IDs
// Fetch and load Parent WI in the included list
func (c *SearchController) enrichWorkItemList(ctx *app.ShowSearchContext, res *app.SearchWorkItemList) {
	fetchInBatch := []uuid.UUID{}
	for _, wi := range res.Data {
		if wi.Relationships != nil && wi.Relationships.Parent != nil && wi.Relationships.Parent.Data != nil {
			parentID := wi.Relationships.Parent.Data.ID
			fetchInBatch = append(fetchInBatch, parentID)
		}
	}
	wis := []*workitem.WorkItem{}
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		wis, err = appl.WorkItems().LoadBatchByID(ctx, fetchInBatch)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"wis": wis,
			"err": err,
		}, "unable to load parent work items in batch: %s", fetchInBatch)
	}
	for _, ele := range wis {
		convertedWI := ConvertWorkItem(ctx.Request, *ele)
		res.Included = append(res.Included, *convertedWI)
	}
}

// Codebases runs the codebases search action.
func (c *SearchController) Codebases(ctx *app.CodebasesSearchContext) error {
	if ctx.URL == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("empty search query not allowed"))
	}
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		matchingCodebases, totalCount, err := appl.Codebases().SearchByURL(ctx, ctx.URL, &offset, &limit)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"url":    ctx.URL,
				"offset": offset,
				"limit":  limit,
				"err":    err,
			}, "unable to search codebases by URL")
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				return jsonapi.JSONErrorResponse(ctx, err)
			default:
				return jsonapi.JSONErrorResponse(ctx, err)
			}
		}
		// look-up the spaces of the matching codebases
		spaceIDs := make([]uuid.UUID, len(matchingCodebases))
		for i, c := range matchingCodebases {
			spaceIDs[i] = c.SpaceID
		}
		relatedSpaces, err := appl.Spaces().LoadMany(ctx, spaceIDs)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		// put all related spaces and associated owners in the `included` data
		includedData := make([]interface{}, len(relatedSpaces))
		for i, relatedSpace := range relatedSpaces {
			appSpace, err := ConvertSpaceFromModel(ctx.Request, relatedSpace)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			includedData[i] = *appSpace
		}
		codebasesData := ConvertCodebases(ctx.Request, matchingCodebases)
		response := app.CodebaseList{
			Links:    &app.PagingLinks{},
			Meta:     &app.CodebaseListMeta{TotalCount: totalCount},
			Data:     codebasesData,
			Included: includedData,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(matchingCodebases), offset, limit, totalCount, "url="+ctx.URL)
		return ctx.OK(&response)
	})
}
