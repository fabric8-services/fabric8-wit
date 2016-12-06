package workitem

import (
	"fmt"
	"log"
	"reflect"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/jinzhu/gorm"
)

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	Load(ctx context.Context, name string) (*app.WorkItemType, error)
	Create(ctx context.Context, extendedTypeID *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error)
	List(ctx context.Context, start *int, length *int) ([]*app.WorkItemType, error)
}

// NewWorkItemRepository creates a wi repository based on gorm
func NewWorkItemRepository(db *gorm.DB) *GormWorkItemRepository {
	return &GormWorkItemRepository{db, &GormWorkItemTypeRepository{db}}
}

// NewWorkItemTypeRepository creates a wi type repository based on gorm
func NewWorkItemTypeRepository(db *gorm.DB) *GormWorkItemTypeRepository {
	return &GormWorkItemTypeRepository{db}
}

// GormWorkItemTypeRepository implements WorkItemTypeRepository using gorm
type GormWorkItemTypeRepository struct {
	db *gorm.DB
}

// Load returns the work item for the given id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, name string) (*app.WorkItemType, error) {
	res, err := r.LoadTypeFromDB(name)
	if err != nil {
		return nil, err
	}

	result := convertTypeFromModels(res)
	return &result, nil
}

// LoadTypeFromDB return work item type for the given id
func (r *GormWorkItemTypeRepository) LoadTypeFromDB(name string) (*WorkItemType, error) {
	log.Printf("loading work item type %s", name)
	res := WorkItemType{}

	db := r.db.Model(&res).Where("name=?", name).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, errors.NewNotFoundError("work item type", name)
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}

	return &res, nil
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, extendedTypeName *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error) {
	existing, _ := r.LoadTypeFromDB(name)
	if existing != nil {
		log.Printf("creating type %s again", name)
		return nil, errors.NewBadParameterError("name", name)
	}
	allFields := map[string]FieldDefinition{}
	path := pathSep + name
	if extendedTypeName != nil {
		extendedType := WorkItemType{}
		db := r.db.First(&extendedType, extendedTypeName)
		if db.RecordNotFound() {
			log.Printf("not found, res=%v", extendedType)
			return nil, errors.NewBadParameterError("extendedTypeName", *extendedTypeName)
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.Path + pathSep + name
	}

	// now process new fields, checking whether they are ok to add.
	for field, definition := range fields {
		existing, exists := allFields[field]
		ct, err := convertFieldTypeToModels(*definition.Type)
		if err != nil {
			return nil, err
		}
		converted := FieldDefinition{
			Required: definition.Required,
			Type:     ct,
		}
		if exists && !compatibleFields(existing, converted) {
			return nil, fmt.Errorf("incompatible change for field %s", field)
		}
		allFields[field] = converted
	}

	created := WorkItemType{
		Version: 0,
		Name:    name,
		Path:    path,
		Fields:  allFields,
	}

	if err := r.db.Save(&created).Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}

	result := convertTypeFromModels(&created)
	return &result, nil
}

// List returns work item types selected by the given criteria.Expression, starting with start (zero-based) and returning at most "limit" item types
func (r *GormWorkItemTypeRepository) List(ctx context.Context, start *int, limit *int) ([]*app.WorkItemType, error) {
	// Currently we don't implement filtering here, so leave this empty
	// TODO: (kwk) implement criteria parsing just like for work items
	var where string
	var parameters []interface{}

	var rows []WorkItemType
	db := r.db.Where(where, parameters...)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*app.WorkItemType, len(rows))

	for index, value := range rows {
		wit := convertTypeFromModels(&value)
		result[index] = &wit
	}

	return result, nil
}

func compatibleFields(existing FieldDefinition, new FieldDefinition) bool {
	return reflect.DeepEqual(existing, new)
}

// converts from models to app representation
func convertTypeFromModels(t *WorkItemType) app.WorkItemType {
	var converted = app.WorkItemType{
		Name:    t.Name,
		Version: t.Version,
		Fields:  map[string]*app.FieldDefinition{},
	}
	for name, def := range t.Fields {
		ct := convertFieldTypeFromModels(def.Type)
		converted.Fields[name] = &app.FieldDefinition{
			Required: def.Required,
			Type:     &ct,
		}
	}
	return converted
}

// converts the field type from modesl to app representation
func convertFieldTypeFromModels(t FieldType) app.FieldType {
	result := app.FieldType{}
	result.Kind = string(t.GetKind())
	switch t2 := t.(type) {
	case ListType:
		kind := string(t2.ComponentType.GetKind())
		result.ComponentType = &kind
	case EnumType:
		kind := string(t2.BaseType.GetKind())
		result.BaseType = &kind
		result.Values = t2.Values
	}

	return result
}

func convertAnyToKind(any interface{}) (*Kind, error) {
	k, ok := any.(string)
	if !ok {
		return nil, fmt.Errorf("kind is not a string value %v", any)
	}

	return convertStringToKind(k)
}

func convertStringToKind(k string) (*Kind, error) {
	kind := Kind(k)
	switch kind {
	case KindString, KindInteger, KindFloat, KindInstant, KindDuration, KindURL, KindWorkitemReference, KindUser, KindEnum, KindList:
		return &kind, nil
	}
	return nil, fmt.Errorf("Not a simple type")
}

func convertFieldTypeToModels(t app.FieldType) (FieldType, error) {
	kind, err := convertStringToKind(t.Kind)
	if err != nil {
		return nil, err
	}
	switch *kind {
	case KindList:
		componentType, err := convertAnyToKind(*t.ComponentType)
		if err != nil {
			return nil, err
		}
		if !componentType.isSimpleType() {
			return nil, fmt.Errorf("Component type is not list type: %s", componentType)
		}
		return ListType{SimpleType{*kind}, SimpleType{*componentType}}, nil
	case KindEnum:
		bt, err := convertAnyToKind(*t.BaseType)
		if err != nil {
			return nil, err
		}
		if !bt.isSimpleType() {
			return nil, fmt.Errorf("baseType type is not list type: %s", bt)
		}
		baseType := SimpleType{*bt}

		values := t.Values
		converted, err := convertList(func(ft FieldType, element interface{}) (interface{}, error) {
			return ft.ConvertToModel(element)
		}, baseType, values)
		if err != nil {
			return nil, err
		}
		return EnumType{SimpleType{*kind}, baseType, converted}, nil
	default:
		return SimpleType{*kind}, nil
	}
}

func TEMPConvertFieldTypesToModel(fields map[string]app.FieldDefinition) (map[string]FieldDefinition, error) {

	allFields := map[string]FieldDefinition{}
	for field, definition := range fields {
		ct, err := convertFieldTypeToModels(*definition.Type)
		if err != nil {
			return nil, err
		}
		converted := FieldDefinition{
			Required: definition.Required,
			Type:     ct,
		}
		allFields[field] = converted
	}
	return allFields, nil
}
