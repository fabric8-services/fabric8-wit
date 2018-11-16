package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/fabric8-services/fabric8-wit/gormsupport"

	"github.com/fabric8-services/fabric8-wit/closeable"

	"github.com/asaskevich/govalidator"
	"github.com/davecgh/go-spew/spew"
	"github.com/fabric8-services/fabric8-common/id"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// KnownURL registration key constants
const (
	HostRegistrationKeyForListWI  = "work-item-list-details"
	HostRegistrationKeyForBoardWI = "work-item-board-details"

	EQ     = "$EQ"
	NE     = "$NE"
	AND    = "$AND"
	OR     = "$OR"
	NOT    = "$NOT"
	IN     = "$IN"
	SUBSTR = "$SUBSTR"
	OPTS   = "$OPTS"

	// This is the replacement for $WITGROUP.
	TypeGroupName = "typegroup.name"

	OptParentExistsKey = "parent-exists"
	OptTreeViewKey     = "tree-view"
)

// GormSearchRepository provides a Gorm based repository
type GormSearchRepository struct {
	db   *gorm.DB
	witr *workitem.GormWorkItemTypeRepository
}

// NewGormSearchRepository creates a new search repository
func NewGormSearchRepository(db *gorm.DB) *GormSearchRepository {
	return &GormSearchRepository{db, workitem.NewWorkItemTypeRepository(db)}
}

func generateSearchQuery(q string) (string, error) {
	return q, nil
}

//searchKeyword defines how a decomposed raw search query will look like
type searchKeyword struct {
	workItemTypes []uuid.UUID
	number        []string
	words         []string
}

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
func isKnownURL(url string) (bool, string) {
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
		return false, ""
	}
	return true, mostReleventMatchName
}

func trimProtocolFromURLString(urlString string) string {
	urlString = strings.TrimPrefix(urlString, `http://`)
	urlString = strings.TrimPrefix(urlString, `https://`)
	return urlString
}

func escapeCharFromURLString(urlString string) string {
	// Replacer will escape `:` and `)` `(`, and `'`.
	var replacer = strings.NewReplacer(":", "\\:", "(", "\\(", ")", "\\)", "'", "")
	return replacer.Replace(urlString)
}

// sanitizeURL does cleaning of URL
// returns DB friendly string
// Trims protocol and escapes ":"
func sanitizeURL(urlString string) string {
	trimmedURL := trimProtocolFromURLString(urlString)
	return escapeCharFromURLString(trimmedURL)
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
		searchQueryString := strings.Join(match[1:], "")
		searchQueryString = strings.Replace(searchQueryString, ":", "\\:", -1)
		// need to escape ":" because this string will go as an input to tsquery
		searchQueryString = fmt.Sprintf("%s:*", searchQueryString)
		if result["id"] != "" {
			// Look for pattern's ID field, if exists update searchQueryString
			// `*A` is used to add sme weight to the work item number in the search results.
			// See https://www.postgresql.org/docs/9.6/static/textsearch-controls.html
			searchQueryString = fmt.Sprintf("(%v:*A | %v)", result["id"], searchQueryString)
			// searchQueryString = "(" + result["id"] + ":*" + " | " + searchQueryString + ")"
		}
		return searchQueryString
	}
	return match[0] + ":*"
}

/*
getSearchQueryFromURLString gets a url string and checks if that matches with any of known urls.
Respectively it will return a string that can be directly used in search query
e.g>
Unknown url : www.google.com then response = "www.google.com:*"
Known url : almighty.io/detail/500 then response = "500:* | almighty.io/detail/500"
*/
func getSearchQueryFromURLString(url string) string {
	known, patternName := isKnownURL(url)
	if known {
		// this url is known to system
		return getSearchQueryFromURLPattern(patternName, url)
	}
	// any URL other than our system's
	// return url without protocol
	return sanitizeURL(url) + ":*"
}

