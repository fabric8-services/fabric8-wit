package models

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

const (
	EndpointWorkItemTypes          = "workitemtypes"
	EndpointWorkItems              = "workitems"
	EndpointWorkItemLinkCategories = "workitemlinkcategories"
	EndpointWorkItemLinkTypes      = "workitemlinktypes"
	EndpointWorkItemLinks          = "workitemlinks"
)

// NewWorkItemLinkRepository creates a work item link repository based on gorm
func NewWorkItemLinkRepository(db *gorm.DB) *GormWorkItemLinkRepository {
	return &GormWorkItemLinkRepository{
		db:                   db,
		workItemRepo:         NewWorkItemRepository(db),
		workItemTypeRepo:     NewWorkItemTypeRepository(db),
		workItemLinkTypeRepo: NewWorkItemLinkTypeRepository(db),
	}
}

// GormWorkItemLinkRepository implements WorkItemLinkRepository using gorm
type GormWorkItemLinkRepository struct {
	db                   *gorm.DB
	workItemRepo         *GormWorkItemRepository
	workItemTypeRepo     *GormWorkItemTypeRepository
	workItemLinkTypeRepo *GormWorkItemLinkTypeRepository
}

// ValidateCorrectSourceAndTargetType returns an error if the Path of
// the source WIT as defined by the work item link type is not part of
// the actual source's WIT; the same applies for the target.
func (r *GormWorkItemLinkRepository) ValidateCorrectSourceAndTargetType(sourceID, targetID uint64, linkTypeID satoriuuid.UUID) error {
	linkType, err := r.workItemLinkTypeRepo.LoadTypeFromDBByID(linkTypeID)
	if err != nil {
		return err
	}
	// Fetch the source work item
	source, err := r.workItemRepo.LoadFromDB(strconv.FormatUint(sourceID, 10))
	if err != nil {
		return err
	}
	// Fetch the target work item
	target, err := r.workItemRepo.LoadFromDB(strconv.FormatUint(targetID, 10))
	if err != nil {
		return err
	}
	// Fetch the concrete work item types of the target and the source.
	sourceWorkItemType, err := r.workItemTypeRepo.LoadTypeFromDB(source.Type)
	if err != nil {
		return err
	}
	targetWorkItemType, err := r.workItemTypeRepo.LoadTypeFromDB(target.Type)
	if err != nil {
		return err
	}
	// Check type paths
	if !sourceWorkItemType.IsTypeOrSubtypeOf(linkType.SourceTypeName) {
		return NewBadParameterError("source work item type", source.Type)
	}
	if !targetWorkItemType.IsTypeOrSubtypeOf(linkType.TargetTypeName) {
		return NewBadParameterError("target work item type", target.Type)
	}
	return nil
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, wiIDStr *string, sourceID, targetID uint64, linkTypeID satoriuuid.UUID) (*app.WorkItemLink, error) {
	wi, err := checkWorkItemExists(r.db, wiIDStr)
	if err != nil {
		return nil, err
	}
	if wiIDStr != nil {
		// Check that the source is the same as the work item ID
		if sourceID != wi.ID {
			return nil, NewBadParameterError("work item link source", sourceID).Expected(wi.ID)
		}
	}
	link := &WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	if err := link.CheckValidForCreation(); err != nil {
		return nil, err
	}
	if err := r.ValidateCorrectSourceAndTargetType(sourceID, targetID, linkTypeID); err != nil {
		return nil, err
	}
	db := r.db.Create(link)
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(*link)
	return &result, nil
}

// Load returns the work item link for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Load(ctx context.Context, wiIDStr *string, ID string) (*app.WorkItemLink, error) {
	if _, err := checkWorkItemExists(r.db, wiIDStr); err != nil {
		return nil, err
	}
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, NewNotFoundError("work item link", ID)
	}
	log.Printf("loading work item link %s", id.String())
	res := WorkItemLink{}
	//db := r.db.Model(&res).Where("id=?", ID).First(&res)
	db := r.db.Where("id=?", id).Find(&res)
	if db.RecordNotFound() {
		log.Printf("not found work item link, res=%v", res)
		return nil, NewNotFoundError("work item link", id.String())
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(res)
	return &result, nil
}

