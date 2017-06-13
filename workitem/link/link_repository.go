package link

import (
	"fmt"
	"strconv"
	"time"

	"context"

	"github.com/almighty/almighty-core/application/repository"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// End points
const (
	EndpointWorkItemTypes          = "workitemtypes"
	EndpointWorkItems              = "workitems"
	EndpointWorkItemLinkCategories = "workitemlinkcategories"
	EndpointWorkItemLinkTypes      = "workitemlinktypes"
	EndpointWorkItemLinks          = "workitemlinks"
)

// WorkItemLinkRepository encapsulates storage & retrieval of work item links
type WorkItemLinkRepository interface {
	repository.Exister
	Create(ctx context.Context, sourceID, targetID uint64, linkTypeID uuid.UUID, creatorID uuid.UUID) (*WorkItemLink, error)
	Load(ctx context.Context, ID uuid.UUID) (*WorkItemLink, error)
	List(ctx context.Context) ([]WorkItemLink, error)
	ListByWorkItemID(ctx context.Context, wiIDStr string) ([]WorkItemLink, error)
	DeleteRelatedLinks(ctx context.Context, wiIDStr string, suppressorID uuid.UUID) error
	Delete(ctx context.Context, ID uuid.UUID, suppressorID uuid.UUID) error
	Save(ctx context.Context, linkCat WorkItemLink, modifierID uuid.UUID) (*WorkItemLink, error)
	ListWorkItemChildren(ctx context.Context, parent string) ([]workitem.WorkItem, error)
	WorkItemHasChildren(ctx context.Context, parent string) (bool, error)
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
func (r *GormWorkItemLinkRepository) ValidateCorrectSourceAndTargetType(ctx context.Context, sourceID, targetID uint64, linkTypeID uuid.UUID) error {
	linkType, err := r.workItemLinkTypeRepo.Load(ctx, linkTypeID)
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

// CheckParentExists returns error if there is an attempt to create more than 1 parent of a workitem.
func (r *GormWorkItemLinkRepository) CheckParentExists(ctx context.Context, targetID uint64, linkType *WorkItemLinkType) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				link_type_id=$1
				AND target_id=$2
				AND deleted_at IS NULL
		)`, WorkItemLink{}.TableName())
	row := r.db.CommonDB().QueryRow(query, linkType.ID, targetID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, errs.Wrapf(err, "failed to check if a parent exists for the work item %d", targetID)
	}
	return exists, nil
}

func (r *GormWorkItemLinkRepository) ValidateTopology(ctx context.Context, targetID uint64, linkType *WorkItemLinkType) error {
	// check to disallow multiple parents in tree topology
	if linkType.Topology == TopologyTree {
		parentExists, err := r.CheckParentExists(ctx, targetID, linkType)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"wilt_id":   linkType.ID,
				"target_id": targetID,
				"err":       err,
			}, "failed to check if the work item %s has a parent work item", targetID)
			return errs.Wrapf(err, "failed to check if the work item %s has a parent work item", targetID)
		}
		if parentExists {
			log.Error(ctx, map[string]interface{}{
				"wilt_id":   linkType.ID,
				"target_id": targetID,
				"err":       err,
			}, "unable to create work item link because a topology of type \"%s\" only allows one parent to exist and the target %d already a parent", TopologyTree, targetID)
			return errors.NewBadParameterError("linkTypeID + targetID", fmt.Sprintf("%s + %d", linkType.ID, targetID)).Expected("single parent in tree topology")
		}
	}
	return nil
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, sourceID, targetID uint64, linkTypeID uuid.UUID, creatorID uuid.UUID) (*WorkItemLink, error) {
	link := &WorkItemLink{
		SourceID:   sourceID,
		TargetID:   targetID,
		LinkTypeID: linkTypeID,
	}
	if err := link.CheckValidForCreation(); err != nil {
		return nil, errs.WithStack(err)
	}

	// Fetch the link type
	linkType, err := r.workItemLinkTypeRepo.Load(ctx, linkTypeID)
	if err != nil {
		return nil, errs.Wrap(err, "failed to load link type")
	}

	if err := r.ValidateCorrectSourceAndTargetType(ctx, sourceID, targetID, linkType.ID); err != nil {
		return nil, errs.WithStack(err)
	}

	if err := r.ValidateTopology(ctx, targetID, linkType); err != nil {
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
	return link, nil
}

// Load returns the work item link for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Load(ctx context.Context, ID uuid.UUID) (*WorkItemLink, error) {
	log.Info(ctx, map[string]interface{}{
		"wil_id": ID,
	}, "Loading work item link")
	result := WorkItemLink{}
	db := r.db.Where("id=?", ID).Find(&result)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wil_id": ID,
		}, "work item link not found")
		return nil, errors.NewNotFoundError("work item link", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &result, nil
}

func (m *GormWorkItemLinkRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				id=$1
				AND deleted_at IS NULL
		)`, WorkItemLink{}.TableName())
	var exists bool
	if err := m.db.CommonDB().QueryRow(query, id).Scan(&exists); err != nil {
		return false, errs.Wrapf(err, "failed to check if a work item link with this id %v", id)
	}
	return exists, nil
}

// ListByWorkItemID returns the work item links that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) ListByWorkItemID(ctx context.Context, wiIDStr string) ([]WorkItemLink, error) {
	var modelLinks []WorkItemLink
	wi, err := r.workItemRepo.LoadFromDB(ctx, wiIDStr)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	// Now fetch all links for that work item
	db := r.db.Model(modelLinks).Where("? IN (source_id, target_id)", wi.ID).Find(&modelLinks)
	if db.Error != nil {
		return nil, db.Error
	}
	return modelLinks, nil
}

