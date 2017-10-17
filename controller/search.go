package controller

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/asaskevich/govalidator"
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
	auth.AuthServiceConfiguration
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

func init() {
	urlRegexString := "(?P<domain>[^/]+)/(?P<org>[^/]+)/(?P<space>[^/]+)/(?P<path>plan/detail)/(?P<number>.*)"
	RegisterAsKnownURL(search.HostRegistrationKeyForListWI, urlRegexString)
	urlRegexString = "(?P<domain>[^/]+)/(?P<org>[^/]+)/(?P<space>[^/]+)/(?P<path>board/detail)/(?P<number>.*)"
	RegisterAsKnownURL(search.HostRegistrationKeyForBoardWI, urlRegexString)
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
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
		searchQuery, err := parseSearchString(ctx, *ctx.Q)
		if err != nil {
			log.Error(ctx, map[string]interface{}{"search_query": *ctx.Q}, "error in search query")
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("query", *ctx.Q))
		}
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, searchQuery, &offset, &limit, ctx.SpaceID)
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

// *************************
// Search query conversion
// *************************

// KnownURL has a regex string format URL and compiled regex for the same
type KnownURL struct {
	URLRegex          string         // regex for URL, Exposed to make the code testable
	compiledRegex     *regexp.Regexp // valid output of regexp.MustCompile()
	groupNamesInRegex []string       // Valid output of SubexpNames called on compliedRegex
}

/*
KnownURLs is set of KnownURLs will be used while searching on a URL
"Known" means that, our system understands the format of URLs
URLs in this slice will be considered while searching to match search string and decouple it into multiple searchable parts
e.g> Following example defines work-item-detail-page URL on client side, with its compiled version
knownURLs["work-item-details"] = KnownURL{
URLRegex:      `^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/work-item/list/detail/)(?P<id>\d*)`,
compiledRegex: regexp.MustCompile(`^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/work-item/list/detail/)(?P<id>\d*)`),
groupNamesInRegex: []string{"protocol", "domain", "path", "id"}
}
above url will be decoupled into two parts "ID:* | domain+path+id:*" while performing search query
*/
var knownURLs = make(map[string]KnownURL)
var knownURLLock sync.RWMutex

// RegisterAsKnownURL appends to KnownURLs
func RegisterAsKnownURL(name, urlRegex string) {
	compiledRegex := regexp.MustCompile(urlRegex)
	groupNames := compiledRegex.SubexpNames()
	knownURLLock.Lock()
	defer knownURLLock.Unlock()
	knownURLs[name] = KnownURL{
		URLRegex:          urlRegex,
		compiledRegex:     regexp.MustCompile(urlRegex),
		groupNamesInRegex: groupNames,
	}
}

// GetAllRegisteredURLs returns all known URLs
func GetAllRegisteredURLs() map[string]KnownURL {
	return knownURLs
}

/*
isKnownURL compares with registered URLs in our system.
Iterates over knownURLs and finds out most relevant matching pattern.
If found, it returns true along with "name" of the KnownURL
*/
func isKnownURL(url string) (bool, *string) {
	// should check on all system's known URLs
	var mostReleventMatchCount int
	var mostReleventMatchName string
	for name, known := range knownURLs {
		match := known.compiledRegex.FindStringSubmatch(url)
		if len(match) > mostReleventMatchCount {
			mostReleventMatchCount = len(match)
			mostReleventMatchName = name
		}
	}
	if mostReleventMatchName == "" {
		return false, nil
	}
	return true, &mostReleventMatchName
}

