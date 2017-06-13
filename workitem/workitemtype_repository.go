package workitem

import (
	"fmt"

	"context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/path"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var cache = NewWorkItemTypeCache()

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	Load(ctx context.Context, spaceID uuid.UUID, id uuid.UUID) (*WorkItemType, error)
	Create(ctx context.Context, spaceID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields map[string]FieldDefinition) (*WorkItemType, error)
	List(ctx context.Context, spaceID uuid.UUID, start *int, length *int) ([]WorkItemType, error)
	ListPlannerItems(ctx context.Context, spaceID uuid.UUID) ([]WorkItemType, error)
}

// NewWorkItemTypeRepository creates a wi type repository based on gorm
func NewWorkItemTypeRepository(db *gorm.DB) *GormWorkItemTypeRepository {
	return &GormWorkItemTypeRepository{db}
}

// GormWorkItemTypeRepository implements WorkItemTypeRepository using gorm
type GormWorkItemTypeRepository struct {
	db *gorm.DB
}

// LoadByID returns the work item for the given id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) LoadByID(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	res, err := r.LoadTypeFromDB(ctx, id)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return res, nil
}

// Exists returns true if the work item with the given ID exists
func (r *GormWorkItemTypeRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	log.Info(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Checking if work item type exists")
	_, exists := cache.Get(id)
	if exists {
		return true, nil
	}
	log.Info(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Work item type doesn't exist in the cache. Checking in DB...")
	query := fmt.Sprintf("SELECT EXISTS(SELECT id FROM %s WHERE id=$1 AND deleted_at IS NULL)", WorkItemType{}.TableName())
	err := r.db.CommonDB().QueryRow(query, id.String()).Scan(&exists)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"wit_id": id,
			"err":    err,
		}, "failed to check if work item type exists")
		return false, errs.Wrap(err, "failed to check if work item type exists")
	}
	return exists, nil
}

// Load returns the work item for the given spaceID and id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, spaceID uuid.UUID, id uuid.UUID) (*WorkItemType, error) {
	log.Info(ctx, map[string]interface{}{
		"wit_id":   id,
		"space_id": spaceID,
	}, "Loading work item type")
	res, ok := cache.Get(id)
	if !ok {
		log.Info(ctx, map[string]interface{}{
			"wit_id":   id,
			"space_id": spaceID,
		}, "Work item type doesn't exist in the cache. Loading from DB...")
		res = WorkItemType{}

		db := r.db.Model(&res).Where("id=? AND space_id=?", id, spaceID).First(&res)
		if db.RecordNotFound() {
			log.Error(ctx, map[string]interface{}{
				"wit_id":   id,
				"space_id": spaceID,
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

// LoadTypeFromDB return work item type for the given id
func (r *GormWorkItemTypeRepository) LoadTypeFromDB(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	log.Info(ctx, map[string]interface{}{
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
			log.Error(ctx, map[string]interface{}{
				"wit_id": id,
			}, "work item type not found")
			return nil, errors.NewNotFoundError("work item type", id.String())
		}
		if err := db.Error; err != nil {
			log.Error(ctx, map[string]interface{}{
				"witID": id,
			}, "work item type retrieval error", err.Error())
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
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, spaceID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields map[string]FieldDefinition) (*WorkItemType, error) {
	// Make sure this WIT has an ID
	if id == nil {
		tmpID := uuid.NewV4()
		id = &tmpID
	}

	allFields := map[string]FieldDefinition{}
	path := LtreeSafeID(*id)
	if extendedTypeID != nil {
		extendedType := WorkItemType{}
		db := r.db.Model(&extendedType).Where("id=?", extendedTypeID).First(&extendedType)
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
	// now process new fields, checking whether they are already there.
	for field, definition := range fields {
		existing, exists := allFields[field]
		if exists && !compatibleFields(existing, definition) {
			return nil, fmt.Errorf("incompatible change for field %s", field)
		}
		allFields[field] = definition
	}

	created := WorkItemType{
		Version:     0,
		ID:          *id,
		Name:        name,
		Description: description,
		Icon:        icon,
		Path:        path,
		Fields:      allFields,
		SpaceID:     spaceID,
	}

	if err := r.db.Create(&created).Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}

	log.Debug(ctx, map[string]interface{}{"witID": created.ID}, "Work item type created successfully!")
	return &created, nil
}

// List returns work item types that derives from PlannerItem type
func (r *GormWorkItemTypeRepository) ListPlannerItems(ctx context.Context, spaceID uuid.UUID) ([]WorkItemType, error) {
	var rows []WorkItemType
	path := path.Path{}
	db := r.db.Select("id").Where("space_id = ? AND path::text LIKE '"+path.ConvertToLtree(SystemPlannerItem)+".%'", spaceID.String())

	if err := db.Find(&rows).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err,
		}, "unable to list the work item types that derive of planner item")
		return nil, errs.WithStack(err)
	}
	return rows, nil
}

// List returns work item types selected by the given criteria.Expression,
// starting with start (zero-based) and returning at most "limit" item types.
func (r *GormWorkItemTypeRepository) List(ctx context.Context, spaceID uuid.UUID, start *int, limit *int) ([]WorkItemType, error) {
	// Currently we don't implement filtering here, so leave this empty
	// TODO: (kwk) implement criteria parsing just like for work items
	var rows []WorkItemType
	db := r.db.Where("space_id = ?", spaceID)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, errs.WithStack(err)
	}
	return rows, nil
}
