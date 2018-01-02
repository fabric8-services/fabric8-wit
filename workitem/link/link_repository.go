package link

import (
	"database/sql"
	"fmt"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"

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
	Create(ctx context.Context, sourceID, targetID uuid.UUID, linkTypeID uuid.UUID, creatorID uuid.UUID) (*WorkItemLink, error)
	Load(ctx context.Context, ID uuid.UUID) (*WorkItemLink, error)
	List(ctx context.Context) ([]WorkItemLink, error)
	ListByWorkItem(ctx context.Context, wiID uuid.UUID) ([]WorkItemLink, error)
	DeleteRelatedLinks(ctx context.Context, wiID uuid.UUID, suppressorID uuid.UUID) error
	Delete(ctx context.Context, ID uuid.UUID, suppressorID uuid.UUID) error
	ListWorkItemChildren(ctx context.Context, parentID uuid.UUID, start *int, limit *int) ([]workitem.WorkItem, uint64, error)
	WorkItemHasChildren(ctx context.Context, parentID uuid.UUID) (bool, error)
	GetParentID(ctx context.Context, ID uuid.UUID) (*uuid.UUID, error) // GetParentID returns parent ID of the given work item if any
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

// CheckParentExists returns `true` if a link to a work item with the given `targetID` and of the given `linkType` already exists, `false` otherwise.
// If the `sourceId` argument is not nil, then existing link from the source item to the target item with the given type is ignored.
// In the context of a link creation, the `sourceID` argument should be nil, so the method will look for a link of the given type and target,
// since during link creation we need to ensure that the child item has no parent yet. During the link update, the verification should not take into account
// the existing record in the database.
func (r *GormWorkItemLinkRepository) CheckParentExists(ctx context.Context, sourceID *uuid.UUID, targetID uuid.UUID, linkType WorkItemLinkType) (bool, error) {
	var row *sql.Row
	if sourceID != nil {
		query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				link_type_id=$1
				AND source_id!=$2
				AND target_id=$3
				AND deleted_at IS NULL
		)`, WorkItemLink{}.TableName())
		row = r.db.CommonDB().QueryRow(query, linkType.ID, *sourceID, targetID)
	} else {
		query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				link_type_id=$1
				AND target_id=$2
				AND deleted_at IS NULL
		)`, WorkItemLink{}.TableName())
		row = r.db.CommonDB().QueryRow(query, linkType.ID, targetID)
	}
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, errs.Wrapf(err, "failed to check if a parent exists for the work item %d", targetID)
	}
	return exists, nil
}

// ValidateTopology validates the link topology of the work item given its ID. I.e, the given item should not have a parent with the same kind of link
// if the `sourceID` arg is not empty, then the corresponding source item is ignored when checking the existing links of the given type.
func (r *GormWorkItemLinkRepository) ValidateTopology(ctx context.Context, sourceID *uuid.UUID, targetID uuid.UUID, linkType WorkItemLinkType) error {
	// check to disallow multiple parents in tree topology
	if linkType.Topology == TopologyTree {
		parentExists, err := r.CheckParentExists(ctx, sourceID, targetID, linkType)
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
			}, "unable to create/update work item link because a topology of type \"%s\" only allows one parent to exist and the target %d already has a parent", TopologyTree, targetID)
			return errors.NewBadParameterError("linkTypeID + targetID", fmt.Sprintf("%s + %d", linkType.ID, targetID)).Expected("single parent in tree topology")
		}
	}
	return nil
}

// Create creates a new work item link in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkRepository) Create(ctx context.Context, sourceID, targetID uuid.UUID, linkTypeID uuid.UUID, creatorID uuid.UUID) (*WorkItemLink, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "create"}, time.Now())
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

	if err := r.ValidateTopology(ctx, nil, targetID, *linkType); err != nil {
		return nil, errs.WithStack(err)
	}

	db := r.db.Create(link)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "work_item_links_unique_idx") {
			log.Error(ctx, map[string]interface{}{
				"err":       db.Error,
				"source_id": sourceID,
			}, "unable to create work item link because a link already exists with the same source_id, target_id and type_id")
			return nil, errors.NewDataConflictError(fmt.Sprintf("work item link already exists with data.relationships.source_id: %s; data.relationships.target_id: %s; data.relationships.link_type_id: %s ", sourceID, targetID, linkTypeID))
		}
		if gormsupport.IsForeignKeyViolation(db.Error, "work_item_links_source_id_fkey") {
			return nil, errors.NewNotFoundError("source", sourceID.String())
		}
		if gormsupport.IsForeignKeyViolation(db.Error, "work_item_links_target_id_fkey") {
			return nil, errors.NewNotFoundError("target", targetID.String())
		}
		return nil, errors.NewInternalError(ctx, db.Error)
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
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "load"}, time.Now())
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
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	return &result, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemLinkRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "exists"}, time.Now())
	return repository.CheckExists(ctx, r.db, WorkItemLink{}.TableName(), id)
}

