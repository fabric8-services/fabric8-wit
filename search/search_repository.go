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

	// WhereClauseForSearchByText This SQL query is used when search is performed across workitem fields and workitem ID
	WhereClauseForSearchByText = `setweight(to_tsvector('english',coalesce(fields->>'system.title','')),'B')||
				setweight(to_tsvector('english',coalesce(fields->>'system.description','')),'C')|| 
				setweight(to_tsvector('english', id::text),'A')
				@@ to_tsquery('english',$1)`

	// WhereClauseForSearchByID This SQL query is used when search is performed across workitem ID ONLY.
	WhereClauseForSearchByID = `to_tsvector('english', id::text || ' ') @@ to_tsquery('english',$1)`
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
	id    []string
	words []string
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
urlRegex:      `^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/detail/)(?P<id>\d*)`,
compiledRegex: regexp.MustCompile(`^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/detail/)(?P<id>\d*)`),
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
	// Join rest of the tokens to make query like "demo.almighty.io/details/100"
	if len(match) > 1 {
		searchQueryString := strings.Join(match[1:], "") + ":*"
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
	return strings.Trim(url, `http[s]://`) + ":*"
}

// parseSearchString accepts a raw string and generates a searchKeyword object
func parseSearchString(rawSearchString string) searchKeyword {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.ToLower(rawSearchString)
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
			res.id = append(res.id, strings.Trim(part, "id:"))
		} else if govalidator.IsURL(part) {
			part := strings.Trim(part, `http[s]://`)
			searchQueryFromURL := getSearchQueryFromURLString(part)
			res.words = append(res.words, searchQueryFromURL)
		} else {
			res.words = append(res.words, part)
		}
	}
	return res
}

// generateSQLSearchInfo accepts searchKeyword and join them in a way that can be used in sql
func generateSQLSearchInfo(keywords searchKeyword) (sqlQuery string, sqlParameter string) {
	idStr := strings.Join(keywords.id, " & ")
	wordStr := strings.Join(keywords.words, " & ")
	searchQuery := WhereClauseForSearchByText

	if len(keywords.id) == 1 && len(keywords.words) == 0 {
		// If the search string is of the form "id:2647326482" then we perform
		// search only on the ID, else we do a full text search.
		// Is "id:45453 id:43234" be valid ? NO, because the no row can have 2 IDs.
		searchQuery = WhereClauseForSearchByID
	}

	searchStr := idStr + wordStr
	if len(wordStr) != 0 && len(idStr) != 0 {
		searchStr = idStr + " & " + wordStr
	}
	return searchQuery, searchStr
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormSearchRepository) search(ctx context.Context, sqlSearchQuery string, sqlSearchQueryParameter string, start *int, limit *int) ([]models.WorkItem, uint64, error) {
	db := r.db.Model(&models.WorkItem{}).Where(sqlSearchQuery, sqlSearchQueryParameter)
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, BadParameterError{"start", *start}
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, BadParameterError{"limit", *limit}
		}
		db = db.Limit(*limit)
	}
	db = db.Select("count(*) over () as cnt2 , *")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	result := []models.WorkItem{}
	value := models.WorkItem{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, InternalError{simpleError{err.Error()}}
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
				return nil, 0, InternalError{simpleError{err.Error()}}
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
		// need to do a count(*) to find out total
		orgDB := orgDB.Select("count(*)")
		rows2, err := orgDB.Rows()
		defer rows2.Close()
		if err != nil {
			return nil, 0, err
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}
	return result, count, nil
	//*/
}

// SearchFullText Search returns work items for the given query
func (r *GormSearchRepository) SearchFullText(ctx context.Context, rawSearchString string, start *int, limit *int) ([]*app.WorkItem, uint64, error) {
	// parse
	// generateSearchQuery
	// ....
	parsedSearchDict := parseSearchString(rawSearchString)

	sqlSearchQuery, sqlSearchQueryParameter := generateSQLSearchInfo(parsedSearchDict)
	var rows []models.WorkItem
	rows, count, err := r.search(ctx, sqlSearchQuery, sqlSearchQueryParameter, start, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*app.WorkItem, len(rows))

	for index, value := range rows {
		var err error
		// FIXME: Against best practice http://go-database-sql.org/retrieving.html
		wiType, err := r.wir.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, InternalError{simpleError{err.Error()}}
		}
		result[index], err = convertFromModel(*wiType, value)
		if err != nil {
			return nil, 0, ConversionError{simpleError{err.Error()}}
		}
	}

	return result, count, nil
}

// Validate ensures that the search string is valid and also ensures its not an injection attack.
func (r *GormSearchRepository) Validate(ctx context.Context, rawSearchString string) error {
	return nil
}

func init() {
	// While registering URLs do not include protocol becasue it will be removed before scanning starts
	RegisterAsKnownURL("work-item-details", `(?P<domain>demo.almighty.io)(?P<path>/detail/)(?P<id>\d*)`)
}
