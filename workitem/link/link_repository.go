package link

import (
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
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
	Create(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID, creatorID satoriuuid.UUID) (*app.WorkItemLinkSingle, error)
	Load(ctx context.Context, ID satoriuuid.UUID) (*app.WorkItemLinkSingle, error)
	List(ctx context.Context) (*app.WorkItemLinkList, error)
	ListByWorkItemID(ctx context.Context, wiIDStr string) (*app.WorkItemLinkList, error)
	DeleteRelatedLinks(ctx context.Context, wiIDStr string, suppressorID satoriuuid.UUID) error
	Delete(ctx context.Context, ID satoriuuid.UUID, suppressorID satoriuuid.UUID) error
	Save(ctx context.Context, linkCat app.WorkItemLinkSingle, modifierID satoriuuid.UUID) (*app.WorkItemLinkSingle, error)
}

// NewWorkItemLinkRepository creates a work item link repository based on gorm
func NewWorkItemLinkRepository(db *gorm.DB) *GormWorkItemLinkRepository {
	return &GormWorkItemLinkRepository{
		db:                   db,
		workItemRepo:         workitem.NewWorkItemRepository(db),
		workItemTypeRepo:     workitem.NewWorkItemTypeRepository(db),
		workItemLinkTypeRepo: NewWorkItemLinkTypeRepository(db),
		revisionRepo:         NewRevisionRepository(db),
	}
}

// GormWorkItemLinkRepository implements WorkItemLinkRepository using gorm
type GormWorkItemLinkRepository struct {
	db                   *gorm.DB
	workItemRepo         *workitem.GormWorkItemRepository
	workItemTypeRepo     *workitem.GormWorkItemTypeRepository
	workItemLinkTypeRepo *GormWorkItemLinkTypeRepository
	revisionRepo         *GormWorkItemLinkRevisionRepository
}

// ValidateCorrectSourceAndTargetType returns an error if the Path of
// the source WIT as defined by the work item link type is not part of
// the actual source's WIT; the same applies for the target.
func (r *GormWorkItemLinkRepository) ValidateCorrectSourceAndTargetType(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID) error {
	linkType, err := r.workItemLinkTypeRepo.LoadTypeFromDBByID(ctx, linkTypeID)
	if err != nil {
		return errs.WithStack(err)
	}
	// Fetch the source work item
	source, err := r.workItemRepo.LoadFromDB(ctx, strconv.FormatUint(sourceID, 10))
	if err != nil {
		return errs.WithStack(err)
	}
	// Fetch the target work item
	target, err := r.workItemRepo.LoadFromDB(ctx, strconv.FormatUint(targetID, 10))
	if err != nil {
		return errs.WithStack(err)
	}
	// Fetch the concrete work item types of the target and the source.
	sourceWorkItemType, err := r.workItemTypeRepo.LoadTypeFromDB(ctx, source.Type)
	if err != nil {
		return errs.WithStack(err)
	}
	targetWorkItemType, err := r.workItemTypeRepo.LoadTypeFromDB(ctx, target.Type)
	if err != nil {
		return errs.WithStack(err)
	}
	// Check type paths
	if !sourceWorkItemType.IsTypeOrSubtypeOf(linkType.SourceTypeID) {
		return errors.NewBadParameterError("source work item type", source.Type)
	}
	if !targetWorkItemType.IsTypeOrSubtypeOf(linkType.TargetTypeID) {
		return errors.NewBadParameterError("target work item type", target.Type)
	}
	return nil
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID, creatorID satoriuuid.UUID) (*app.WorkItemLinkSingle, error) {
	link := &WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	if err := link.CheckValidForCreation(); err != nil {
		return nil, errs.WithStack(err)
	}
	if err := r.ValidateCorrectSourceAndTargetType(ctx, sourceID, targetID, linkTypeID); err != nil {
		return nil, errs.WithStack(err)
	}
	db := r.db.Create(link)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "work_item_links_unique_idx") {
			// TODO(kwk): Make NewBadParameterError a variadic function to avoid this ugliness ;)
			return nil, errors.NewBadParameterError("data.relationships.source_id + data.relationships.target_id + data.relationships.link_type_id", sourceID).Expected("unique")
		}
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// save a revision of the created work item link
	if err := r.revisionRepo.Create(ctx, creatorID, RevisionTypeCreate, *link); err != nil {
		return nil, errs.Wrapf(err, "error while creating work item")
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(*link)
	return &result, nil
}

// Load returns the work item link for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Load(ctx context.Context, ID satoriuuid.UUID) (*app.WorkItemLinkSingle, error) {
	log.Info(ctx, map[string]interface{}{
		"wilID": ID,
	}, "Loading work item link")
	res := WorkItemLink{}
	db := r.db.Where("id=?", ID).Find(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilID": ID,
		}, "work item link not found")
		return nil, errors.NewNotFoundError("work item link", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// Convert the created link type entry into a JSONAPI response
	result := ConvertLinkFromModel(res)
	return &result, nil
}

type fetchLinksFunc func() ([]WorkItemLink, error)

