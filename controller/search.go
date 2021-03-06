package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-common/id"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/rest/proxy"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

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

// WorkitemsCSV converts workitems to CSV format.
func (c *SearchController) WorkitemsCSV(ctx *app.WorkitemsCSVSearchContext) error {
	if ctx.FilterExpression == nil {
		return goa.ErrBadRequest(fmt.Sprintf("bad parameter error exporting work items as CSV: param 'filter[expression]' missing"))
	}
	if ctx.PageLimit != nil && *(ctx.PageLimit) > 1000 {
		// the window size is too large
		return goa.ErrBadRequest("maximum limit value is 1000")
	}
	// set page size from query or use default
	limit := 100
	if ctx.PageLimit != nil {
		// we add an overflow indicator element
		limit = *(ctx.PageLimit) + 1
	}
	offset := 0
	if ctx.PageOffset != nil {
		offset = *(ctx.PageOffset)
	}
	// retrieve the first work item window to get a reference for the WITs
	wiResult, childsResult, parentsResult, err := getWorkItemsByFilterExpression(*ctx, c.db, *ctx.FilterExpression, ctx.FilterParentexists, &offset, &limit)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("error searching work items for expression '%s': %s", *ctx.FilterExpression, err))
	}
	// set header data, adding timestamp to the filename
	currentTime := time.Now().UTC()
	timeStr := currentTime.Format(time.RFC3339)
	ctx.ResponseData.Header().Set("Content-Disposition", "attachment; filename='workitems-"+timeStr+".csv'")
	// see if we have a non-empty result set
	if len(wiResult) == 0 {
		// empty result, nothing matched the filter expression, we're done
		return ctx.OK([]byte(""))
	}
	// retrieve the spaceID from the returned first result entry
	spaceID := wiResult[0].SpaceID
	// retieve all WITs for this space
	var wits []workitem.WorkItemType
	err = application.Transactional(c.db, func(appl application.Application) error {
		var err error
		thisSpace, err := appl.Spaces().Load(ctx.Context, spaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":     err,
				"spaceID": spaceID,
			}, "unable to retrieve space")
			return errs.Wrapf(err, "error retrieving space for spaceID: %s", spaceID.String())
		}
		spaceTemplateID := thisSpace.SpaceTemplateID
		wits, err = appl.WorkItemTypes().List(ctx.Context, spaceTemplateID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":     err,
				"spaceID": spaceID,
			}, "unable to retrieve work item types")
			return errs.Wrapf(err, "error retrieving work item types for spaceID: %s", spaceID.String())
		}
		return nil
	})
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("error retrieving work item types for expression '%s': %s", *ctx.FilterExpression, err))
	}
	// the cache used for caching id to number resolve results
	idNumberCache := make(map[string]string)
	if ctx.PageLimit != nil {
		// if the overflow entry is in the result set, then there is more than this window, add a note to the bottom of the returned list
		moreMessage := ""
		if len(wiResult) >= limit {
			// there is an overflow, slice off the overflow item and add the message
			wiResult = wiResult[:len(wiResult)-1]
			moreMessage = "\nWIT_NOTE_MORE: There are more result entries. You may want to narrow down your query or use paging to retrieve more results."
		}
		// Convert them to CSV format the return value contains the final CSV
		wisCSV, _, err := ConvertWorkItemsToCSV(ctx.Context, c.db, wits, wiResult, childsResult, parentsResult, idNumberCache, true)
		if err != nil {
			return goa.ErrBadRequest(fmt.Sprintf("error converting work items to output format for expression '%s': %s", *ctx.FilterExpression, err))
		}
		// add the note to the bottom of the result csv (if there is a note)
		wisCSV = wisCSV + moreMessage
		// return to client
		return ctx.OK([]byte(wisCSV))
	}
	// we want to retrieve everything, first serialize the work items we already got
	wisCSV, _, err := ConvertWorkItemsToCSV(ctx.Context, c.db, wits, wiResult, childsResult, parentsResult, idNumberCache, true)
	if err != nil {
		return goa.ErrBadRequest(fmt.Sprintf("error converting work items to output format for expression '%s': %s", *ctx.FilterExpression, err))
	}
	// write them out to the client
	ctx.ResponseWriter.Write([]byte(wisCSV))
	// page over the database, convert and send in chunks to the client
	var wiResultWindow []workitem.WorkItem
	var childLinksWindow link.WorkItemLinkList
	var parentWindow link.AncestorList
	// we already fetched limit entries, so set offset
	offset = offset + limit
	// now iterate as long as the returned item count reaches the limit
	for {
		wiResultWindow, childLinksWindow, parentWindow, err = getWorkItemsByFilterExpression(ctx.Context, c.db, *ctx.FilterExpression, ctx.FilterParentexists, ptr.Int(offset), ptr.Int(limit))
		if err != nil {
			return goa.ErrBadRequest(fmt.Sprintf("error retrieving work item types for expression '%s': %s", *ctx.FilterExpression, err))
		}
		wisCSV, _, err = ConvertWorkItemsToCSV(ctx.Context, c.db, wits, wiResultWindow, childLinksWindow, parentWindow, idNumberCache, false)
		if err != nil {
			return goa.ErrBadRequest(fmt.Sprintf("error converting work items to output format for expression '%s': %s", *ctx.FilterExpression, err))
		}
		ctx.ResponseWriter.Write([]byte(wisCSV))
		if len(wiResultWindow) < limit {
			break
		}
		offset = offset + len(wiResultWindow)
	}
	// finally, return the result
	return ctx.OK(nil)
}

