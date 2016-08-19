package models

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
)

// GormWorkItemTypeRepository implements WorkItemTypeRepository using gorm
type GormWorkItemTypeRepository struct {
	ts *GormTransactionSupport
}

// NewWorkItemTypeRepository constructs a WorkItemTypeRepository
func NewWorkItemTypeRepository(ts *GormTransactionSupport) *GormWorkItemTypeRepository {
	return &GormWorkItemTypeRepository{ts}
}

// Load returns the work item for the given id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, ID string) (*app.WorkItemType, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"work item", ID}
	}

	log.Printf("loading work item %d", id)
	res := WorkItemType{}

	if r.ts.tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item", ID}
	}
	if err := r.ts.tx.Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	result := convertTypeFromModels(res)
	return &result, nil
}

func (r *GormWorkItemTypeRepository) loadTypeFromDB(ctx context.Context, ID string) (*WorkItemType, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"work item", ID}
	}

	log.Printf("loading work item %d", id)
	res := WorkItemType{}

	if r.ts.tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item", ID}
	}
	if err := r.ts.tx.Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	return &res, nil
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, extendedTypeID *string, name string, fields map[string]FieldDefinition) (*app.WorkItemType, error) {
	allFields := map[string]FieldDefinition{}
	path := "/"

	if extendedTypeID != nil {
		id, err := strconv.ParseUint(*extendedTypeID, 10, 64)
		if err != nil {
			return nil, BadParameterError{parameter: "extendedTypeId", value: *extendedTypeID}
		}
		extendedType := WorkItemType{}
		if r.ts.tx.First(&extendedType, id).RecordNotFound() {
			log.Printf("not found, res=%v", extendedType)
			return nil, BadParameterError{parameter: "extendedTypeId", value: *extendedTypeID}
		}
		if err := r.ts.tx.Error; err != nil {
			return nil, InternalError{simpleError{err.Error()}}
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.ParentPath + "/" + strconv.FormatUint(extendedType.ID, 10)
	}

	// new process new fields, checking whether they are ok to add.

	created := WorkItemType{
		Version:    0,
		Name:       name,
		ParentPath: path,
		Fields:     allFields,
	}

	if err := r.ts.tx.Create(&created).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	result := convertTypeFromModels(created)
	return &result, nil
}

func convertTypeFromModels(t WorkItemType) app.WorkItemType {
	var converted = app.WorkItemType{
		ID:      strconv.FormatUint(t.ID, 10),
		Name:    t.Name,
		Version: t.Version,
		Fields:  map[string]*app.FieldDefinition{},
	}
	for name, def := range t.Fields {
		converted.Fields[name] = &app.FieldDefinition{
			Required: def.Required,
			Type:     def.Type,
		}
	}
	return converted
}

var wellKnown = map[string]*WorkItemType{
	"1": &WorkItemType{
		ID:   1,
		Name: "system.workitem",
		Fields: map[string]FieldDefinition{
			"system.owner": FieldDefinition{Type: SimpleType{Kind: KindUser}, Required: true},
			"system.state": FieldDefinition{Type: SimpleType{Kind: KindString}, Required: true},
		}}}
