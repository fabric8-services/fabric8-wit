package link

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategoryRepository encapsulates storage & retrieval of work item link categories
type WorkItemLinkCategoryRepository interface {
	Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategorySingle, error)
	Load(ctx context.Context, ID uuid.UUID) (*app.WorkItemLinkCategorySingle, error)
	List(ctx context.Context) (*app.WorkItemLinkCategoryList, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Save(ctx context.Context, linkCat app.WorkItemLinkCategorySingle) (*app.WorkItemLinkCategorySingle, error)
}

// NewWorkItemLinkCategoryRepository creates a work item link category repository based on gorm
func NewWorkItemLinkCategoryRepository(db *gorm.DB) *GormWorkItemLinkCategoryRepository {
	return &GormWorkItemLinkCategoryRepository{db}
}

// GormWorkItemLinkCategoryRepository implements WorkItemLinkCategoryRepository using gorm
type GormWorkItemLinkCategoryRepository struct {
	db *gorm.DB
}

// Create creates a new work item link category in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategorySingle, error) {
	if name == nil || *name == "" {
		return nil, errors.NewBadParameterError("name", name)
	}
	created := WorkItemLinkCategory{
		// Omit "lifecycle" and "ID" fields as they will be filled by the DB
		Name:        *name,
		Description: description,
	}
	db := r.db.Create(&created)
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link category entry into a JSONAPI response
	result := ConvertLinkCategoryFromModel(created)
	return &result, nil
}

// Load returns the work item link category for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Load(ctx context.Context, ID uuid.UUID) (*app.WorkItemLinkCategorySingle, error) {
	log.Info(ctx, map[string]interface{}{
		"wilc_id": ID,
	}, "Loading work item link category")

	res := WorkItemLinkCategory{}
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": ID,
		}, "work item link category not found by id ", ID)
		return nil, errors.NewNotFoundError("work item link category", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}

	// Convert the created link category entry into a JSONAPI response
	result := ConvertLinkCategoryFromModel(res)
	return &result, nil
}

// LoadCategoryFromDB return work item link category for the name
func (r *GormWorkItemLinkCategoryRepository) LoadCategoryFromDB(ctx context.Context, name string) (*WorkItemLinkCategory, error) {
	log.Info(ctx, map[string]interface{}{
		"categoryName": name,
	}, "Loading work item link category: %s", name)

	res := WorkItemLinkCategory{}
	db := r.db.Model(&res).Where("name=?", name).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilcName": name,
		}, "work item link category not found")
		return nil, errors.NewNotFoundError("work item link category", name)
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &res, nil
}

// List returns all work item link categories
// TODO: Handle pagination
func (r *GormWorkItemLinkCategoryRepository) List(ctx context.Context) (*app.WorkItemLinkCategoryList, error) {
	var rows []WorkItemLinkCategory
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	res := app.WorkItemLinkCategoryList{}
	res.Data = make([]*app.WorkItemLinkCategoryData, len(rows))
	for index, value := range rows {
		cat := ConvertLinkCategoryFromModel(value)
		res.Data[index] = cat.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkCategoryListMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link category with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	var cat = WorkItemLinkCategory{
		ID: ID,
	}

	log.Info(ctx, map[string]interface{}{
		"wilc_id": ID,
	}, "Work item link category to delete")

	db := r.db.Delete(&cat)
	if db.Error != nil {
		return errors.NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link category", ID.String())
	}
	return nil
}

// Save updates the given work item link category in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Save(ctx context.Context, linkCat app.WorkItemLinkCategorySingle) (*app.WorkItemLinkCategorySingle, error) {
	res := WorkItemLinkCategory{}
	if linkCat.Data.ID == nil {
		return nil, errors.NewBadParameterError("data.id", linkCat.Data.ID)
	}
	ID := *linkCat.Data.ID

	// If the name is not nil, it MUST NOT be empty
	if linkCat.Data.Attributes.Name != nil && *linkCat.Data.Attributes.Name == "" {
		return nil, errors.NewBadParameterError("data.attributes.name", *linkCat.Data.Attributes.Name)
	}

	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": ID,
		}, "work item link category not found")
		return nil, errors.NewNotFoundError("work item link category", ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": ID,
			"err":     db.Error,
		}, "unable to find work item link category")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	if linkCat.Data.Attributes.Version == nil || res.Version != *linkCat.Data.Attributes.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	newLinkCat := WorkItemLinkCategory{
		ID:      ID,
		Version: *linkCat.Data.Attributes.Version + 1,
	}

	if linkCat.Data.Attributes.Name != nil {
		newLinkCat.Name = *linkCat.Data.Attributes.Name
	}
	if linkCat.Data.Attributes.Description != nil {
		newLinkCat.Description = linkCat.Data.Attributes.Description
	}

	db = db.Save(&newLinkCat)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": newLinkCat.ID,
			"err":     db.Error,
		}, "unable to save work item link category repository")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	log.Info(ctx, map[string]interface{}{
		"wilc_id":         newLinkCat.ID,
		"newLinkCategory": newLinkCat,
	}, "Work item link category updated")
	result := ConvertLinkCategoryFromModel(newLinkCat)
	return &result, nil
}