// getWorkItemsByFilterExpression retrieves Work Items, children and parents for a given expression and parameters
func getWorkItemsByFilterExpression(ctx context.Context, db application.DB, filterExpression string, filterParentexists *bool, offset *int, limit *int) ([]workitem.WorkItem, link.WorkItemLinkList, link.AncestorList, error) {
	var result []workitem.WorkItem
	var childLinks link.WorkItemLinkList
	var parents link.AncestorList
	err := application.Transactional(db, func(appl application.Application) error {
		var err error
		// add tree option to query if not already present
		var reqMap map[string]interface{}
		err = json.Unmarshal([]byte(filterExpression), &reqMap)
		if err != nil {
			return errs.Errorf("error unmarshalling query expression for CSV filtering: %s", filterExpression)
		}
		if _, ok := reqMap["$OPTS"]; !ok {
			reqMap["$OPTS"] = make(map[string]interface{})
		}
		// Set "tree-view" to true. We always want tree-view to be enabled
		(reqMap["$OPTS"].(map[string]interface{}))["tree-view"] = true
		updatedFilterExpression, err := json.Marshal(reqMap)
		if err != nil {
			return errs.Errorf("error adding tree opt to query expression for CSV filtering: %s", filterExpression)
		}
		// execute query
		result, _, parents, childLinks, err = appl.SearchItems().Filter(ctx, string(updatedFilterExpression), filterParentexists, offset, limit)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":               err,
				"filter_expression": filterExpression,
			}, "unable to list the work items")
			return errs.Wrapf(err, "error executing filter expression for CSV filtering: %s", filterExpression)
		}
		return nil
	})
	return result, childLinks, parents, err
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
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
		var result []workitem.WorkItem
		var count int
		var ancestors link.AncestorList
		var childLinks link.WorkItemLinkList
		err := application.Transactional(c.db, func(appl application.Application) error {
			var err error
			result, count, ancestors, childLinks, err = appl.SearchItems().Filter(ctx.Context, *ctx.FilterExpression, ctx.FilterParentexists, &offset, &limit)
			if err != nil {
				cause := errs.Cause(err)
				switch cause.(type) {
				case errors.BadParameterError:
					return goa.ErrBadRequest(fmt.Sprintf("error listing work items for expression '%s': %s", *ctx.FilterExpression, err))
				default:
					log.Error(ctx, map[string]interface{}{
						"err":               err,
						"filter_expression": *ctx.FilterExpression,
					}, "unable to list the work items")
					return goa.ErrInternal(fmt.Sprintf("unable to list the work items: %s", err))
				}
			}
			return nil

		})
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		matchingWorkItemIDs := make(id.Slice, len(result))
		for i, wi := range result {
			matchingWorkItemIDs[i] = wi.ID
		}
		hasChildren := workItemIncludeHasChildren(ctx, c.db, childLinks)
		includeParent := includeParentWorkItem(ctx, ancestors, childLinks)
		// Load all work item types
		wits, err := loadWorkItemTypesFromArr(ctx.Context, c.db, result)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "failed to load work item types"))
		}

		wis, err := ConvertWorkItems(ctx.Request, wits, result, hasChildren, includeParent)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		response := app.SearchWorkItemList{
			Links: &app.PagingLinks{},
			Meta: &app.WorkItemListResponseMeta{
				TotalCount: count,
			},
			Data: wis,
		}
		err = c.enrichWorkItemList(ctx, ancestors, matchingWorkItemIDs, childLinks, &response, hasChildren) // append parentWI and ancestors (if not empty) in response
		if err != nil {
			return errs.Wrap(err, "failed to enrich work item list")
		}
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
	}
	var result []workitem.WorkItem
	var count int
	err := application.Transactional(c.db, func(appl application.Application) error {
		if ctx.Q == nil || *ctx.Q == "" {
			return goa.ErrBadRequest("empty search query not allowed")
		}
		var err error
		result, count, err = appl.SearchItems().SearchFullText(ctx.Context, *ctx.Q, &offset, &limit, ctx.SpaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":        err,
				"expression": *ctx.Q,
			}, "unable to list the work items")
			return errs.Wrapf(err, "unable to list the work items for expression: %s", *ctx.Q)
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	wits, err := loadWorkItemTypesFromArr(ctx.Context, c.db, result)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	wis, err := ConvertWorkItems(ctx.Request, wits, result)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	response := app.SearchWorkItemList{
		Links: &app.PagingLinks{},
		Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
		Data:  wis,
	}
	setPagingLinks(response.Links, buildAbsoluteURL(ctx.Request), len(result), offset, limit, count, "q="+*ctx.Q)
	return ctx.OK(&response)
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
	err = application.Transactional(c.db, func(appl application.Application) error {
		result, count, err = appl.Spaces().Search(ctx, &q, &offset, &limit)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"query":  q,
				"offset": offset,
				"limit":  limit,
				"err":    err,
			}, "unable to list spaces")
		}
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
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
}

