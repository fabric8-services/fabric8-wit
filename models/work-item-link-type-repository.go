package models

import (
	"fmt"
	"log"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

// NewWorkItemLinkTypeRepository creates a work item link type repository based on gorm
func NewWorkItemLinkTypeRepository(db *gorm.DB) *GormWorkItemLinkTypeRepository {
	return &GormWorkItemLinkTypeRepository{db}
}

// GormWorkItemLinkTypeRepository implements WorkItemLinkTypeRepository using gorm
type GormWorkItemLinkTypeRepository struct {
	db *gorm.DB
}

// Create creates a new work item link type in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Create(ctx context.Context, linkType *WorkItemLinkType) (*app.WorkItemLinkType, error) {
	if err := linkType.CheckValidForCreation(); err != nil {
		return nil, err
	}

	// Check link category exists
	linkCategory := WorkItemLinkCategory{}
	db := r.db.Where("id=?", linkType.LinkCategoryID).Find(&linkCategory)
	if db.RecordNotFound() {
		return nil, NewBadParameterError("work item link category", linkType.LinkCategoryID)
	} else if db.Error != nil {
		return nil, NewInternalError(fmt.Sprintf("Failed to find work item link category: %s", db.Error.Error()))
	}

	db = r.db.Create(linkType)
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(linkType)
	return &result, nil
}

// Load returns the work item link type for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Load(ctx context.Context, ID string) (*app.WorkItemLinkType, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NewNotFoundError("work item link type", ID)
	}
	log.Printf("loading work item link type %s", id.String())
	res := WorkItemLinkType{}
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found work item link type, res=%v", res)
		return nil, NewNotFoundError("work item link type", id.String())
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkTypeFromModel(&res)

	return &result, nil
}

// LoadTypeFromDB return work item link type for the name
func (r *GormWorkItemLinkTypeRepository) LoadTypeFromDB(ctx context.Context, name string, categoryId satoriuuid.UUID) (*WorkItemLinkType, error) {
	log.Printf("loading work item link type %s with category ID", name, categoryId.String())
	res := WorkItemLinkType{}
	db := r.db.Model(&res).Where("name=? AND link_category_id=?", name, categoryId.String()).First(&res)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NewNotFoundError("work item link type", name)
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	return &res, nil
}

// List returns all work item link types
// TODO: Handle pagination
func (r *GormWorkItemLinkTypeRepository) List(ctx context.Context) (*app.WorkItemLinkTypeArray, error) {
	// We don't have any where clause or paging at the moment.
	var rows []WorkItemLinkType
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	res := app.WorkItemLinkTypeArray{}
	res.Data = make([]*app.WorkItemLinkTypeData, len(rows))
	for index, value := range rows {
		linkType := ConvertLinkTypeFromModel(&value)
		res.Data[index] = linkType.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkTypeArrayMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link type with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkTypeRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NewNotFoundError("work item link type", ID)
	}
	var cat = WorkItemLinkType{
		ID: id,
	}
	log.Printf("work item link type to delete %v\n", cat)
	db := r.db.Delete(&cat)
	if db.Error != nil {
		return NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return NewNotFoundError("work item link type", id.String())
	}
	return nil
}

// Save updates the given work item link type in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkTypeRepository) Save(ctx context.Context, lt app.WorkItemLinkType) (*app.WorkItemLinkType, error) {
	res := WorkItemLinkType{}
	if lt.Data.ID == nil {
		return nil, NewBadParameterError("work item link type", nil)
	}
	db := r.db.Model(&res).Where("id=?", *lt.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("work item link type not found, res=%v", res)
		return nil, NewNotFoundError("work item link type", *lt.Data.ID)
	}
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, NewVersionConflictError("version conflict")
	}
	if err := ConvertLinkTypeToModel(&lt, &res); err != nil {
		return nil, err
	}
	res.Version = res.Version + 1
	db = db.Save(&res)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	log.Printf("updated work item link type to %v\n", res)
	result := ConvertLinkTypeFromModel(&res)
	return &result, nil
}