func (r *GormWorkItemLinkRepository) list(ctx context.Context, fetchFunc fetchLinksFunc) (*app.WorkItemLinkList, error) {
	rows, err := fetchFunc()
	if err != nil {
		return nil, errs.WithStack(err)
	}
	res := app.WorkItemLinkList{}
	res.Data = make([]*app.WorkItemLinkData, len(rows))
	for index, value := range rows {
		cat := ConvertLinkFromModel(value)
		res.Data[index] = cat.Data
	}
	// TODO: When adding pagination, this must not be len(rows) but
	// the overall total number of elements from all pages.
	res.Meta = &app.WorkItemLinkListMeta{
		TotalCount: len(rows),
	}
	return &res, nil
}

// ListByWorkItemID returns the work item links that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) ListByWorkItemID(ctx context.Context, wiIDStr string) (*app.WorkItemLinkList, error) {
	fetchFunc := func() ([]WorkItemLink, error) {
		var rows []WorkItemLink
		wi, err := r.workItemRepo.LoadFromDB(ctx, wiIDStr)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		// Now fetch all links for that work item
		db := r.db.Model(&WorkItemLink{}).Where("? IN (source_id, target_id)", wi.ID).Find(&rows)
		if db.Error != nil {
			return nil, db.Error
		}
		return rows, nil
	}
	return r.list(ctx, fetchFunc)
}

// List returns all work item links if wiID is nil; otherwise the work item links are returned
// that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) List(ctx context.Context) (*app.WorkItemLinkList, error) {
	fetchFunc := func() ([]WorkItemLink, error) {
		var rows []WorkItemLink
		db := r.db.Find(&rows)
		if db.Error != nil {
			return nil, db.Error
		}
		return rows, nil
	}
	return r.list(ctx, fetchFunc)
}

// Delete deletes the work item link with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkRepository) Delete(ctx context.Context, linkID satoriuuid.UUID, suppressorID satoriuuid.UUID) error {
	var lnk = WorkItemLink{}
	tx := r.db.Where("id = ?", linkID).Find(&lnk)
	if tx.RecordNotFound() {
		return errors.NewNotFoundError("work item link", linkID.String())
	}
	r.deleteLink(ctx, lnk, suppressorID)
	return nil
}

// DeleteRelatedLinks deletes all links in which the source or target equals the
// given work item ID.
func (r *GormWorkItemLinkRepository) DeleteRelatedLinks(ctx context.Context, wiIDStr string, suppressorID satoriuuid.UUID) error {
	log.Info(ctx, map[string]interface{}{
		"workitem_id": wiIDStr,
	}, "Deleting the links related to work item")

	wiID, err := strconv.ParseUint(wiIDStr, 10, 64)
	if err != nil {
		// treat as not found: clients don't know it must be a uint64
		return errors.NewNotFoundError("work item link", wiIDStr)
	}
	var workitemLinks = []WorkItemLink{}
	r.db.Where("? in (source_id, target_id)", wiID).Find(&workitemLinks)
	// delete one by one to trigger the creation of a new work item link revision
	for _, workitemLink := range workitemLinks {
		r.deleteLink(ctx, workitemLink, suppressorID)
	}
	return nil
}

// Delete deletes the work item link with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkRepository) deleteLink(ctx context.Context, lnk WorkItemLink, suppressorID satoriuuid.UUID) error {
	log.Info(ctx, map[string]interface{}{
		"wilID": lnk.ID,
	}, "Deleting the work item link")

	tx := r.db.Delete(&lnk)
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link", lnk.ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilID": lnk.ID,
			"err":   tx.Error,
		}, "unable to delete work item link")
		return errors.NewInternalError(tx.Error.Error())
	}
	// save a revision of the deleted work item link
	if err := r.revisionRepo.Create(ctx, suppressorID, RevisionTypeDelete, lnk); err != nil {
		return errs.Wrapf(err, "error while deleting work item")
	}
	return nil
}

// Save updates the given work item link in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Save(ctx context.Context, lt app.WorkItemLinkSingle, modifierID satoriuuid.UUID) (*app.WorkItemLinkSingle, error) {
	res := WorkItemLink{}
	if lt.Data.ID == nil {
		return nil, errors.NewBadParameterError("work item link", nil)
	}
	ID := *lt.Data.ID
	db := r.db.Model(&res).Where("id=?", ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilID": ID,
		}, "work item link not found")
		return nil, errors.NewNotFoundError("work item link", ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilID": ID,
			"err":   db.Error,
		}, "unable to find work item link")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	if lt.Data.Attributes.Version == nil || res.Version != *lt.Data.Attributes.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	if err := ConvertLinkToModel(lt, &res); err != nil {
		return nil, errs.WithStack(err)
	}
	res.Version = res.Version + 1
	if err := r.ValidateCorrectSourceAndTargetType(ctx, res.SourceID, res.TargetID, res.LinkTypeID); err != nil {
		return nil, errs.WithStack(err)
	}
	db = r.db.Save(&res)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilID": res.ID,
			"err":   db.Error,
		}, "unable to save work item link")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// save a revision of the modified work item link
	if err := r.revisionRepo.Create(ctx, modifierID, RevisionTypeUpdate, res); err != nil {
		return nil, errs.Wrapf(err, "error while saving work item")
	}
	log.Info(ctx, map[string]interface{}{
		"wilID": res.ID,
	}, "Work item link updated")
	result := ConvertLinkFromModel(res)
	return &result, nil
}