// Users runs the user search action.
func (c *SearchController) Users(ctx *app.UsersSearchContext) error {
	return proxy.RouteHTTP(ctx, c.configuration.GetAuthShortServiceHostName())
}

// Iterate over the WI list and read parent IDs
// Fetch and load Parent WI in the included list
func (c *SearchController) enrichWorkItemList(ctx *app.ShowSearchContext, ancestors link.AncestorList, matchingIDs id.Slice, childLinks link.WorkItemLinkList, res *app.SearchWorkItemList, hasChildren WorkItemConvertFunc) error {
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
		return errs.Wrapf(err, "unable to load work item items in batch: %s", fetchInBatch)
	}

	for _, ele := range wis {
		wit, err := c.db.WorkItemTypes().Load(ctx.Context, ele.Type)
		if err != nil {
			return errs.Wrapf(err, "failed to load work item type: %s", ele.Type)
		}
		convertedWI, err := ConvertWorkItem(ctx.Request, *wit, *ele, hasChildren, includeParentWorkItem(ctx, ancestors, childLinks))
		if err != nil {
			return errs.WithStack(err)
		}
		res.Included = append(res.Included, *convertedWI)
	}
	return nil
}

// Codebases runs the codebases search action.
func (c *SearchController) Codebases(ctx *app.CodebasesSearchContext) error {
	if ctx.URL == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("empty search query not allowed"))
	}
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	var matchingCodebases []codebase.Codebase
	var relatedSpaces []space.Space
	var totalCount int
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		url, err := convertGithubURL(ctx.URL)
		if err != nil {
			return nil
		}
		matchingCodebases, totalCount, err = appl.Codebases().SearchByURL(ctx, url, &offset, &limit)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"url":    ctx.URL,
				"offset": offset,
				"limit":  limit,
				"err":    err,
			}, "unable to search codebases by URL")
			return err
		}
		// look-up the spaces of the matching codebases
		spaceIDs := make([]uuid.UUID, len(matchingCodebases))
		for i, c := range matchingCodebases {
			spaceIDs[i] = c.SpaceID
		}
		relatedSpaces, err = appl.Spaces().LoadMany(ctx, spaceIDs)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	// put all related spaces and associated owners in the `included` data
	includedData := make([]interface{}, len(relatedSpaces))
	for i, relatedSpace := range relatedSpaces {
		appSpace, err := ConvertSpaceFromModel(ctx.Request, relatedSpace)
		if err != nil {
			return err
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
}

func convertGithubURL(urlRaw string) (string, error) {
	// if the URL is https then we don't need to make any changes
	if strings.HasPrefix(urlRaw, "https") {
		return urlRaw, nil
	}

	validGit := regexp.MustCompile(`^(https|git|http)(:\/\/|@)([^\/:]+)[\/:]([^\/:]+)\/(.+)`)
	if !validGit.MatchString(urlRaw) {
		return "", fmt.Errorf("invalid URL: %v", urlRaw)
	}
	components := validGit.FindStringSubmatch(urlRaw)
	l := len(components)

	repo := components[l-1]
	if !strings.HasSuffix(repo, ".git") {
		repo = repo + ".git"
	}
	user := components[l-2]
	host := components[l-3]

	rawURL := path.Join(host, user, repo)

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.Scheme = "https"

	return u.String(), nil
}
