package models

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/jinzhu/gorm"
)

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	db  *gorm.DB
	wir *GormWorkItemTypeRepository
}

// Load returns the work item for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) Load(ctx context.Context, ID string) (*app.WorkItem, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"work item", ID}
	}

	log.Printf("loading work item %d", id)
	res := WorkItem{}
	if r.db.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item", ID}
	}
	wiType, err := r.wir.loadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	result, err := convertFromModel(*wiType, res)
	if err != nil {
		return nil, ConversionError{simpleError{err.Error()}}
	}
	return result, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemRepository) Delete(ctx context.Context, ID string) error {
	var workItem = WorkItem{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treat as not found: clients don't know it must be a number
		return NotFoundError{entity: "work item", ID: ID}
	}
	workItem.ID = id
	tx := r.db

	if err = tx.Delete(workItem).Error; err != nil {
		if tx.RecordNotFound() {
			return NotFoundError{entity: "work item", ID: ID}
		}
		return InternalError{simpleError{err.Error()}}
	}

	return nil
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemRepository) Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error) {
	res := WorkItem{}
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		return nil, NotFoundError{entity: "work item", ID: wi.ID}
	}

	log.Printf("looking for id %d", id)
	tx := r.db
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "work item", ID: wi.ID}
	}
	if res.Version != wi.Version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	wiType, err := r.wir.loadTypeFromDB(ctx, wi.Type)
	if err != nil {
		return nil, BadParameterError{"Type", wi.Type}
	}

	newWi := WorkItem{
		ID:      id,
		Type:    wi.Type,
		Version: wi.Version + 1,
		Fields:  Fields{},
	}

	for fieldName, fieldDef := range wiType.Fields {
		fieldValue := wi.Fields[fieldName]
		var err error
		newWi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, BadParameterError{fieldName, fieldValue}
		}
	}

	if err := tx.Save(&newWi).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated item to %v\n", newWi)
	result, err := convertFromModel(*wiType, newWi)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	return result, nil
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemRepository) Create(ctx context.Context, typeID string, fields map[string]interface{}) (*app.WorkItem, error) {
	wiType, err := r.wir.loadTypeFromDB(ctx, typeID)
	if err != nil {
		return nil, BadParameterError{parameter: "type", value: typeID}
	}
	wi := WorkItem{
		Type:   typeID,
		Fields: Fields{},
	}
	for fieldName, fieldDef := range wiType.Fields {
		fieldValue := fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, BadParameterError{fieldName, fieldValue}
		}
	}
	tx := r.db

	if err = tx.Create(&wi).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created item %v\n", wi)
	result, err := convertFromModel(*wiType, wi)
	if err != nil {
		return nil, ConversionError{simpleError{err.Error()}}
	}

	return result, nil
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormWorkItemRepository) listItemsFromDB(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]WorkItem, uint64, error) {
	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return nil, 0, BadParameterError{"expression", criteria}
	}

	log.Printf("executing query: '%s' with params %v", where, parameters)
 
	db := r.db.Model(&WorkItem{}).Where(where, parameters...)
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

	result := []WorkItem{}
	value := WorkItem{}
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
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormWorkItemRepository) List(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]*app.WorkItem, uint64, error) {
	result, count, err := r.listItemsFromDB(ctx, criteria, start, limit)
	if err != nil {
		return nil, 0, err
	}

	res := make([]*app.WorkItem, len(result))

	for index, value := range result {
		wiType, err := r.wir.loadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, InternalError{simpleError{err.Error()}}
		}
		res[index], err = convertFromModel(*wiType, result[index])
		if err != nil {
			return nil, 0, ConversionError{simpleError{err.Error()}}
		}
	}

	return res, count, nil
}

func convertFromModel(wiType WorkItemType, workItem WorkItem) (*app.WorkItem, error) {
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