// parseSearchString accepts a raw string and generates a searchKeyword object
func parseSearchString(ctx context.Context, rawSearchString string) (searchKeyword, error) {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.Trim(rawSearchString, "/") // get rid of trailing slashes
	rawSearchString = strings.Trim(rawSearchString, "\"")
	parts := strings.Fields(rawSearchString)
	var res searchKeyword
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
			res.number = append(res.number, strings.TrimPrefix(part, "number:")+":*A")
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
			res.workItemTypes = append(res.workItemTypes, typeID)
		} else if govalidator.IsURL(part) {
			log.Debug(ctx, map[string]interface{}{"url": part}, "found a URL in the query string")
			part := strings.ToLower(part)
			part = trimProtocolFromURLString(part)
			searchQueryFromURL := getSearchQueryFromURLString(part)
			log.Debug(ctx, map[string]interface{}{"url": part, "search_query": searchQueryFromURL}, "found a URL in the query string")
			res.words = append(res.words, searchQueryFromURL)
		} else {
			part := strings.ToLower(part)
			part = sanitizeURL(part)
			res.words = append(res.words, part+":*")
		}
	}
	log.Info(nil, nil, "Search keywords: '%s' -> %v", rawSearchString, res)
	return res, nil
}

func parseMap(queryMap map[string]interface{}, q *Query) {
	childSet := false
	for key, val := range queryMap {
		switch concreteVal := val.(type) {
		case []interface{}:
			q.Name = key
			parseArray(val.([]interface{}), &q.Children)
		case string:
			q.Name = key
			s := string(concreteVal)
			q.Value = &s
			if q.Name == "iteration" || q.Name == "area" {
				if !childSet {
					q.Child = true
				}
			}
		case bool:
			s := concreteVal
			if key == "negate" {
				q.Negate = s
			} else if key == "child" {
				q.Child = s
				childSet = true
			}

		case nil:
			q.Name = key
			q.Value = nil
		case map[string]interface{}:
			if key == OPTS {
				continue
			}
			q.Name = key
			if v, ok := concreteVal[IN]; ok {
				q.Name = OR
				c := &q.Children
				for _, vl := range v.([]interface{}) {
					sq := Query{}
					sq.Name = key
					t := vl.(string)
					sq.Value = &t
					*c = append(*c, sq)
				}
			} else if v, ok := concreteVal[EQ]; ok {
				switch v.(type) {
				case string:
					s := v.(string)
					q.Value = &s
				case nil:
					q.Value = nil
				}
			} else if v, ok := concreteVal[NE]; ok {
				s := v.(string)
				q.Value = &s
				q.Negate = true
			} else if v, ok := concreteVal[SUBSTR]; ok {
				s := v.(string)
				q.Value = &s
				q.Substring = true
			}
		default:
			log.Error(nil, nil, "Unexpected value: %#v", val)
		}
	}
}

func parseOptions(queryMap map[string]interface{}) *QueryOptions {
	for key, val := range queryMap {
		if ifArr, ok := val.(map[string]interface{}); key == OPTS && ok {
			options := QueryOptions{}
			for k, v := range ifArr {
				switch k {
				case OptParentExistsKey:
					options.ParentExists = v.(bool)
				case OptTreeViewKey:
					options.TreeView = v.(bool)
				}
			}
			return &options
		}
	}
	return nil
}

func parseArray(anArray []interface{}, l *[]Query) {
	for _, val := range anArray {
		if o, ok := val.(map[string]interface{}); ok {
			q := Query{}
			parseMap(o, &q)
			*l = append(*l, q)
		}
	}
}

// QueryOptions represents all options provided user
type QueryOptions struct {
	TreeView     bool
	ParentExists bool
}

