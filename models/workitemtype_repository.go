package models

import (
	"fmt"
	"log"
	"reflect"

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