// List returns all work item links if wiID is nil; otherwise the work item links are returned
// that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) List(ctx context.Context, wiIDStr *string) (*app.WorkItemLinkArray, error) {
	var rows []WorkItemLink
	db := r.db
	if wiIDStr == nil {
		// When no work item ID is given, return all links
		db = db.Find(&rows)
		if db.Error != nil {
			return nil, db.Error
		}
	} else {
		// When work item ID is given, filter by it
		wi, err := checkWorkItemExists(r.db, wiIDStr)
		if err != nil {
			return nil, err
		}
		// Now fetch all links for that work item
		db = r.db.Model(&WorkItemLink{}).Where("? IN (source_id, target_id)", wi.ID).Find(&rows)
		if db.Error != nil {
			return nil, db.Error
		}
	}

	res := app.WorkItemLinkArray{}
	res.Data = make([]*app.WorkItemLinkData, len(rows))
	for index, value := range rows {
		cat := ConvertLinkFromModel(value)
		res.Data[index] = cat.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkArrayMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// Delete deletes the work item link with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkRepository) Delete(ctx context.Context, wiIDStr *string, ID string) error {
	if _, err := checkWorkItemExists(r.db, wiIDStr); err != nil {
		return err
	}
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return NewNotFoundError("work item link", ID)
	}
	var link = WorkItemLink{
		ID: id,
	}
	log.Printf("work item link to delete %v\n", link)
	db := r.db.Delete(&link)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return NewNotFoundError("work item link", id.String())
	}
	return nil
}

// Save updates the given work item link in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Save(ctx context.Context, wiIDStr *string, lt app.WorkItemLink) (*app.WorkItemLink, error) {
	_, err := checkWorkItemExists(r.db, wiIDStr)
	if err != nil {
		return nil, err
	}
	if wiIDStr != nil {
		// Check that the source is the same as the work item ID
		if lt.Data.Relationships.Source.Data.ID != *wiIDStr {
			return nil, NewBadParameterError("work item link source", lt.Data.Relationships.Source.Data.ID).Expected(*wiIDStr)
		}
	}

	res := WorkItemLink{}
	if lt.Data.ID == nil {
		return nil, NewBadParameterError("work item link", nil)
	}
	db := r.db.Model(&res).Where("id=?", *lt.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("work item link not found, res=%v", res)
		return nil, NewNotFoundError("work item link", *lt.Data.ID)
	}
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, NewVersionConflictError("version conflict")
	}
	if err := ConvertLinkToModel(lt, &res); err != nil {
		return nil, err
	}
	res.Version = res.Version + 1
	if err := r.ValidateCorrectSourceAndTargetType(res.SourceID, res.TargetID, res.LinkTypeID); err != nil {
		return nil, err
	}
	db = r.db.Save(&res)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, NewInternalError(db.Error.Error())
	}
	log.Printf("updated work item link to %v\n", res)
	result := ConvertLinkFromModel(res)
	return &result, nil
}

// checkWorkItemExists returns nil if no work item ID string is given or if work
// item ID string could be converted into a number and looked up in the
// database; otherwise it returns an error.
func checkWorkItemExists(db *gorm.DB, wiIDStr *string) (*WorkItem, error) {
	if db == nil {
		return nil, NewInternalError("db must not be nil")
	}
	if wiIDStr == nil {
		return nil, nil
	}
	// When work item ID is given, filter by it
	wiID, err := strconv.ParseUint(*wiIDStr, 10, 64)
	if err != nil {
		return nil, NewBadParameterError("work item id", *wiIDStr)
	}
	// Check that work item exists or return NotFoundError
	wi := WorkItem{}
	db = db.Model(&WorkItem{}).Where("id=?", wiID).Find(&wi)
	if db.RecordNotFound() {
		return nil, NewNotFoundError("work item", *wiIDStr)
	}
	if db.Error != nil {
		return nil, NewInternalError(db.Error.Error())
	}
	return &wi, nil
}
