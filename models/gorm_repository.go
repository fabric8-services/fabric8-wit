package models

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models/criteria"
)

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	ts *GormTransactionSupport
}

// NewRepository constructs a WorkItemRepository
func NewWorkItemRepository(ts *GormTransactionSupport) *GormWorkItemRepository {
	return &GormWorkItemRepository{ts}
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
	if r.ts.tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item", ID}
	}
	wiType, err := loadTypeFromDB(res.Type)
	if err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	result, err := convertFromModel(*wiType, res)
	if err != nil {
		return nil, ConversionError{simpleError{err.Error()}}
	}
	return result, nil
}

func loadTypeFromDB(id string) (*WorkItemType, error) {
	if wellKnown[id] == nil {
		return nil, fmt.Errorf("Work item type not found: %s", id)
	}
	return wellKnown[id], nil
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
	tx := r.ts.tx

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
	tx := r.ts.tx
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "work item", ID: wi.ID}
	}
	if res.Version != wi.Version {
		return nil, VersionConflictError{simpleError{"version conflict"}}
	}

	wiType, err := loadTypeFromDB(wi.Type)
	if err != nil {
		return nil, BadParameterError{"Type", wi.Type}
	}

	newWi := WorkItem{
		ID:      id,
		Name:    wi.Name,
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
func (r *GormWorkItemRepository) Create(ctx context.Context, typeID string, name string, fields map[string]interface{}) (*app.WorkItem, error) {
	wiType, err := loadTypeFromDB(typeID)
	if err != nil {
		return nil, BadParameterError{parameter: "type", value: typeID}
	}
	wi := WorkItem{
		Name:   name,
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
	tx := r.ts.tx

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

func (r *GormWorkItemRepository) List(ctx context.Context, criteria criteria.Expression, start int, length int) ([]*app.WorkItem, error) {
	where, parameters, err := Compile(criteria)
	if err != nil {
		return nil, BadParameterError{"expression", criteria}
	}

	var rows []WorkItem
	if err := r.ts.tx.Where(where, parameters).Offset(start).Limit(length).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*app.WorkItem, len(rows))

	for index, value := range rows {
		var err error
		wiType, err := loadTypeFromDB(value.Type)
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

var wellKnown = map[string]*WorkItemType{
	"1": {
		ID:   1,
		Name: "system.workitem",
		Fields: map[string]FieldDefinition{
			"system.owner": FieldDefinition{Type: SimpleType{Kind: KindUser}, Required: true},
			"system.state": FieldDefinition{Type: SimpleType{Kind: KindString}, Required: true},
		}}}

func convertFromModel(wiType WorkItemType, workItem WorkItem) (*app.WorkItem, error) {
	result := app.WorkItem{
		ID:      strconv.FormatUint(workItem.ID, 10),
		Name:    workItem.Name,
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