// Query represents tree structure of the filter query
type Query struct {
	// Name can contain a field name to search for (e.g. "space") or one of the
	// binary operators "$AND", or "$OR". If the Name is not an operator, we
	// compare the Value against a column in the database that maps to this
	// Name. We check the Value for equality and for inequality if the Negate
	// field is set to true.
	Name string
	// Operator nodes, with names "$AND", "$OR" etc. will not have values.
	// Since this struct represents tail nodes as well as these operator nodes,
	// the pointer is more suitable.
	Value *string
	// When Negate is true the comparison desribed above is negated; hence we
	// check for inequality. When Name is an operator, the Negate field has no
	// effect.
	Negate bool
	// If Substring is true, instead of exact match, anything that matches partially
	// will be considered.
	Substring bool
	// A Query is expected to have child queries only if the Name field contains
	// an operator like "$AND", or "$OR". If the Name is not an operator, the
	// Children slice MUST be empty.
	Children []Query
	// The Options represent the query options provided by the user.
	Options *QueryOptions
	// Consider child iteration/area
	Child bool
}

func isOperator(str string) bool {
	return str == AND || str == OR
}

var searchKeyMap = map[string]string{
	"area":         workitem.SystemArea,
	"iteration":    workitem.SystemIteration,
	"assignee":     workitem.SystemAssignees,
	"title":        workitem.SystemTitle,
	"creator":      workitem.SystemCreator,
	"label":        workitem.SystemLabels,
	"state":        workitem.SystemState,
	"boardcolumn":  workitem.SystemBoardcolumns,
	"type":         "Type",
	"workitemtype": "Type", // same as 'type' - added for compatibility. (Ref. #1564)
	"space":        "SpaceID",
	"number":       "Number",
}

func (q Query) determineLiteralType(key string, val string) criteria.Expression {
	switch key {
	case workitem.SystemAssignees, workitem.SystemLabels, workitem.SystemBoardcolumns, workitem.SystemBoard:
		return criteria.Literal([]string{val})
	default:
		return criteria.Literal(val)
	}
}

func (q Query) generateExpression() (criteria.Expression, error) {
	var myexpr []criteria.Expression
	currentOperator := q.Name

	if !isOperator(currentOperator) || currentOperator == OPTS {
		key, ok := searchKeyMap[q.Name]
		// check that none of the default table joins handles this column:
		var handledByJoin bool
		joins := workitem.DefaultTableJoins()
		for _, j := range joins {
			if j.HandlesFieldName(q.Name) {
				handledByJoin = true
				key = q.Name
				break
			}
		}
		if !ok && !handledByJoin {
			return nil, errors.NewBadParameterError("key not found", q.Name)
		}
		left := criteria.Field(key)
		if q.Value != nil {
			right := q.determineLiteralType(key, *q.Value)
			if q.Negate {
				myexpr = append(myexpr, criteria.Not(left, right))
			} else {
				if q.Substring {
					myexpr = append(myexpr, criteria.Substring(left, right))
				} else {
					if q.Child {
						myexpr = append(myexpr, criteria.Child(left, right))
					} else {
						myexpr = append(myexpr, criteria.Equals(left, right))
					}
				}
			}
		} else {
			if q.Negate {
				return nil, errors.NewBadParameterError("negate for null not supported", q.Name)
			}
			myexpr = append(myexpr, criteria.IsNull(key))
		}
	}
	for _, child := range q.Children {
		if isOperator(child.Name) || currentOperator == OPTS {
			exp, err := child.generateExpression()
			if err != nil {
				return nil, err
			}
			myexpr = append(myexpr, exp)
		} else {
			key, ok := searchKeyMap[child.Name]
			// check that none of the default table joins handles this column:
			var handledByJoin bool
			joins := workitem.DefaultTableJoins()
			for _, j := range joins {
				if j.HandlesFieldName(child.Name) {
					handledByJoin = true
					key = child.Name
					break
				}
			}
			if !ok && !handledByJoin {
				return nil, errors.NewBadParameterError("key not found", child.Name)
			}
			left := criteria.Field(key)
			if child.Value != nil {
				right := q.determineLiteralType(key, *child.Value)
				if child.Negate {
					myexpr = append(myexpr, criteria.Not(left, right))
				} else {
					if child.Substring {
						myexpr = append(myexpr, criteria.Substring(left, right))
					} else {
						if child.Child {
							myexpr = append(myexpr, criteria.Child(left, right))
						} else {
							myexpr = append(myexpr, criteria.Equals(left, right))
						}
					}
				}
			} else {
				if child.Negate {
					return nil, errors.NewBadParameterError("negate for null not supported", child.Name)
				}
				myexpr = append(myexpr, criteria.IsNull(key))

			}
		}
	}
	var res criteria.Expression
	switch currentOperator {
	case AND:
		for _, expr := range myexpr {
			if res == nil {
				res = expr
			} else {
				res = criteria.And(res, expr)
			}
		}
	case OR:
		for _, expr := range myexpr {
			if res == nil {
				res = expr
			} else {
				res = criteria.Or(res, expr)
			}
		}
	default:
		for _, expr := range myexpr {
			if res == nil {
				res = expr
			}
		}
	}
	return res, nil
}

