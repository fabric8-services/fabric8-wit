package workitem

import (
	"fmt"
	"reflect"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var cache = NewWorkItemTypeCache()

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	Load(ctx context.Context, id uuid.UUID) (*app.WorkItemTypeSingle, error)
	Create(ctx context.Context, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, fields map[string]app.FieldDefinition) (*app.WorkItemTypeSingle, error)
	List(ctx context.Context, start *int, length *int) (*app.WorkItemTypeList, error)
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
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, id uuid.UUID) (*app.WorkItemTypeSingle, error) {
	res, err := r.LoadTypeFromDB(ctx, id)
	if err != nil {
		return nil, errs.WithStack(err)
	}

	result := convertTypeFromModels(res)
	return &app.WorkItemTypeSingle{Data: &result}, nil
}

// LoadTypeFromDB return work item type for the given id
func (r *GormWorkItemTypeRepository) LoadTypeFromDB(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	log.Logger().Infoln("Loading work item type", id)
	res, ok := cache.Get(id)
	if !ok {
		log.Info(ctx, map[string]interface{}{
			"witID": id,
		}, "Work item type doesn't exist in the cache. Loading from DB...")
		res = WorkItemType{}

		db := r.db.Model(&res).Where("id=?", id).First(&res)
		if db.RecordNotFound() {
			log.Error(ctx, map[string]interface{}{
				"witID": id,
			}, "work item type not found")
			return nil, errors.NewNotFoundError("work item type", id.String())
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		cache.Put(res)
	}

	return &res, nil
}

// ClearGlobalWorkItemTypeCache removes all work items from the global cache
func ClearGlobalWorkItemTypeCache() {
	cache.Clear()
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, fields map[string]app.FieldDefinition) (*app.WorkItemTypeSingle, error) {
	// Make sure this WIT has an ID
	if id == nil {
		tmpID := uuid.NewV4()
		id = &tmpID
	}

	existing, _ := r.LoadTypeFromDB(ctx, *id)
	if existing != nil {
		log.Error(ctx, map[string]interface{}{"witID": *id}, "unable to create new work item type")
		return nil, errors.NewBadParameterError("name", *id)
	}
	allFields := map[string]FieldDefinition{}
	path := LtreeSafeID(*id)
	if extendedTypeID != nil {
		extendedType := WorkItemType{}
		db := r.db.First(&extendedType, "id = ?", *extendedTypeID)
		if db.RecordNotFound() {
			return nil, errors.NewBadParameterError("extendedTypeID", *extendedTypeID)
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.Path + pathSep + path
	}

	// now process new fields, checking whether they are ok to add.
	for field, definition := range fields {
		existing, exists := allFields[field]
		ct, err := convertFieldTypeToModels(*definition.Type)
		if err != nil {
			return nil, errs.WithStack(err)
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
		Version:     0,
		ID:          *id,
		Name:        name,
		Description: description,
		Path:        path,
		Fields:      allFields,
	}

	if err := r.db.Create(&created).Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}

	result := convertTypeFromModels(&created)

	log.Debug(ctx, map[string]interface{}{"witID": created.ID}, "Work item type created successfully!")

	return &app.WorkItemTypeSingle{Data: &result}, nil
}

// List returns work item types selected by the given criteria.Expression, starting with start (zero-based) and returning at most "limit" item types
func (r *GormWorkItemTypeRepository) List(ctx context.Context, start *int, limit *int) (*app.WorkItemTypeList, error) {
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
		return nil, errs.WithStack(err)
	}
	result := &app.WorkItemTypeList{}
	result.Data = make([]*app.WorkItemTypeData, len(rows))

	for index, value := range rows {
		wit := convertTypeFromModels(&value)
		result.Data[index] = &wit
	}

	return result, nil
}

func compatibleFields(existing FieldDefinition, new FieldDefinition) bool {
	return reflect.DeepEqual(existing, new)
}

// converts from models to app representation
func convertTypeFromModels(t *WorkItemType) app.WorkItemTypeData {
	id := t.ID
	var converted = app.WorkItemTypeData{
		Type: "workitemtypes",
		ID:   &id,
		Attributes: &app.WorkItemTypeAttributes{
			Version:     t.Version,
			Description: t.Description,
			Name:        t.Name,
			Fields:      map[string]*app.FieldDefinition{},
		},
	}
	for name, def := range t.Fields {
		ct := convertFieldTypeFromModels(def.Type)
		converted.Attributes.Fields[name] = &app.FieldDefinition{
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
	case KindString, KindInteger, KindFloat, KindInstant, KindDuration, KindURL, KindWorkitemReference, KindUser, KindEnum, KindList, KindIteration, KindMarkup, KindArea:
		return &kind, nil
	}
	return nil, fmt.Errorf("Not a simple type")
}

func convertFieldTypeToModels(t app.FieldType) (FieldType, error) {
	kind, err := convertStringToKind(t.Kind)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	switch *kind {
	case KindList:
		componentType, err := convertAnyToKind(*t.ComponentType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !componentType.isSimpleType() {
			return nil, fmt.Errorf("Component type is not list type: %T", componentType)
		}
		return ListType{SimpleType{*kind}, SimpleType{*componentType}}, nil
	case KindEnum:
		bt, err := convertAnyToKind(*t.BaseType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !bt.isSimpleType() {
			return nil, fmt.Errorf("baseType type is not list type: %T", bt)
		}
		baseType := SimpleType{*bt}

		values := t.Values
		converted, err := convertList(func(ft FieldType, element interface{}) (interface{}, error) {
			return ft.ConvertToModel(element)
		}, baseType, values)
		if err != nil {
			return nil, errs.WithStack(err)
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
			return nil, errs.WithStack(err)
		}
		converted := FieldDefinition{
			Required: definition.Required,
			Type:     ct,
		}
		allFields[field] = converted
	}
	return allFields, nil
}
