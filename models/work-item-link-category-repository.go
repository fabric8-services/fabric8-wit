package models

import (
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

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
func (r *GormWorkItemLinkCategoryRepository) Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategory, error) {
	if name == nil || *name == "" {
		return nil, NewBadParameterError("name", name)
	}
	created := WorkItemLinkCategory{
		// Omit "lifecycle" and "ID" fields as they will be filled by the DB
		Name:        *name,
		Description: description,
	}
	db := r.db.Create(&created)
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	// Convert the created link category entry into a JSONAPI response
	result := ConvertLinkCategoryFromModel(created)
	return &result, nil
}

// Load returns the work item link category for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Load(ctx context.Context, ID string) (*app.WorkItemLinkCategory, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NewNotFoundError("work item link category", ID)
	}
	log.Printf("loading work item link category %s", id.String())
	res := WorkItemLinkCategory{}
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NewNotFoundError("work item link category", id.String())
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}

	// Convert the created link category entry into a JSONAPI response
	result := ConvertLinkCategoryFromModel(res)
	return &result, nil
}

// LoadCategoryFromDB return work item link category for the name
func (r *GormWorkItemLinkCategoryRepository) LoadCategoryFromDB(ctx context.Context, name string) (*WorkItemLinkCategory, error) {
	log.Printf("loading work item link category %s", name)
	res := WorkItemLinkCategory{}
	db := r.db.Model(&res).Where("name=?", name).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NewNotFoundError("work item link category", name)
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	return &res, nil
}

// List returns all work item link categories
// TODO: Handle pagination
func (r *GormWorkItemLinkCategoryRepository) List(ctx context.Context) (*app.WorkItemLinkCategoryArray, error) {
	var rows []WorkItemLinkCategory
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	res := app.WorkItemLinkCategoryArray{}
	res.Data = make([]*app.WorkItemLinkCategoryData, len(rows))
	for index, value := range rows {
		cat := ConvertLinkCategoryFromModel(value)
		res.Data[index] = cat.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkCategoryArrayMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link category with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NewNotFoundError("work item link category", ID)
	}

	var cat = WorkItemLinkCategory{
		ID: id,
	}

	log.Printf("work item link category to delete %v\n", cat)

	db := r.db.Delete(&cat)
	if db.Error != nil {
		return NewInternalError(db.Error.Error())
	}

	if db.RowsAffected == 0 {
		return NewNotFoundError("work item link category", id.String())
	}
	return nil
}

// Save updates the given work item link category in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Save(ctx context.Context, linkCat app.WorkItemLinkCategory) (*app.WorkItemLinkCategory, error) {
	res := WorkItemLinkCategory{}
	if linkCat.Data.ID == nil {
		return nil, NewBadParameterError("data.id", linkCat.Data.ID)
	}
	id, err := satoriuuid.FromString(*linkCat.Data.ID)
	if err != nil {
		//log.Printf("Error when converting %s to UUID: %s", *linkCat.Data.ID, err.Error())
		// treat as not found: clients don't know it must be a UUID
		return nil, NewNotFoundError("work item link category", id.String())
	}

	if linkCat.Data.Type != EndpointWorkItemLinkCategories {
		return nil, NewBadParameterError("data.type", linkCat.Data.Type).Expected(EndpointWorkItemLinkCategories)
	}

	// If the name is not nil, it MUST NOT be empty
	if linkCat.Data.Attributes.Name != nil && *linkCat.Data.Attributes.Name == "" {
		return nil, NewBadParameterError("data.attributes.name", *linkCat.Data.Attributes.Name)
	}

	db := r.db.Model(&res).Where("id=?", *linkCat.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("work item link category not found, res=%v", res)
		return nil, NewNotFoundError("work item link category", id.String())
	}
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	if linkCat.Data.Attributes.Version == nil || res.Version != *linkCat.Data.Attributes.Version {
		return nil, NewVersionConflictError("version conflict")
	}

	newLinkCat := WorkItemLinkCategory{
		ID:      id,
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
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	log.Printf("updated work item link category to %v\n", newLinkCat)
	result := ConvertLinkCategoryFromModel(newLinkCat)
	return &result, nil
}