// ParseFilterString accepts a raw string and generates a criteria expression
func ParseFilterString(ctx context.Context, rawSearchString string) (criteria.Expression, *QueryOptions, error) {
	fm := map[string]interface{}{}
	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(rawSearchString), &fm)

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":             err,
			"rawSearchString": rawSearchString,
		}, "failed to unmarshal raw search string")
		return nil, nil, errors.NewBadParameterError("expression", rawSearchString+": "+err.Error())
	}
	q := Query{}
	parseMap(fm, &q)

	q.Options = parseOptions(fm)

	exp, err := q.generateExpression()
	return exp, q.Options, err
}

// generateSQLSearchInfo accepts searchKeyword and join them in a way that can be used in sql
func generateSQLSearchInfo(keywords searchKeyword) (sqlParameter string) {
	numberStr := strings.Join(keywords.number, " & ")
	wordStr := strings.Join(keywords.words, " & ")
	var fragments []string
	for _, v := range []string{numberStr, wordStr} {
		if v != "" {
			fragments = append(fragments, v)
		}
	}
	searchStr := strings.Join(fragments, " & ")
	return searchStr
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormSearchRepository) search(ctx context.Context, sqlSearchQueryParameter string, workItemTypes []uuid.UUID, start *int, limit *int, spaceID *string) ([]workitem.WorkItemStorage, int, error) {
	db := r.db.Model(workitem.WorkItemStorage{}).Where("tsv @@ query")
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}
	if len(workItemTypes) > 0 {
		// restrict to all given types and their subtypes
		query := fmt.Sprintf("%[1]s.type in ("+
			"select distinct subtype.id from %[2]s subtype "+
			"join %[2]s supertype on subtype.path <@ supertype.path "+
			"where supertype.id in (?))", workitem.WorkItemStorage{}.TableName(), workitem.WorkItemType{}.TableName())
		db = db.Where(query, workItemTypes)
	}

	db = db.Select("count(*) over () as cnt2 , *").Order(workitem.Column(workitem.WorkItemStorage{}.TableName(), "execution_order") + " desc")
	db = db.Joins(", to_tsquery('english', ?) as query, ts_rank(tsv, query) as rank", sqlSearchQueryParameter)
	if spaceID != nil {
		db = db.Where("space_id=?", *spaceID)
	}
	db = db.Order(fmt.Sprintf("rank desc,%s.updated_at desc", workitem.WorkItemStorage{}.TableName()))

	rows, err := db.Rows()
	defer closeable.Close(ctx, rows)
	if err != nil {
		return nil, 0, errs.Wrapf(err, "failed to execute search query")
	}

	result := []workitem.WorkItemStorage{}
	value := workitem.WorkItemStorage{}
	columns, err := rows.Columns()
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to get column names")
		return nil, 0, errors.NewInternalError(ctx, errs.Wrap(err, "failed to get column names"))
	}

	// need to set up a result for Scan() in order to extract total count.
	var count int
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				log.Error(ctx, map[string]interface{}{
					"err": err,
				}, "failed to scan rows")
				return nil, 0, errors.NewInternalError(ctx, errs.Wrap(err, "failed to scan rows"))
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query,
		count = 0
	}
	log.Info(ctx, nil, "Search results: %d matches", count)
	return result, count, nil
	//*/
}

