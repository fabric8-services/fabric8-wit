package controller

import (
	"fmt"
	"sort"

	"github.com/fabric8-services/fabric8-wit/id"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

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

// WorkItemPtrSlice exists in order to allow sorting results in a search
// response.
type WorkItemPtrSlice []*app.WorkItem

// Len is the number of elements in the collection.
func (a WorkItemPtrSlice) Len() int {
	return len(a)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (a WorkItemPtrSlice) Less(i, j int) bool {
	title1, foundTitle1 := a[i].Attributes[workitem.SystemTitle]
	title2, foundTitle2 := a[j].Attributes[workitem.SystemTitle]
	if foundTitle1 && foundTitle2 {
		t1, cast1Ok := title1.(string)
		t2, cast2Ok := title2.(string)
		if cast1Ok && cast2Ok {
			return t1 < t2
		}
	}
	return a[i].ID.String() < a[j].ID.String()
}

// Swap swaps the elements with indexes i and j.
func (a WorkItemPtrSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Ensure WorkItemPtrSlice implements the sort.Interface
var _ sort.Interface = WorkItemPtrSlice{}
var _ sort.Interface = (*WorkItemPtrSlice)(nil)

// WorkItemInterfaceSlice exists in order to allow sorting results in a search
// response.
type WorkItemInterfaceSlice []interface{}

// Len is the number of elements in the collection.
func (a WorkItemInterfaceSlice) Len() int {
	return len(a)
}

// Less reports whether the element with index i should sort before the element
// with index j.
//
// NOTE: For now we assume that included elements in the search response are
// only work items. We can handle work items as pointers or objects. Both
// options are possible.
func (a WorkItemInterfaceSlice) Less(i, j int) bool {
	var x, y *app.WorkItem

	switch v := a[i].(type) {
	case app.WorkItem:
		x = &v
	case *app.WorkItem:
		x = v
	}

	switch v := a[j].(type) {
	case app.WorkItem:
		y = &v
	case *app.WorkItem:
		y = v
	}

	if x == nil || y == nil {
		return false
	}

	title1, foundTitle1 := x.Attributes[workitem.SystemTitle]
	title2, foundTitle2 := y.Attributes[workitem.SystemTitle]
	if foundTitle1 && foundTitle2 {
		t1, cast1Ok := title1.(string)
		t2, cast2Ok := title2.(string)
		if cast1Ok && cast2Ok {
			return t1 < t2
		}
	}
	return x.ID.String() < y.ID.String()
}

// Swap swaps the elements with indexes i and j.
func (a WorkItemInterfaceSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Ensure WorkItemInterfaceSlice implements the sort.Interface
var _ sort.Interface = WorkItemInterfaceSlice{}
var _ sort.Interface = (*WorkItemInterfaceSlice)(nil)

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
			result, cnt, ancestors, childLinks, err := appl.SearchItems().Filter(ctx.Context, *ctx.FilterExpression, ctx.FilterParentexists, &offset, &limit)
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

			matchingWorkItemIDs := make(id.Slice, len(result))
			for i, wi := range result {
				matchingWorkItemIDs[i] = wi.ID
			}

			hasChildren := workItemIncludeHasChildren(ctx, appl, childLinks)
			includeParent := includeParentWorkItem(ctx, ancestors, childLinks)
			response := app.SearchWorkItemList{
				Links: &app.PagingLinks{},
				Meta: &app.WorkItemListResponseMeta{
					TotalCount: count,
				},
				Data: ConvertWorkItems(ctx.Request, result, hasChildren, includeParent),
			}
			c.enrichWorkItemList(ctx, ancestors, matchingWorkItemIDs, childLinks, &response, hasChildren) // append parentWI and ancestors (if not empty) in response
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count, "filter[expression]="+*ctx.FilterExpression)

			// Sort "data" by name or ID if no title given
			var data WorkItemPtrSlice = response.Data
			sort.Sort(data)
			response.Data = data

			// Sort work items in the "included" array by ID or title
			var included WorkItemInterfaceSlice = response.Included
			sort.Sort(included)
			response.Included = included

			// build up list of sorted ancestor IDs from already sorted work items
			ancestorIDs := ancestors.GetDistinctAncestorIDs().ToMap()
			sortedAncestorIDs := make(id.Slice, len(ancestorIDs))
			i := 0
			for _, wi := range response.Data {
				if len(ancestorIDs) <= 0 {
					break
				}
				_, ok := ancestorIDs[*wi.ID]
				if ok {
					sortedAncestorIDs[i] = *wi.ID
					i++
					delete(ancestorIDs, *wi.ID)
				}
			}
			for _, ifObj := range response.Included {
				if len(ancestorIDs) <= 0 {
					break
				}
				var wi *app.WorkItem
				switch v := ifObj.(type) {
				case app.WorkItem:
					wi = &v
				case *app.WorkItem:
					wi = v
				default:
					continue
				}
				_, ok := ancestorIDs[*wi.ID]
				if ok {
					sortedAncestorIDs[i] = *wi.ID
					i++
					delete(ancestorIDs, *wi.ID)
				}
			}
			response.Meta.AncestorIDs = sortedAncestorIDs

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
func (c *SearchController) enrichWorkItemList(ctx *app.ShowSearchContext, ancestors link.AncestorList, matchingIDs id.Slice, childLinks link.WorkItemLinkList, res *app.SearchWorkItemList, hasChildren WorkItemConvertFunc) {

	parentIDs := id.Slice{}
	for _, wi := range res.Data {
		if wi.Relationships != nil && wi.Relationships.Parent != nil && wi.Relationships.Parent.Data != nil {
			parentIDs = append(parentIDs, wi.Relationships.Parent.Data.ID)
		}
	}

	// Also append the ancestors not already included in the parent list
	fetchInBatch := parentIDs
	fetchInBatch.Add(ancestors.GetDistinctAncestorIDs().Diff(parentIDs))

	// Append direct children of matching work items that have a child that is
	// also a match and are not yet in the list.
	nonMatchingChildItems := childLinks.GetDistinctListOfTargetIDs(link.SystemWorkItemLinkTypeParentChildID)
	fetchInBatch.Add(fetchInBatch.Diff(nonMatchingChildItems))

	// Eliminate work items already in the search
	fetchInBatch = fetchInBatch.Sub(matchingIDs)

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
		convertedWI := ConvertWorkItem(ctx.Request, *ele, hasChildren, includeParentWorkItem(ctx, ancestors, childLinks))
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