// List returns all work item links if wiID is nil; otherwise the work item links are returned
// that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) List(ctx context.Context) ([]WorkItemLink, error) {
	var modelLinks []WorkItemLink
	db := r.db.Find(&modelLinks)
	if db.Error != nil {
		return nil, db.Error
	}
	return modelLinks, nil
}

// Delete deletes the work item link with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkRepository) Delete(ctx context.Context, linkID uuid.UUID, suppressorID uuid.UUID) error {
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
func (r *GormWorkItemLinkRepository) DeleteRelatedLinks(ctx context.Context, wiIDStr string, suppressorID uuid.UUID) error {
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
func (r *GormWorkItemLinkRepository) deleteLink(ctx context.Context, lnk WorkItemLink, suppressorID uuid.UUID) error {
	log.Info(ctx, map[string]interface{}{
		"wil_id": lnk.ID,
	}, "Deleting the work item link")

	tx := r.db.Delete(&lnk)
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link", lnk.ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wil_id": lnk.ID,
			"err":    tx.Error,
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
func (r *GormWorkItemLinkRepository) Save(ctx context.Context, linkToSave WorkItemLink, modifierID uuid.UUID) (*WorkItemLink, error) {
	log.Info(ctx, map[string]interface{}{
		"wil_id": linkToSave.LinkTypeID,
	}, "Saving workitem link with type =  %s", linkToSave.LinkTypeID)
	existingLink := WorkItemLink{}
	db := r.db.Model(&existingLink).Where("id=?", linkToSave.ID).First(&existingLink)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wil_id": linkToSave.ID,
		}, "work item link not found")
		return nil, errors.NewNotFoundError("work item link", linkToSave.ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wil_id": linkToSave.ID,
			"err":    db.Error,
		}, "unable to find work item link")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	if existingLink.Version != linkToSave.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	linkToSave.Version = linkToSave.Version + 1

	linkTypeToSave, err := r.workItemLinkTypeRepo.Load(ctx, linkToSave.LinkTypeID)
	if err != nil {
		return nil, errs.Wrap(err, "failed to load link type")
	}

	if err := r.ValidateCorrectSourceAndTargetType(ctx, linkToSave.SourceID, linkToSave.TargetID, linkTypeToSave.ID); err != nil {
		return nil, errs.WithStack(err)
	}

	if err := r.ValidateTopology(ctx, linkToSave.TargetID, linkTypeToSave); err != nil {
		return nil, errs.WithStack(err)
	}

	// save
	db = r.db.Save(&linkToSave)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wil_id": linkToSave.ID,
			"err":    db.Error,
		}, "unable to save work item link")
		return nil, errors.NewInternalError(db.Error.Error())
	}
	// save a revision of the modified work item link
	if err := r.revisionRepo.Create(ctx, modifierID, RevisionTypeUpdate, linkToSave); err != nil {
		return nil, errs.Wrapf(err, "error while saving work item")
	}
	log.Info(ctx, map[string]interface{}{
		"wil_id": linkToSave.ID,
	}, "Work item link updated")
	return &linkToSave, nil
}

// ListWorkItemChildren get all child work items
func (r *GormWorkItemLinkRepository) ListWorkItemChildren(ctx context.Context, parent string) ([]workitem.WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "children", "query"}, time.Now())

	where := fmt.Sprintf(`
	id in (
		SELECT target_id FROM %s
		WHERE source_id = ? AND link_type_id IN (
			SELECT id FROM %s WHERE forward_name = 'parent of'
		)
	)`, WorkItemLink{}.TableName(), WorkItemLinkType{}.TableName())
	db := r.db.Model(&workitem.WorkItemStorage{}).Where(where, parent)
	rows, err := db.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []workitem.WorkItemStorage{}

	for rows.Next() {
		value := workitem.WorkItemStorage{}
		db.ScanRows(rows, &value)

		result = append(result, value)
	}
	res := make([]workitem.WorkItem, len(result))
	for index, value := range result {
		wiType, err := r.workItemTypeRepo.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		modelWI, err := workitem.ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			return nil, errors.NewInternalError(err.Error())
		}
		res[index] = *modelWI
	}

	return res, nil
}

// WorkItemHasChildren returns true if the given parent work item has children;
// otherwise false is returned
func (r *GormWorkItemLinkRepository) WorkItemHasChildren(ctx context.Context, parent string) (bool, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "has", "children"}, time.Now())
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s WHERE id in (
				SELECT target_id FROM %[2]s
				WHERE source_id = $1 AND link_type_id IN (
					SELECT id FROM %[3]s WHERE forward_name = 'parent of'
				)
			)
		)`,
		workitem.WorkItemStorage{}.TableName(),
		WorkItemLink{}.TableName(),
		WorkItemLinkType{}.TableName())
	var hasChildren bool
	db := r.db.CommonDB()
	stmt, err := db.Prepare(query)
	if err != nil {
		return false, errs.Wrapf(err, "failed prepare statement: %s", query)
	}
	defer stmt.Close()
	err = stmt.QueryRow(parent).Scan(&hasChildren)
	if err != nil {
		return false, errs.Wrapf(err, "failed to check if work item %s has children: %s", parent, query)
	}
	return hasChildren, nil
}