// SearchFullText Search returns work items for the given query
func (r *GormSearchRepository) SearchFullText(ctx context.Context, rawSearchString string, start *int, limit *int, spaceID *string) ([]workitem.WorkItem, int, error) {
	// parse
	// generateSearchQuery
	// ....
	parsedSearchDict, err := parseSearchString(ctx, rawSearchString)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}

	sqlSearchQueryParameter := generateSQLSearchInfo(parsedSearchDict)
	var rows []workitem.WorkItemStorage
	log.Debug(ctx, map[string]interface{}{"search query": sqlSearchQueryParameter}, "searching for work items")
	rows, count, err := r.search(ctx, sqlSearchQueryParameter, parsedSearchDict.workItemTypes, start, limit, spaceID)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	result := make([]workitem.WorkItem, len(rows))

	for index, value := range rows {
		var err error
		// FIXME: Against best practice http://go-database-sql.org/retrieving.html
		wiType, err := r.witr.Load(ctx, value.Type)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
				"wit": value.Type,
			}, "failed to load work item type")
			spew.Dump(value)
			return nil, 0, errors.NewInternalError(ctx, errs.Wrap(err, "failed to load work item type"))
		}
		wiModel, err := workitem.ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			return nil, 0, errors.NewConversionError(err.Error())
		}
		result[index] = *wiModel
	}

	return result, count, nil
}

func (r *GormSearchRepository) listItemsFromDB(ctx context.Context, criteria criteria.Expression, parentExists *bool, start *int, limit *int) ([]workitem.WorkItemStorage, int, error) {
	where, parameters, joins, compileError := workitem.Compile(criteria)
	if compileError != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        compileError,
			"expression": criteria,
		}, "failed to compile expression")
		return nil, 0, errors.NewBadParameterError("expression", criteria)
	}

	if parentExists != nil && !*parentExists {
		where += fmt.Sprintf(` AND
			NOT EXISTS (
				SELECT wil.target_id FROM work_item_links wil
				WHERE wil.link_type_id = '%[1]s'
				AND wil.target_id = work_items.id
				AND wil.deleted_at IS NULL)`, link.SystemWorkItemLinkTypeParentChildID)
	}

	db := r.db.Model(&workitem.WorkItemStorage{}).Where(where, parameters...)
	for _, j := range joins {
		if err := j.Validate(db); err != nil {
			log.Error(ctx, map[string]interface{}{"expression": criteria, "err": err}, "table join not valid")
			return nil, 0, errors.NewBadParameterError("expression", criteria).Expected("valid table join")
		}
		db = db.Joins(j.GetJoinExpression())
	}
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}

	db = db.Select("count(*) over () as cnt2 , *").Order(workitem.Column(workitem.WorkItemStorage{}.TableName(), "execution_order") + " desc")

	rows, err := db.Rows()
	defer closeable.Close(ctx, rows)
	if err != nil {
		if gormsupport.IsDataException(err) {
			// Remove "pq: " from the original message and return it.
			errMessage := strings.Replace(err.Error(), "pq: ", "", -1)
			return nil, 0, errors.NewBadParameterErrorFromString(errMessage)
		}
		return nil, 0, errs.WithStack(err)
	}
	result := []workitem.WorkItemStorage{}
	columns, err := rows.Columns()
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to list column names")
		return nil, 0, errors.NewInternalError(ctx, errs.Wrap(err, "failed to list column names"))
	}

	// need to set up a result for Scan() in order to extract total count.
	var count int
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := workitem.WorkItemStorage{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				log.Error(ctx, map[string]interface{}{
					"err": err,
				}, "failed to scan rows")
				return nil, 0, errors.NewInternalError(ctx, errs.Wrap(err, "failed to scan rows"))
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
		// need to do a count(*) to find out total
		orgDB := orgDB.Select("count(*)")
		rows2, err := orgDB.Rows()
		defer closeable.Close(ctx, rows2)
		if err != nil {
			return nil, 0, errs.WithStack(err)
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}
	return result, count, nil
}

