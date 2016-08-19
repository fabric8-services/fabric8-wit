package models

import (
	"log"

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
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, name string) (*app.WorkItemType, error) {
	res, err := r.loadTypeFromDB(ctx, name)
	if err != nil {
		return nil, err
	}

	result := convertTypeFromModels(res)
	return &result, nil
}

func (r *GormWorkItemTypeRepository) loadTypeFromDB(ctx context.Context, name string) (*WorkItemType, error) {
	log.Printf("loading work item type %s", name)
	res := WorkItemType{}

	if r.ts.tx.Where("name=?", name).First(&res).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"work item type", name}
	}
	if err := r.ts.tx.Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}

	return &res, nil
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, extendedTypeName *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error) {
	allFields := map[string]FieldDefinition{}
	path := "/"

	if extendedTypeName != nil {
		extendedType := WorkItemType{}
		if r.ts.tx.First(&extendedType, extendedTypeName).RecordNotFound() {
			log.Printf("not found, res=%v", extendedType)
			return nil, BadParameterError{parameter: "extendedTypeName", value: *extendedTypeName}
		}
		if err := r.ts.tx.Error; err != nil {
			return nil, InternalError{simpleError{err.Error()}}
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.ParentPath + "/" + extendedType.Name
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

	result := convertTypeFromModels(&created)
	return &result, nil
}

func convertTypeFromModels(t *WorkItemType) app.WorkItemType {
	var converted = app.WorkItemType{
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
