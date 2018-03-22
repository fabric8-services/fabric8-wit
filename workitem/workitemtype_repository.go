package workitem

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var cache = NewWorkItemTypeCache()

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	repository.Exister
	Load(ctx context.Context, id uuid.UUID) (*WorkItemType, error)
	Create(ctx context.Context, spaceTemplateID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields FieldDefinitions, canConstruct bool) (*WorkItemType, error)
	List(ctx context.Context, spaceTemplateID uuid.UUID, start *int, length *int) ([]WorkItemType, error)
	ListPlannerItemTypes(ctx context.Context, spaceTemplateID uuid.UUID) ([]WorkItemType, error)
	AddChildTypes(ctx context.Context, parentTypeID uuid.UUID, childTypeIDs []uuid.UUID) error
}

// NewWorkItemTypeRepository creates a wi type repository based on gorm
func NewWorkItemTypeRepository(db *gorm.DB) *GormWorkItemTypeRepository {
	return &GormWorkItemTypeRepository{db}
}

// GormWorkItemTypeRepository implements WorkItemTypeRepository using gorm
type GormWorkItemTypeRepository struct {
	db *gorm.DB
}

// Load returns the work item for the given spaceID and id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "load"}, time.Now())
	log.Debug(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Loading work item type")
	res, ok := cache.Get(id)
	if !ok {
		log.Info(ctx, map[string]interface{}{
			"wit_id": id,
		}, "Work item type doesn't exist in the cache. Loading from DB...")
		res = WorkItemType{}

		db := r.db.Model(&res).Where("id=?", id).First(&res)
		if db.RecordNotFound() {
			log.Info(ctx, map[string]interface{}{
				"wit_id": id,
			}, "work item type not found")
			return nil, errors.NewNotFoundError("work item type", id.String())
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		childTypes, err := r.loadChildTypeList(ctx, res.ID)
		if err != nil {
			return nil, errs.Wrapf(err, `failed to load child types for WIT "%s" (%s)`, res.Name, res.ID)
		}
		res.ChildTypeIDs = childTypes
		cache.Put(res)
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemTypeRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "exists"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Checking if work item type exists")

	_, exists := cache.Get(id)
	if exists {
		return nil
	}
	return repository.CheckExists(ctx, r.db, WorkItemType{}.TableName(), id)
}

// ClearGlobalWorkItemTypeCache removes all work items from the global cache
func ClearGlobalWorkItemTypeCache() {
	cache.Clear()
}

// Create creates a new work item type in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, spaceTemplateID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields FieldDefinitions, canConstruct bool) (*WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "create"}, time.Now())
	// Make sure this WIT has an ID
	if id == nil {
		tmpID := uuid.NewV4()
		id = &tmpID
	}

	allFields := map[string]FieldDefinition{}
	path := LtreeSafeID(*id)
	if extendedTypeID != nil && *extendedTypeID != uuid.Nil {
		extendedType := WorkItemType{}
		db := r.db.Model(&extendedType).Where("id=?", extendedTypeID).First(&extendedType)
		if db.RecordNotFound() {
			return nil, errors.NewBadParameterError("extendedTypeID", *extendedTypeID)
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.Path + pathSep + path
	}
	// now process new fields, checking whether they are already there.
	for field, definition := range fields {
		existing, exists := allFields[field]
		if exists && !compatibleFields(existing, definition) {
			return nil, errs.Errorf("incompatible change for field %s", field)
		}
		allFields[field] = definition
	}

	model := WorkItemType{
		Version:         0,
		ID:              *id,
		Name:            name,
		Description:     description,
		Icon:            icon,
		Path:            path,
		Fields:          allFields,
		SpaceTemplateID: spaceTemplateID,
		CanConstruct:    canConstruct,
	}

	db := r.db.Create(&model)
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	return &model, nil
}

// ListPlannerItemTypes returns work item types that derives from PlannerItem type
func (r *GormWorkItemTypeRepository) ListPlannerItemTypes(ctx context.Context, spaceTemplateID uuid.UUID) ([]WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "listPlannerItems"}, time.Now())
	var wits []WorkItemType
	db := r.db.Select("id").Where("space_template_id = ? AND path::text LIKE '"+path.ConvertToLtree(SystemPlannerItem)+".%'", spaceTemplateID.String()).Order("name")
	if err := db.Find(&wits).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_template_id": spaceTemplateID,
			"err":               err,
		}, "unable to list the work item types that derive of planner item")
		return nil, errs.WithStack(err)
	}
	for i, wit := range wits {
		childTypes, err := r.loadChildTypeList(ctx, wit.ID)
		if err != nil {
			return nil, errs.Wrapf(err, `failed to load child types for WIT "%s" (%s)`, wit.Name, wit.ID)
		}
		wits[i].ChildTypeIDs = childTypes
	}
	return wits, nil

}