// Filter returns the work items matching the search as well as their count. If
// the filter did specify the "tree-view" option to be "true", then we will also
// create a list of ancestors as well as a list of links. The ancestors exist in
// order to list the parent of each matching work item up to its root work item.
// The child links are there in order to know what siblings to load for matching
// work items.
func (r *GormSearchRepository) Filter(ctx context.Context, rawFilterString string, parentExists *bool, start *int, limit *int) (matches []workitem.WorkItem, count int, ancestors link.AncestorList, childLinks link.WorkItemLinkList, err error) {
	// parse
	// generateSearchQuery
	// ....
	exp, opts, err := ParseFilterString(ctx, rawFilterString)
	if err != nil {
		return nil, 0, nil, nil, errs.Wrap(err, "failed to parse filter string")
	}
	log.Debug(ctx, map[string]interface{}{
		"expression": exp,
		"raw_filter": rawFilterString,
	}, "Filtering work items...")

	if exp == nil {
		log.Error(ctx, map[string]interface{}{
			"expression": exp,
			"raw_filter": rawFilterString,
		}, "unable to parse the raw filter string")
		return nil, 0, nil, nil, errors.NewBadParameterError("rawFilterString", rawFilterString)
	}

	result, count, err := r.listItemsFromDB(ctx, exp, parentExists, start, limit)
	if err != nil {
		return nil, 0, nil, nil, errs.WithStack(err)
	}

	// if requested search for ancestors of all matched work items
	if opts != nil && opts.TreeView {
		linkRepo := link.NewWorkItemLinkRepository(r.db)
		matchingIDs := make([]uuid.UUID, len(result))
		for i, wi := range result {
			matchingIDs[i] = wi.ID
		}
		ancestors, err = linkRepo.GetAncestors(ctx, link.SystemWorkItemLinkTypeParentChildID, link.AncestorLevelAll, matchingIDs...)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"expression":  exp,
				"raw_filter":  rawFilterString,
				"err":         err,
				"matchingIDs": matchingIDs,
			}, "failed to find ancestors for these work items")
			return nil, 0, nil, nil, errs.Wrapf(err, "failed to find ancestors for these work items: %s", matchingIDs)
		}

		// For each matchingIDs work item that has a child which is also a matching
		// work item, we load all direct children.
		includeChildrenFor := id.Slice{}
		for _, match := range matchingIDs {
			var includeChildren bool
			// Check if this matched work item appears as a parent for one of
			// the other matches. If it does, then include its direct children.
			for i := 0; i < len(result) && !includeChildren; i++ {
				if result[i].ID == match {
					continue
				}
				parent := ancestors.GetParentOf(result[i].ID)
				if parent != nil && parent.ID == match {
					includeChildren = true
					includeChildrenFor = append(includeChildrenFor, match)
				}
			}
		}
		childLinks, err = linkRepo.ListChildLinks(ctx, link.SystemWorkItemLinkTypeParentChildID, includeChildrenFor...)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"expression": exp,
				"raw_filter": rawFilterString,
				"err":        err,
			}, "failed to list child links for work items %+v", includeChildrenFor)
			return nil, 0, nil, nil, errs.Wrapf(err, "failed to list child links for work item %+v", includeChildrenFor)
		}
	}

	matches = make([]workitem.WorkItem, len(result))
	for index, value := range result {
		wiType, err := r.witr.Load(ctx, value.Type)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
				"wit": value.Type,
			}, "failed to load work item type")
			return nil, 0, nil, nil, errors.NewInternalError(ctx, errs.Wrap(err, "failed to load work item type"))
		}
		modelWI, err := workitem.ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to convert to storage to model")
			return nil, 0, nil, nil, errors.NewInternalError(ctx, errs.Wrap(err, "failed to convert storage to model"))
		}
		matches[index] = *modelWI
	}
	return matches, count, ancestors, childLinks, nil
}