// parseSearchString accepts a raw string and generates a Keywords object
func parseSearchString(ctx context.Context, rawSearchString string) (search.Keywords, error) {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.Trim(rawSearchString, "/") // get rid of trailing slashes
	rawSearchString = strings.Trim(rawSearchString, "\"")
	parts := strings.Fields(rawSearchString)
	var res search.Keywords
	for _, part := range parts {
		// QueryUnescape is required in case of encoded url strings.
		// And does not harm regular search strings
		// but this processing is required because at this moment, we do not know if
		// search input is a regular string or a URL

		part, err := url.QueryUnescape(part)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"part": part,
			}, "unable to escape url!")
		}
		// IF part is for search with number:1234
		// TODO: need to find out the way to use ID fields.
		if strings.HasPrefix(part, "number:") {
			res.Number = append(res.Number, strings.TrimPrefix(part, "number:")+":*A")
		} else if strings.HasPrefix(part, "type:") {
			typeIDStr := strings.TrimPrefix(part, "type:")
			if len(typeIDStr) == 0 {
				log.Error(ctx, map[string]interface{}{}, "type: part is empty")
				return res, errors.NewBadParameterError("Type ID must not be empty", part)
			}
			typeID, err := uuid.FromString(typeIDStr)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"err":    err,
					"typeID": typeIDStr,
				}, "failed to convert type ID string to UUID")
				return res, errors.NewBadParameterError("failed to parse type ID string as UUID", typeIDStr)
			}
			res.WorkItemTypes = append(res.WorkItemTypes, typeID)
		} else if govalidator.IsURL(part) {
			log.Warn(ctx, map[string]interface{}{"url": part}, "found a URL in the query string")
			part := strings.ToLower(part)
			part = trimProtocolFromURLString(part)
			searchQueryFromURL, _ := getSearchQueryFromURLString(part)
			log.Debug(ctx, map[string]interface{}{"url": part, "search_query": searchQueryFromURL}, "found a URL in the query string")
			res.Words = append(res.Words, searchQueryFromURL)
		} else {
			part := strings.ToLower(part)
			part = sanitizeURL(part)
			res.Words = append(res.Words, part+":*")
		}
	}
	log.Info(nil, nil, "Search keywords: '%s' -> %v", rawSearchString, res)
	return res, nil
}

// sanitizeURL does cleaning of URL
// returns DB friendly string
// Trims protocol and escapes ":"
func sanitizeURL(urlString string) string {
	trimmedURL := trimProtocolFromURLString(urlString)
	return escapeCharFromURLString(trimmedURL)
}

func trimProtocolFromURLString(urlString string) string {
	urlString = strings.TrimPrefix(urlString, `http://`)
	urlString = strings.TrimPrefix(urlString, `https://`)
	return urlString
}

func escapeCharFromURLString(urlString string) string {
	// Replacer will escape `:` and `)` `(`.
	var replacer = strings.NewReplacer(":", "\\:", "(", "\\(", ")", "\\)")
	return replacer.Replace(urlString)
}

/*
getSearchQueryFromURLString gets a url string and checks if that matches with any of known urls.
Respectively it will return a string that can be directly used in search query
e.g>
Unknown url : www.google.com then response = "www.google.com:*"
Known url : almighty.io/detail/500 then response = "500:* | almighty.io/detail/500"
Also returns a bool to indicate if the given URL was a known form
*/
func getSearchQueryFromURLString(url string) (string, bool) {
	known, patternName := isKnownURL(url)
	if known {
		// this url is known to system
		return getSearchQueryFromURLPattern(*patternName, url), true
	}
	// any URL other than our system's
	// return url without protocol
	return sanitizeURL(url) + ":*", false
}

/*
getSearchQueryFromURLPattern takes
patternName - name of the KnownURL
stringToMatch - search string
Finds all string match for given pattern
Iterates over pattern's groupNames and loads respective values into result
*/
func getSearchQueryFromURLPattern(patternName, stringToMatch string) string {
	pattern := knownURLs[patternName]
	// TODO : handle case for 0 matches
	match := pattern.compiledRegex.FindStringSubmatch(stringToMatch)
	result := make(map[string]string)
	// result will hold key-value for groupName to its value
	// e.g> "domain": "demo.almighty.io", "id": 200
	for i, name := range pattern.groupNamesInRegex {
		if i == 0 {
			continue
		}
		if i > len(match)-1 {
			result[name] = ""
		} else {
			result[name] = match[i]
		}
	}
	// first value from FindStringSubmatch is always full input itself, hence ignored
	// Join rest of the tokens to make query like "demo.almighty.io/work-item/list/detail/100"
	if len(match) > 1 {
		searchQueryString := strings.Join(match[1:], "/")
		searchQueryString = strings.Replace(searchQueryString, ":", "\\:", -1)
		// need to escape ":" because this string will go as an input to tsquery
		searchQueryString = fmt.Sprintf("%s:*", searchQueryString)
		if result["number"] != "" {
			// Look for pattern's ID field, if exists update searchQueryString
			// `*A` is used to add some weight to the work item number in the search results.
			// See https://www.postgresql.org/docs/9.6/static/textsearch-controls.html
			searchQueryString = fmt.Sprintf("(%v:*A | %v)", result["number"], searchQueryString)
			// searchQueryString = "(" + result["id"] + ":*" + " | " + searchQueryString + ")"
		}
		return searchQueryString
	}
	if len(match) > 0 {
		return match[0] + ":*"
	}
	return ""
}