// List returns work item types selected by the given criteria.Expression,
// starting with start (zero-based) and returning at most "limit" item types.
func (r *GormWorkItemTypeRepository) List(ctx context.Context, spaceTemplateID uuid.UUID, start *int, limit *int) ([]WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "list"}, time.Now())

	// check space template exists
	if err := spacetemplate.NewRepository(r.db).CheckExists(ctx, spaceTemplateID); err != nil {
		return nil, errors.NewNotFoundError("space template", spaceTemplateID.String())
	}

	// Currently we don't implement filtering here, so leave this empty
	// TODO: (kwk) implement criteria parsing just like for work items
	var wits []WorkItemType
	db := r.db.Where("space_template_id = ?", spaceTemplateID).Order("name")
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&wits).Error; err != nil {
		return nil, errs.WithStack(err)
	}
	for i, wit := range wits {
		childTypes, err := r.loadChildTypeList(ctx, wit.ID)
		if err != nil {
			return nil, errs.Wrapf(err, `failed to load child types for WIT "%s" (%s)`, wit.Name, wit.ID)
		}
		wits[i].ChildTypeIDs = childTypes
	}
	return wits, nil
}

// ChildType models the relationship from one parent work item type to its child
// types.
type ChildType struct {
	gormsupport.Lifecycle
	ID                   uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	ParentWorkItemTypeID uuid.UUID `sql:"type:uuid"`
	ChildWorkItemTypeID  uuid.UUID `sql:"type:uuid"`
	Position             int       // position in type list of child types
}

// TableName implements gorm.tabler
func (wit ChildType) TableName() string {
	return "work_item_child_types"
}

// AddChildTypes adds the given child work item types to the parent work item
// type.
func (r *GormWorkItemTypeRepository) AddChildTypes(ctx context.Context, parentTypeID uuid.UUID, childTypeIDs []uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "add_child_types"}, time.Now())
	if len(childTypeIDs) <= 0 {
		return nil
	}
	// Create entries for each child in the type list
	for idx, ID := range childTypeIDs {
		childType := ChildType{
			ParentWorkItemTypeID: parentTypeID,
			ChildWorkItemTypeID:  ID,
			Position:             idx,
		}
		db := r.db.Create(&childType)
		if db.Error != nil {
			return errors.NewInternalError(ctx, db.Error)
		}
	}
	ClearGlobalWorkItemTypeCache()
	return nil

}

// loadChildTypeList loads all child work item types associated with the given
// work item type
func (r *GormWorkItemTypeRepository) loadChildTypeList(ctx context.Context, parentTypeID uuid.UUID) ([]uuid.UUID, error) {
	types := []ChildType{}
	db := r.db.Model(&types).Where("parent_work_item_type_id=?", parentTypeID).Order("position ASC").Find(&types)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{"wit_id": parentTypeID}, "work item type child types not found")
		return nil, errors.NewNotFoundError("work item type child types", parentTypeID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	res := make([]uuid.UUID, len(types))
	for i, childType := range types {
		res[i] = childType.ChildWorkItemTypeID
	}
	return res, nil
}
