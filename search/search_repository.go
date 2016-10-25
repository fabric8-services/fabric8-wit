package search

import (
	"golang.org/x/net/context"

	"log"
	"strconv"

	"strings"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

const (
	/*
		- The SQL queries do a case-insensitive search.
		- English words are normalized during search which means words like qualifying === qualify
		- To disable the above normalization change "to_tsquery('english',$1)" to "to_tsquery($1)"
		- Create GIN indexes : https://www.postgresql.org/docs/9.5/static/textsearch-tables.html#TEXTSEARCH-TABLES-INDEX

	*/

	// This SQL query is used when search is performed across workitem fields and workitem ID
	testText = `select * from work_items WHERE to_tsvector('english', id::text || ' ' || fields::text) @@ to_tsquery('english',$1) and deleted_at is NULL`

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

// parseSearchString accepts a raw string and generates a searchKeyword object
func parseSearchString(rawSearchString string) searchKeyword {
	// TODO remove special characters and exclaimations if any
	rawSearchString = strings.ToLower(rawSearchString)
	parts := strings.Split(rawSearchString, " ")
	var res searchKeyword
	for _, part := range parts {
		if strings.HasPrefix(part, "id:") {
			res.id = append(res.id, strings.Trim(part, "id:"))
		} else {
			res.words = append(res.words, part)
		}
	}
	return res
}

func generateSQLSearchString(keywords searchKeyword) string {
	idStr := strings.Join(keywords.id, " & ")
	wordStr := strings.Join(keywords.words, " & ")
	return idStr + wordStr
}

// SearchFullText Search returns work items for the given query
func (r *GormSearchRepository) SearchFullText(ctx context.Context, rawSearchString string) ([]*app.WorkItem, error) {
	// parse
	// generateSearchQuery
	// ....
	parsedSearchDict := parseSearchString(rawSearchString)
	searchQuery := testText
	if len(parsedSearchDict.id) > 0 {
		searchQuery = testID
	}
	sqlSearchStringParameter := generateSQLSearchString(parsedSearchDict)
	var rows []models.WorkItem
	db := r.ts.TX().Raw(searchQuery, sqlSearchStringParameter)
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
