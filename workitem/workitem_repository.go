package workitem

import (
	"strconv"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/rendering"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const orderValue = 1000

type DirectionType string

const (
	DirectionAbove  DirectionType = "above"
	DirectionBelow  DirectionType = "below"
	DirectionTop    DirectionType = "top"
	DirectionBottom DirectionType = "bottom"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	Load(ctx context.Context, ID string) (*app.WorkItem, error)
	Save(ctx context.Context, wi app.WorkItem, modifierID uuid.UUID) (*app.WorkItem, error)
	Reorder(ctx context.Context, direction DirectionType, targetID *string, wi app.WorkItem, modifierID uuid.UUID) (*app.WorkItem, error)
	Delete(ctx context.Context, ID string, suppressorID uuid.UUID) error
	Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*app.WorkItem, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error)
	Fetch(ctx context.Context, criteria criteria.Expression) (*app.WorkItem, error)
	GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error)
	GetCountsForIteration(ctx context.Context, iterationID uuid.UUID) (map[string]WICountsPerIteration, error)
}

// NewWorkItemRepository creates a GormWorkItemRepository
func NewWorkItemRepository(db *gorm.DB) *GormWorkItemRepository {
	repository := &GormWorkItemRepository{db, &GormWorkItemTypeRepository{db}, &GormRevisionRepository{db}}
	return repository
}

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	db   *gorm.DB
	witr *GormWorkItemTypeRepository
	wirr *GormRevisionRepository
}

// ************************************************
// WorkItemRepository implementation
// ************************************************

// LoadFromDB returns the work item with the given ID in model representation.
func (r *GormWorkItemRepository) LoadFromDB(ctx context.Context, workitemID string) (*WorkItem, error) {
	id, err := strconv.ParseUint(workitemID, 10, 64)
	if err != nil || id == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", workitemID)
	}
	log.Info(nil, map[string]interface{}{
		"wiID": workitemID,
	}, "Loading work item")

	res := WorkItem{}
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wiID": workitemID,
		}, "work item not found")
		return nil, errors.NewNotFoundError("work item", workitemID)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &res, nil
}

// Load returns the work item for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) Load(ctx context.Context, ID string) (*app.WorkItem, error) {
	res, err := r.LoadFromDB(ctx, ID)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, res)
}

// LoadTopWorkitem returns top most work item of the list. Top most workitem has the Highest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadTopWorkitem(ctx context.Context) (*app.WorkItem, error) {
	res := WorkItem{}
	db := r.db.Model(WorkItem{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s)",
		WorkItem{}.TableName(),
	)
	db = db.Where(query).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &res)
}

// LoadBottomWorkitem returns bottom work item of the list. Bottom most workitem has the lowest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadBottomWorkitem(ctx context.Context) (*app.WorkItem, error) {
	res := WorkItem{}
	db := r.db.Model(WorkItem{})
	query := fmt.Sprintf("execution_order = (SELECT min(execution_order) FROM %[1]s)",
		WorkItem{}.TableName(),
	)
	db = db.Where(query).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &res)
}

// LoadHighestOrder returns the highest order
func (r *GormWorkItemRepository) LoadHighestOrder() (float64, error) {
	res := WorkItem{}
	db := r.db.Model(WorkItem{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s)",
		WorkItem{}.TableName(),
	)
	db = db.Where(query).First(&res)
	order, err := strconv.ParseFloat(fmt.Sprintf("%v", res.ExecutionOrder), 64)
	if err != nil {
		return 0, errors.NewInternalError(err.Error())
	}
	return order, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemRepository) Delete(ctx context.Context, workitemID string, suppressorID uuid.UUID) error {
	var workItem = WorkItem{}
	id, err := strconv.ParseUint(workitemID, 10, 64)
	if err != nil || id == 0 {
		// treat as not found: clients don't know it must be a number
		return errors.NewNotFoundError("work item", workitemID)
	}
	workItem.ID = id
	// retrieve the current version of the work item to delete
	r.db.Select("id, version, type").Where("id = ?", workItem.ID).Find(&workItem)
	// delete the work item
	tx := r.db.Delete(workItem)
	if err = tx.Error; err != nil {
		return errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item", workitemID)
	}
	// store a revision of the deleted work item
	err = r.wirr.Create(context.Background(), suppressorID, RevisionTypeDelete, workItem)
	if err != nil {
		return errs.Wrapf(err, "error while deleting work item")
	}
	log.Debug(ctx, map[string]interface{}{"wiID": workitemID}, "Work item deleted successfully!")
	return nil
}

// Calculates the order of the reorder workitem
func (r *GormWorkItemRepository) CalculateOrder(above, below *float64) float64 {
	return (*above + *below) / 2
}

