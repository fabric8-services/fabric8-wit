package search

import (
	"context"
	"log"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

const (
	fulltextPsqlQuery = `select * from work_items WHERE to_tsvector('english', id::text || ' ' || fields::text) @@ to_tsquery($1)`
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

// Search returns work items for the given query
func (r *GormSearchRepository) Search(ctx context.Context, q string) ([]*app.WorkItem, error) {
	searchQuery, err := generateSearchQuery(q)

	if err != nil {
		return nil, BadParameterError{"expression", q}
	}

	var rows []models.WorkItem
	db := r.ts.TX().Raw(fulltextPsqlQuery, searchQuery)
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
