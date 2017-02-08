package workitem

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/rendering"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	Load(ctx context.Context, ID string) (*app.WorkItem, error)
	Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error)
	Reorder(ctx context.Context, before string, wi app.WorkItem) (*app.WorkItem, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error)
}

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	db  *gorm.DB
	wir *GormWorkItemTypeRepository
}

// LoadFromDB returns the work item with the given ID in model representation.
func (r *GormWorkItemRepository) LoadFromDB(ID string) (*WorkItem, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", ID)
	}
	res := WorkItem{}
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, errors.NewNotFoundError("work item", ID)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res, nil
}

// Load returns the work item for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) Load(ctx context.Context, ID string) (*app.WorkItem, error) {
	res, err := r.LoadFromDB(ID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	wiType, err := r.wir.LoadTypeFromDB(res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return convertWorkItemModelToApp(wiType, res)
}

// LoadHighestOrder returns the highest order
func (r *GormWorkItemRepository) LoadHighestOrder() (float64, error) {
	res := WorkItem{}
	tx := r.db.Order("fields->'order' desc").Last(&res)
	if tx.RecordNotFound() {
		return 0, nil
	}
	if tx.Error != nil {
		return 0, errors.NewInternalError(tx.Error.Error())
	}
	order, err := strconv.ParseFloat(fmt.Sprintf("%v", res.Fields[SystemOrder]), 64)
	if err != nil {
		return 0, errors.NewInternalError(err.Error())
	}
	return order, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemRepository) Delete(ctx context.Context, ID string) error {
	var workItem = WorkItem{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treat as not found: clients don't know it must be a number
		return errors.NewNotFoundError("work item", ID)
	}
	workItem.ID = id
	tx := r.db.Delete(workItem)

	if err = tx.Error; err != nil {
		return errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item", ID)
	}

	return nil
}

// Reorder places the to-be-reordered workitem(s) above the input workitem.
// The order of workitems are spaced by a factor of 1000.
// The new order of workitem := (order of previousitem + order of nextitem)/2
// Version must be the same as the one int the stored version
func (r *GormWorkItemRepository) Reorder(ctx context.Context, before string, wi app.WorkItem) (*app.WorkItem, error) {
	var order float64
	res := WorkItem{}
	beforeItem := WorkItem{}
	afterItem := WorkItem{}
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if res.Version != wi.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.wir.LoadTypeFromDB(wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	if before != "" {
		beforeId, err := strconv.ParseUint(before, 10, 64)
		if err != nil || beforeId == 0 {
			return nil, errors.NewNotFoundError("work item", before)
		}
		tx = r.db.First(&beforeItem, beforeId)
		if tx.RecordNotFound() {
			return nil, errors.NewNotFoundError("work item", string(beforeId))
		}
		if tx.Error != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		beforeOrder, err := strconv.ParseFloat(fmt.Sprintf("%v", beforeItem.Fields[SystemOrder]), 64)
		if err != nil {
			return nil, errors.NewBadParameterError("data.attributes.order", res.Fields[SystemOrder])
		}
		tx2 := r.db.Where("fields -> 'order' < ?", beforeItem.Fields[SystemOrder]).Order("fields->'order' desc", true).Last(&afterItem)
		if afterItem.ID == 0 {
			// The item is moved to first position
			order = (0 + beforeOrder) / 2
		} else {
			afterOrder, err := strconv.ParseFloat(fmt.Sprintf("%v", afterItem.Fields[SystemOrder]), 64)
			if err != nil {
				return nil, errors.NewBadParameterError("data.attributes.order", res.Fields[SystemOrder])
			}
			if tx2.RecordNotFound() {
				return nil, errors.NewNotFoundError("work item", before)
			}
			if tx2.Error != nil {
				return nil, errors.NewInternalError(err.Error())
			}
			order = (beforeOrder + afterOrder) / 2
		}
	} else {
		// the item is moved at last position

		tx2 := r.db.Order("fields->'order' desc", true).Last(&afterItem)
		if tx2.RecordNotFound() {
			return nil, errors.NewNotFoundError("work item", string(afterItem.ID))
		}
		if tx2.Error != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		afterOrder, _ := strconv.ParseFloat(fmt.Sprintf("%v", afterItem.Fields[SystemOrder]), 64)
		order = afterOrder + 1000
	}

	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}

	wi.Fields[SystemOrder] = order
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt {
			continue
		}
		fieldValue := wi.Fields[fieldName]
		var err error
		res.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}

	tx = tx.Where("Version = ?", wi.Version).Save(&res)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	return convertWorkItemModelToApp(wiType, &res)
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemRepository) Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error) {
	res := WorkItem{}
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	log.Printf("looking for id %d", id)
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if res.Version != wi.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.wir.LoadTypeFromDB(wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt {
			continue
		}
		fieldValue := wi.Fields[fieldName]
		var err error
		res.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}

	tx = tx.Where("Version = ?", wi.Version).Save(&res)
	if err := tx.Error; err != nil {
		log.Print(err.Error())
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	log.Printf("updated item to %v\n", res)
	return convertWorkItemModelToApp(wiType, &res)
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemRepository) Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error) {
	wiType, err := r.wir.LoadTypeFromDB(typeID)
	if err != nil {
		return nil, errors.NewBadParameterError("type", typeID)
	}
	wi := WorkItem{
		Type:   typeID,
		Fields: Fields{},
	}
	// Order
	position, err := r.LoadHighestOrder()
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	fields[SystemOrder] = position + 1000

	fields[SystemCreator] = creator
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt {
			continue
		}
		fieldValue := fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
		if fieldName == SystemDescription && wi.Fields[fieldName] != nil {
			description := rendering.NewMarkupContentFromMap(wi.Fields[fieldName].(map[string]interface{}))
			if !rendering.IsMarkupSupported(description.Markup) {
				return nil, errors.NewBadParameterError(fieldName, fieldValue)
			}
		}
	}
	tx := r.db
	if err = tx.Create(&wi).Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return convertWorkItemModelToApp(wiType, &wi)
}

func convertWorkItemModelToApp(wiType *WorkItemType, wi *WorkItem) (*app.WorkItem, error) {
	result, err := wiType.ConvertFromModel(*wi)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	if _, ok := wiType.Fields[SystemCreatedAt]; ok {
		result.Fields[SystemCreatedAt] = wi.CreatedAt
	}
	return result, nil

}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormWorkItemRepository) listItemsFromDB(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]WorkItem, uint64, error) {
	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return nil, 0, errors.NewBadParameterError("expression", criteria)
	}

	log.Printf("executing query: '%s' with params %v", where, parameters)

	db := r.db.Model(&WorkItem{}).Where(where, parameters...)
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
	db = db.Select("count(*) over () as cnt2 , *").Order("fields->'order'")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	defer rows.Close()

	result := []WorkItem{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, errors.NewInternalError(err.Error())
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
		value := WorkItem{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(err.Error())
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
			return nil, 0, errs.WithStack(err)
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
		return nil, 0, errs.WithStack(err)
	}

	res := make([]*app.WorkItem, len(result))

	for index, value := range result {
		wiType, err := r.wir.LoadTypeFromDB(value.Type)
		if err != nil {
			return nil, 0, errors.NewInternalError(err.Error())
		}
		res[index], err = convertWorkItemModelToApp(wiType, &value)
	}

	return res, count, nil
}