// ListByWorkItem returns the work item links that have wiID as source or target.
// TODO: Handle pagination
func (r *GormWorkItemLinkRepository) ListByWorkItem(ctx context.Context, wiID uuid.UUID) ([]WorkItemLink, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "listByWorkItem"}, time.Now())
	var modelLinks []WorkItemLink
	wi, err := r.workItemRepo.LoadFromDB(ctx, wiID)
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
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "list"}, time.Now())
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
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "delete"}, time.Now())
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
func (r *GormWorkItemLinkRepository) DeleteRelatedLinks(ctx context.Context, wiID uuid.UUID, suppressorID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "deleteRelatedLinks"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wi_id": wiID,
	}, "Deleting the links related to work item")

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
		return errors.NewInternalError(ctx, tx.Error)
	}
	// save a revision of the deleted work item link
	if err := r.revisionRepo.Create(ctx, suppressorID, RevisionTypeDelete, lnk); err != nil {
		return errs.Wrapf(err, "error while deleting work item")
	}
	return nil
}

// ListWorkItemChildren get all child work items
func (r *GormWorkItemLinkRepository) ListWorkItemChildren(ctx context.Context, parentID uuid.UUID, start *int, limit *int) ([]workitem.WorkItem, uint64, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "children", "query"}, time.Now())
	where := fmt.Sprintf(`
	id in (
		SELECT target_id FROM %s
		WHERE source_id = ? AND link_type_id IN (
			SELECT id FROM %s WHERE forward_name = 'parent of'
		)
	)`, WorkItemLink{}.TableName(), WorkItemLinkType{}.TableName())
	db := r.db.Model(&workitem.WorkItemStorage{}).Where(where, parentID.String())
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}
	db = db.Select("count(*) over () as cnt2 , *")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result := []workitem.WorkItemStorage{}

	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, errors.NewInternalError(ctx, err)
	}

	var count uint64
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := workitem.WorkItemStorage{}
		db.ScanRows(rows, &value)

		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(ctx, err)
			}
		}
		result = append(result, value)
	}

	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
		// need to do a count(*) to find out total
		db := db.Select("count(*)")
		rows2, err := db.Rows()
		defer rows2.Close()
		if err != nil {
			return nil, 0, errs.WithStack(err)
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}

	res := make([]workitem.WorkItem, len(result))
	for index, value := range result {
		wiType, err := r.workItemTypeRepo.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, errors.NewInternalError(ctx, err)
		}
		modelWI, err := workitem.ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			return nil, 0, errors.NewInternalError(ctx, err)
		}
		res[index] = *modelWI
	}

	return res, count, nil
}

// WorkItemHasChildren returns true if the given parent work item has children;
// otherwise false is returned
func (r *GormWorkItemLinkRepository) WorkItemHasChildren(ctx context.Context, parentID uuid.UUID) (bool, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "has", "children"}, time.Now())
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s WHERE id in (
				SELECT target_id FROM %[2]s
				WHERE source_id = $1 AND deleted_at IS NULL AND link_type_id IN (
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
	err = stmt.QueryRow(parentID.String()).Scan(&hasChildren)
	if err != nil {
		return false, errs.Wrapf(err, "failed to check if work item %s has children: %s", parentID.String(), query)
	}
	return hasChildren, nil
}

// GetParentID returns parent ID of the given work item if any
func (r *GormWorkItemLinkRepository) GetParentID(ctx context.Context, ID uuid.UUID) (*uuid.UUID, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlink", "get", "parent"}, time.Now())
	query := fmt.Sprintf(`
			SELECT id FROM %[1]s WHERE id in (
				SELECT source_id FROM %[2]s
				WHERE target_id = $1 AND deleted_at IS NULL AND link_type_id IN (
					SELECT id FROM %[3]s WHERE forward_name = 'parent of' and topology = '%[4]s'
				)
			)`,
		workitem.WorkItemStorage{}.TableName(),
		WorkItemLink{}.TableName(),
		WorkItemLinkType{}.TableName(),
		TopologyTree)
	var parentID uuid.UUID
	db := r.db.CommonDB()
	err := db.QueryRow(query, ID.String()).Scan(&parentID)
	if err != nil {
		return nil, errs.Wrapf(err, "parent not found for work item: %s", ID.String(), query)
	}
	return &parentID, nil
}
