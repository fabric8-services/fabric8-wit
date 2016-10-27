package search

import (
	"golang.org/x/net/context"

	"log"
	"strconv"

	"strings"

	"regexp"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/asaskevich/govalidator"
)

const (
	/*
		- The SQL queries do a case-insensitive search.
		- English words are normalized during search which means words like qualifying === qualify
		- To disable the above normalization change "to_tsquery('english',$1)" to "to_tsquery($1)"
		- Create GIN indexes : https://www.postgresql.org/docs/9.5/static/textsearch-tables.html#TEXTSEARCH-TABLES-INDEX
		- To perform "LIKE" query we are appending ":*" to the search token

	*/

	// This SQL query is used when search is performed across workitem fields and workitem ID
	testText = `select * from work_items WHERE
		setweight(to_tsvector('english', id::text), 'A') ||
		setweight(to_tsvector('english', fields::text), 'B') @@ to_tsquery('english',$1) and deleted_at is NULL`

	// This SQL query is used when search is performed across workitem ID ONLY.
	testID = `select * from work_items WHERE to_tsvector('english', id::text || ' ') @@ to_tsquery('english',$1) and deleted_at is NULL`
)

// GormSearchRepository provides a Gorm based repository
type GormSearchRepository struct {
	ts  *models.GormTransactionSupport
	wir *models.GormWorkItemTypeRepository
}

// NewGormSearchRepository creates a new search repository
func NewGormSearchRepository(ts *models.GormTransactionSupport, wir *models.GormWorkItemTypeRepository) *GormSearchRepository {
	return &GormSearchRepository{ts, wir}
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

func (r *GormSearchRepository) loadTypeFromDB(ctx context.Context, name string) (*models.WorkItemType, error) {
	log.Printf("loading work item type %s", name)
	res := models.WorkItemType{}

	if r.ts.TX().Where("name=?", name).First(&res).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item type", name}
	}
	if err := r.ts.TX().Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	return &res, nil
}

//searchKeyword defines how a decomposed raw search query will look like
type searchKeyword struct {
	id    []string
	words []string
}

// RegexWorkItemDetailClientURL tells us how URL for WI details on front end looks like
const RegexWorkItemDetailClientURL = `^(?P<protocol>http[s]?)://(?P<domain>demo.almighty.io)(?P<path>/#/detail/)(?P<id>\d*)`

// Use following compiled form in future use
var compiledWIURL = regexp.MustCompile(RegexWorkItemDetailClientURL)

// First index is ignored because of behaviour of "SubexpNames"
var groupsWIURL = compiledWIURL.SubexpNames()[:1]

//mapURLGroupWithValues accepts slice of group names and slice of values.
//If both slices have different lenghts, empty value will be put for group name.
func mapURLGroupWithValues(groupNames []string, stringToMatch string) map[string]string {
	match := compiledWIURL.FindStringSubmatch(stringToMatch)
	result := make(map[string]string)
	for i, name := range groupNames {
		if i > len(match)-1 {
			result[name] = ""
		} else {
			result[name] = match[i]
		}
	}
	return result
}

// parseSearchString accepts a raw string and generates a searchKeyword object
func parseSearchString(rawSearchString string) searchKeyword {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.ToLower(rawSearchString)
	parts := strings.Split(rawSearchString, " ")
	var res searchKeyword
	for _, part := range parts {
		if strings.HasPrefix(part, "id:") {
			res.id = append(res.id, strings.Trim(part, "id:"))
		} else if govalidator.IsURL(part) {
			values := mapURLGroupWithValues(groupsWIURL, part)
			if values["id"] != "" {
				res.words = append(res.words, values["id"]+":*")
			}
			res.words = append(res.words, part+":*")
		} else {
			res.words = append(res.words, part)
		}
	}
	return res
}

func generateSQLSearchInfo(keywords searchKeyword) (sqlQuery string, sqlParameter string) {
	idStr := strings.Join(keywords.id, " & ")
	wordStr := strings.Join(keywords.words, " & ")
	searchQuery := testText

	if len(keywords.id) == 1 && len(keywords.words) == 0 {
		// If the search string is of the form "id:2647326482" then we perform
		// search only on the ID, else we do a full text search.
		// Is "id:45453 id:43234" be valid ? NO, because the no row can have 2 IDs.
		searchQuery = testID
	}
	return searchQuery, idStr + wordStr
}

// SearchFullText Search returns work items for the given query
func (r *GormSearchRepository) SearchFullText(ctx context.Context, rawSearchString string) ([]*app.WorkItem, error) {
	// parse
	// generateSearchQuery
	// ....
	parsedSearchDict := parseSearchString(rawSearchString)

	sqlSearchQuery, sqlSearchQueryParameter := generateSQLSearchInfo(parsedSearchDict)
	var rows []models.WorkItem
	db := r.ts.TX().Raw(sqlSearchQuery, sqlSearchQueryParameter)
	if err := db.Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*app.WorkItem, len(rows))

	for index, value := range rows {
		var err error
		// FIXME: Against best practice http://go-database-sql.org/retrieving.html
		wiType, err := r.loadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, InternalError{simpleError{err.Error()}}
		}
		result[index], err = convertFromModel(*wiType, value)
		if err != nil {
			return nil, ConversionError{simpleError{err.Error()}}
		}
	}

	return result, nil
}

// Validate ensures that the search string is valid and also ensures its not an injection attack.
func (r *GormSearchRepository) Validate(ctx context.Context, rawSearchString string) error {
	return nil
}