// Reordering a workitem requires order of two closest workitems: above and below.
// FindSecondItem returns the order of the second workitem required to reorder.
// If direction == "above", then
//	FindFirstItem returns the value above which reorder item has to be placed
//      FindSecondItem returns the value below which reorder item has to be placed
// If direction == "below", then
//	FindFirstItem returns the value below which reorder item has to be placed
//      FindSecondItem returns the value above which reorder item has to be placed
func (r *GormWorkItemRepository) FindSecondItem(order *float64, secondItemDirection DirectionType) (*string, *float64, error) {
	Item := WorkItem{}
	var tx *gorm.DB
	switch secondItemDirection {
	case DirectionAbove:
		// Finds the item above which reorder item has to be placed
		tx = r.db.Where("execution_order < ?", order).Order("execution_order desc", true).Last(&Item)

	case DirectionBelow:
		// Finds the item below which reorder item has to be placed
		tx = r.db.Where("execution_order > ?", order).Order("execution_order", true).Last(&Item)
	default:
		return nil, nil, nil
	}
	if tx.RecordNotFound() {
		// Item is placed at first or last position
		ItemId := strconv.FormatUint(Item.ID, 10)
		return &ItemId, nil, nil
	}
	if tx.Error != nil {
		return nil, nil, errors.NewInternalError(tx.Error.Error())
	}

	ItemId := strconv.FormatUint(Item.ID, 10)
	return &ItemId, &Item.ExecutionOrder, nil

}

// FindFirstItem returns the order of the target workitem
func (r *GormWorkItemRepository) FindFirstItem(id string) (*float64, error) {
	Item := WorkItem{}
	Id, err := strconv.ParseUint(id, 10, 64)
	if err != nil || Id == 0 {
		return nil, errors.NewNotFoundError("work item", string(Id))
	}
	tx := r.db.First(&Item, Id)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", id)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &Item.ExecutionOrder, nil
}

// Reorder places the to-be-reordered workitem above the input workitem.
// The order of workitems are spaced by a factor of 1000.
// The new order of workitem := (order of previousitem + order of nextitem)/2
// Version must be the same as the one int the stored version
func (r *GormWorkItemRepository) Reorder(ctx context.Context, direction DirectionType, targetID *string, wi app.WorkItem, modifierID uuid.UUID) (*app.WorkItem, error) {
	var order float64
	res := WorkItem{}

	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if res.Version != wi.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.witr.LoadTypeFromDB(ctx, wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	switch direction {
	case DirectionBelow:
		// if direction == "below", place the reorder item **below** the workitem having id equal to targetID
		aboveItemOrder, err := r.FindFirstItem(*targetID)
		if aboveItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", *targetID)
		}
		belowItemId, belowItemOrder, err := r.FindSecondItem(aboveItemOrder, DirectionAbove)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", *targetID)
		}
		if *belowItemId == "0" {
			// Item is placed at last position
			belowItemOrder := float64(0)
			order = r.CalculateOrder(aboveItemOrder, &belowItemOrder)
		} else if *belowItemId == strconv.FormatUint(res.ID, 10) {
			// When same reorder request is made again
			order = wi.ExecutionOrder
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionAbove:
		// if direction == "above", place the reorder item **above** the workitem having id equal to targetID
		belowItemOrder, _ := r.FindFirstItem(*targetID)
		if belowItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", *targetID)
		}
		aboveItemId, aboveItemOrder, err := r.FindSecondItem(belowItemOrder, DirectionBelow)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", *targetID)
		}
		if *aboveItemId == "0" {
			// Item is placed at first position
			order = *belowItemOrder + float64(orderValue)
		} else if *aboveItemId == strconv.FormatUint(res.ID, 10) {
			// When same reorder request is made again
			order = wi.ExecutionOrder
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionTop:
		// if direction == "top", place the reorder item at the topmost position. Now, the reorder item has the highest order in the whole list.
		res, err := r.LoadTopWorkitem(ctx)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.ExecutionOrder
		} else {
			topItemOrder := res.Fields[SystemOrder].(float64)
			order = topItemOrder + orderValue
		}
	case DirectionBottom:
		// if direction == "bottom", place the reorder item at the bottom most position. Now, the reorder item has the lowest order in the whole list
		res, err := r.LoadBottomWorkitem(ctx)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.ExecutionOrder
		} else {
			bottomItemOrder := res.Fields[SystemOrder].(float64)
			order = bottomItemOrder / 2
		}
	default:
		return &wi, nil
	}
	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}

	res.ExecutionOrder = order

	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
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
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, res)
	if err != nil {
		return nil, err
	}
	return convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &res)
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemRepository) Save(ctx context.Context, wi app.WorkItem, modifierID uuid.UUID) (*app.WorkItem, error) {
	res := WorkItem{}
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	log.Info(ctx, map[string]interface{}{
		"wiID": wi.ID,
	}, "Looking for id for the work item repository")
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wiID": wi.ID,
		}, "work item repository not found")
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if res.Version != wi.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.witr.LoadTypeFromDB(ctx, wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}
	res.ExecutionOrder = wi.ExecutionOrder
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
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
		log.Error(ctx, map[string]interface{}{
			"wiID": wi.ID,
			"err":  err,
		}, "unable to save the work item repository")
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, res)
	if err != nil {
		return nil, errs.Wrapf(err, "error while saving work item")
	}
	log.Info(ctx, map[string]interface{}{
		"wiID": wi.ID,
	}, "Updated work item repository")
	return convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &res)
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemRepository) Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*app.WorkItem, error) {
	wiType, err := r.witr.LoadTypeFromDB(ctx, typeID)
	if err != nil {
		return nil, errors.NewBadParameterError("typeID", typeID)
	}

	// The order of workitems are spaced by a factor of 1000.
	pos, err := r.LoadHighestOrder()
	if err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	pos = pos + orderValue
	wi := WorkItem{
		Type:           typeID,
		Fields:         Fields{},
		ExecutionOrder: pos,
		SpaceID:        spaceID,
	}
	fields[SystemCreator] = creatorID.String()
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
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
		return nil, errs.Wrapf(err, "failed to create work item")
	}

	witem, err := convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &wi)
	if err != nil {
		return nil, err
	}
	// store a revision of the created work item
	err = r.wirr.Create(context.Background(), creatorID, RevisionTypeCreate, wi)
	if err != nil {
		return nil, errs.Wrapf(err, "error while creating work item")
	}
	log.Debug(ctx, map[string]interface{}{"pkg": "workitem", "wiID": wi.ID}, "Work item created successfully!")
	return witem, nil
}

