package link

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/workitem"
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

// WorkItemLinkRepository encapsulates storage & retrieval of work item links
type WorkItemLinkRepository interface {
	Create(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID) (*app.WorkItemLink, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLink, error)
	List(ctx context.Context) (*app.WorkItemLinkArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLink) (*app.WorkItemLink, error)
}

// NewWorkItemLinkRepository creates a work item link repository based on gorm
func NewWorkItemLinkRepository(db *gorm.DB) *GormWorkItemLinkRepository {
	return &GormWorkItemLinkRepository{
		db:                   db,
		workItemRepo:         workitem.NewWorkItemRepository(db),
		workItemTypeRepo:     workitem.NewWorkItemTypeRepository(db),
		workItemLinkTypeRepo: NewWorkItemLinkTypeRepository(db),
	}
}

// GormWorkItemLinkRepository implements WorkItemLinkRepository using gorm
type GormWorkItemLinkRepository struct {
	db                   *gorm.DB
	workItemRepo         *workitem.GormWorkItemRepository
	workItemTypeRepo     *workitem.GormWorkItemTypeRepository
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
		return errors.NewBadParameterError("source work item type", source.Type)
	}
	if !targetWorkItemType.IsTypeOrSubtypeOf(linkType.TargetTypeName) {
		return errors.NewBadParameterError("target work item type", target.Type)
	}
	return nil
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID) (*app.WorkItemLink, error) {
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
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(*link)
	return &result, nil
}

// Load returns the work item link for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Load(ctx context.Context, ID string) (*app.WorkItemLink, error) {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return nil, errors.NewNotFoundError("work item link", ID)
	}
	log.Printf("loading work item link %s", id.String())
	res := WorkItemLink{}
	//db := r.db.Model(&res).Where("id=?", ID).First(&res)
	db := r.db.Where("id=?", id).Find(&res)
	if db.RecordNotFound() {
		log.Printf("not found work item link, res=%v", res)
		return nil, errors.NewNotFoundError("work item link", id.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(res)
	return &result, nil
}

// List returns all work item links
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) List(ctx context.Context) (*app.WorkItemLinkArray, error) {
	// We don't have any where clause or paging at the moment.
	var rows []WorkItemLink
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
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
func (r *GormWorkItemLinkRepository) Delete(ctx context.Context, ID string) error {
	id, err := satoriuuid.FromString(ID)
	if err != nil {
		// treat as not found: clients don't know it must be a UUID
		return errors.NewNotFoundError("work item link", ID)
	}
	var link = WorkItemLink{
		ID: id,
	}
	log.Printf("work item link to delete %v\n", link)
	db := r.db.Delete(&link)
	if db.Error != nil {
		log.Print(db.Error.Error())
		return errors.NewInternalError(db.Error.Error())
	}
	if db.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link", id.String())
	}
	return nil
}

// Save updates the given work item link in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Save(ctx context.Context, lt app.WorkItemLink) (*app.WorkItemLink, error) {
	res := WorkItemLink{}
	if lt.Data.ID == nil {
		return nil, errors.NewBadParameterError("work item link", nil)
	}
	db := r.db.Model(&res).Where("id=?", *lt.Data.ID).First(&res)
	if db.RecordNotFound() {
		log.Printf("work item link not found, res=%v", res)
		return nil, errors.NewNotFoundError("work item link", *lt.Data.ID)
	}
	if db.Error != nil {
		log.Print(db.Error.Error())
		return nil, errors.NewInternalError(db.Error.Error())
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, errors.NewVersionConflictError("version conflict")
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
		return nil, errors.NewInternalError(db.Error.Error())
	}
	log.Printf("updated work item link to %v\n", res)
	result := ConvertLinkFromModel(res)
	return &result, nil
}
