package search

import (
	"fmt"
	"sync"

	"golang.org/x/net/context"

	"strconv"

	"strings"

	"regexp"

	"net/url"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/asaskevich/govalidator"
	"github.com/jinzhu/gorm"
)

const (
/*
	- The SQL queries do a case-insensitive search.
	- English words are normalized during search which means words like qualifying === qualify
	- To disable the above normalization change "to_tsquery('english',$1)" to "to_tsquery($1)"
	- Create GIN indexes : https://www.postgresql.org/docs/9.5/static/textsearch-tables.html#TEXTSEARCH-TABLES-INDEX
	- To perform "LIKE" query we are appending ":*" to the search token

*/
)

// GormSearchRepository provides a Gorm based repository
type GormSearchRepository struct {
	db  *gorm.DB
	wir *models.GormWorkItemTypeRepository
}

// NewGormSearchRepository creates a new search repository
func NewGormSearchRepository(db *gorm.DB) *GormSearchRepository {
	return &GormSearchRepository{db, models.NewWorkItemTypeRepository(db)}
}

func generateSearchQuery(q string) (string, error) {
	return q, nil
}

func convertFromModel(wiType models.WorkItemType, workItem models.WorkItem) (*app.WorkItem, error) {
	result := app.WorkItem{
		ID:      strconv.FormatUint(workItem.ID, 10),
		Type:    workItem.Type,
		Version: workItem.Version,
		Fields:  map[string]interface{}{}}

	for name, field := range wiType.Fields {
		var err error
		result.Fields[name], err = field.ConvertFromModel(name, workItem.Fields[name])
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

//searchKeyword defines how a decomposed raw search query will look like
type searchKeyword struct {
	workItemTypes []string
	id            []string
	words         []string
}

// KnownURL has a regex string format URL and compiled regex for the same
type KnownURL struct {
	urlRegex          string         // regex for URL
	compiledRegex     *regexp.Regexp // valid output of regexp.MustCompile()
	groupNamesInRegex []string       // Valid output of SubexpNames called on compliedRegex
}

/*
KnownURLs is set of KnownURLs will be used while searching on a URL
"Known" means that, our system understands the format of URLs
URLs in this slice will be considered while searching to match search string and decouple it into multiple searchable parts
e.g> Following example defines work-item-detail-page URL on client side, with its compiled version
knownURLs["work-item-details"] = KnownURL{
urlRegex:      `^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/work-item-list/detail/)(?P<id>\d*)`,
compiledRegex: regexp.MustCompile(`^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/work-item-list/detail/)(?P<id>\d*)`),
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
		urlRegex:          urlRegex,
		compiledRegex:     regexp.MustCompile(urlRegex),
		groupNamesInRegex: groupNames,
	}
}

/*
isKnownURL compares with registered URLs in our system.
Iterates over knownURLs and finds out most relevent matching pattern.
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
	return strings.Replace(urlString, ":", "\\:", -1)
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
	// Join rest of the tokens to make query like "demo.almighty.io/work-item-list/detail/100"
	if len(match) > 1 {
		searchQueryString := strings.Join(match[1:], "")
		searchQueryString = strings.Replace(searchQueryString, ":", "\\:", -1)
		// need to escape ":" because this string will go as an input to tsquery
		searchQueryString = fmt.Sprintf("%s:*", searchQueryString)
		if result["id"] != "" {
			// Look for pattern's ID field, if exists update searchQueryString
			searchQueryString = fmt.Sprintf("(%v:* | %v)", result["id"], searchQueryString)
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
func parseSearchString(rawSearchString string) (searchKeyword, error) {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.Trim(rawSearchString, "/") // get rid of trailing slashes
	rawSearchString = strings.Trim(rawSearchString, "\"")
	parts := strings.Fields(rawSearchString)
	var res searchKeyword
	for _, part := range parts {
		// QueryUnescape is required in case of encoded url strings.
		// And does not harm regular search strings
		// but this processing is required becasue at this moment, we do not know if
		// search input is a regular string or a URL

		part, err := url.QueryUnescape(part)
		if err != nil {
			fmt.Println("Could not escape url", err)
		}
		// IF part is for search with id:1234
		// TODO: need to find out the way to use ID fields.
		if strings.HasPrefix(part, "id:") {
			res.id = append(res.id, strings.TrimPrefix(part, "id:")+":*A")
		} else if strings.HasPrefix(part, "type:") {
			typeName := strings.TrimPrefix(part, "type:")
			if len(typeName) == 0 {
				return res, models.NewBadParameterError("Type name must not be empty", part)
			}
			res.workItemTypes = append(res.workItemTypes, typeName)
		} else if govalidator.IsURL(part) {
			part := strings.ToLower(part)
			part = trimProtocolFromURLString(part)
			searchQueryFromURL := getSearchQueryFromURLString(part)
			res.words = append(res.words, searchQueryFromURL)
		} else {
			part := strings.ToLower(part)
			part = sanitizeURL(part)
			res.words = append(res.words, part+":*")
		}
	}
	return res, nil
}

// generateSQLSearchInfo accepts searchKeyword and join them in a way that can be used in sql
func generateSQLSearchInfo(keywords searchKeyword) (sqlParameter string) {
	idStr := strings.Join(keywords.id, " & ")
	wordStr := strings.Join(keywords.words, " & ")

	searchStr := idStr + wordStr
	if len(wordStr) != 0 && len(idStr) != 0 {
		searchStr = idStr + " & " + wordStr
	}
	return searchStr
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormSearchRepository) search(ctx context.Context, sqlSearchQueryParameter string, workItemTypes []string, start *int, limit *int) ([]models.WorkItem, uint64, error) {
	db := r.db.Model(models.WorkItem{}).Where("tsv @@ query")
	if start != nil {
		if *start < 0 {
			return nil, 0, models.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, models.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}
	if len(workItemTypes) > 0 {
		// restrict to all given types and their subtypes
		query := fmt.Sprintf("%[1]s.type in ("+
			"select distinct subtype.name from %[2]s subtype "+
			"join %[2]s supertype on subtype.path like (supertype.path || '%%') "+
			"where supertype.name in (?))", models.WorkItem{}.TableName(), models.WorkItemType{}.TableName())
		db = db.Where(query, workItemTypes)
	}

	db = db.Select("count(*) over () as cnt2 , *")
	db = db.Joins(", to_tsquery('english', ?) as query, ts_rank(tsv, query) as rank", sqlSearchQueryParameter)
	db = db.Order(fmt.Sprintf("rank desc,%s.updated_at desc", models.WorkItem{}.TableName()))

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []models.WorkItem{}
	value := models.WorkItem{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, models.NewInternalError(err.Error())
	}

	// need to set up a result for Scan() in order to extract total count.
	var count uint64
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
				return nil, 0, models.NewInternalError(err.Error())
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query,
		count = 0
	}
	return result, count, nil
	//*/
}

// SearchFullText Search returns work items for the given query
func (r *GormSearchRepository) SearchFullText(ctx context.Context, rawSearchString string, start *int, limit *int) ([]*app.WorkItem, uint64, error) {
	// parse
	// generateSearchQuery
	// ....
	parsedSearchDict, err := parseSearchString(rawSearchString)
	if err != nil {
		return nil, 0, err
	}

	sqlSearchQueryParameter := generateSQLSearchInfo(parsedSearchDict)
	var rows []models.WorkItem
	rows, count, err := r.search(ctx, sqlSearchQueryParameter, parsedSearchDict.workItemTypes, start, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*app.WorkItem, len(rows))

	for index, value := range rows {
		var err error
		// FIXME: Against best practice http://go-database-sql.org/retrieving.html
		wiType, err := r.wir.LoadTypeFromDB(value.Type)
		if err != nil {
			return nil, 0, models.NewInternalError(err.Error())
		}
		result[index], err = convertFromModel(*wiType, value)
		if err != nil {
			return nil, 0, models.NewConversionError(err.Error())
		}
	}

	return result, count, nil
}

func init() {
	// While registering URLs do not include protocol becasue it will be removed before scanning starts
	// Please do not include trailing slashes becasue it will be removed before scanning starts
	RegisterAsKnownURL("work-item-details", `(?P<domain>demo.almighty.io)(?P<path>/work-item-list/detail/)(?P<id>\d*)`)
	RegisterAsKnownURL("localhost-work-item-details", `(?P<domain>localhost)(?P<port>:\d+){0,1}(?P<path>/work-item-list/detail/)(?P<id>\d*)`)
}