func convertWorkItemModelToApp(request *goa.RequestData, wiType *WorkItemType, wi *WorkItem) (*app.WorkItem, error) {
	result, err := wiType.ConvertFromModel(request, *wi)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	if _, ok := wiType.Fields[SystemCreatedAt]; ok {
		result.Fields[SystemCreatedAt] = wi.CreatedAt
	}
	if _, ok := wiType.Fields[SystemUpdatedAt]; ok {
		result.Fields[SystemUpdatedAt] = wi.UpdatedAt
	}
	if _, ok := wiType.Fields[SystemOrder]; ok {
		result.Fields[SystemOrder] = wi.ExecutionOrder
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

	log.Info(ctx, map[string]interface{}{
		"where":      where,
		"parameters": parameters,
	}, "Executing query : '%s' with params %v", where, parameters)

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
	db = db.Select("count(*) over () as cnt2 , *").Order("execution_order desc")

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
		wiType, err := r.witr.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, errors.NewInternalError(err.Error())
		}
		res[index], err = convertWorkItemModelToApp(goa.ContextRequest(ctx), wiType, &value)
	}
	return res, count, nil
}

// Fetch fetches the (first) work item matching by the given criteria.Expression.
func (r *GormWorkItemRepository) Fetch(ctx context.Context, criteria criteria.Expression) (*app.WorkItem, error) {
	limit := 1
	results, count, err := r.List(ctx, criteria, nil, &limit)
	if err != nil {
		return nil, err
	}
	// if no result
	if count == 0 {
		return nil, nil
	}
	// one result
	result := results[0]
	return result, nil
}

// GetCountsPerIteration fetches WI count from DB and returns a map of iterationID->WICountsPerIteration
// This function executes following query to fetch 'closed' and 'total' counts of the WI for each iteration in given spaceID
// 	SELECT iterations.id as IterationId, count(*) as Total,
// 		count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed
// 		FROM "work_items" left join iterations
// 		on fields@> concat('{"system.iteration": "', iterations.id, '"}')::jsonb
// 		WHERE (iterations.space_id = '33406de1-25f1-4969-bcec-88f29d0a7de3'
// 		and work_items.deleted_at IS NULL) GROUP BY IterationId
func (r *GormWorkItemRepository) GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error) {
	var res []WICountsPerIteration
	db := r.db.Table("work_items").Select(`iterations.id as IterationId, count(*) as Total,
				count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed`).Joins(`left join iterations
				on fields@> concat('{"system.iteration": "', iterations.id, '"}')::jsonb`).Where(`iterations.space_id = ?
				and work_items.deleted_at IS NULL`, spaceID).Group(`IterationId`).Scan(&res)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	countsMap := map[string]WICountsPerIteration{}
	for _, iterationWithCount := range res {
		countsMap[iterationWithCount.IterationId] = iterationWithCount
	}
	return countsMap, nil
}

// GetCountsForIteration returns Closed and Total counts of WI for given iteration
// It executes
// SELECT count(*) as Total, count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed FROM "work_items" where fields@> concat('{"system.iteration": "%s"}')::jsonb and work_items.deleted_at is null
func (r *GormWorkItemRepository) GetCountsForIteration(ctx context.Context, iterationID uuid.UUID) (map[string]WICountsPerIteration, error) {
	var res WICountsPerIteration
	query := fmt.Sprintf(`SELECT count(*) as Total,
						count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed
						FROM "work_items"
						where fields@> concat('{"system.iteration": "%s"}')::jsonb
						and work_items.deleted_at is null`, iterationID)
	db := r.db.Raw(query)
	db.Scan(&res)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	countsMap := map[string]WICountsPerIteration{}
	countsMap[iterationID.String()] = WICountsPerIteration{
		IterationId: iterationID.String(),
		Closed:      res.Closed,
		Total:       res.Total,
	}
	return countsMap, nil
}
