package workitem

import (
	"strconv"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var _ WorkItemRepository = &UndoableWorkItemRepository{}

// NewUndoableWorkItemRepository creates a new undoable work item repo
func NewUndoableWorkItemRepository(wrapped *GormWorkItemRepository, undoScript *gormsupport.DBScript) *UndoableWorkItemRepository {
	return &UndoableWorkItemRepository{wrapped, undoScript}
}

// An UndoableWorkItemRepository is a wrapper that appends inverse operations to an undo
// script for every operation and then calls the wrapped repo
type UndoableWorkItemRepository struct {
	wrapped *GormWorkItemRepository
	undo    *gormsupport.DBScript
}

// Load implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Load(ctx context.Context, workitemID string) (*app.WorkItem, error) {
	return r.wrapped.Load(ctx, workitemID)
}

// Save implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Save(ctx context.Context, wi app.WorkItem, currentUser uuid.UUID) (*app.WorkItem, error) {
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	log.Info(ctx, map[string]interface{}{
		"id": id,
	}, "Loading work item")
	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("could not load %s, %s", wi.ID, db.Error.Error()))
	}

	res, err := r.wrapped.Save(ctx, wi, currentUser)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Save(&old)
			return db.Error
		})
	}
	return res, errs.WithStack(err)
}

// Reorder implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Reorder(ctx context.Context, direction string, targetID string, wi app.WorkItem) (*app.WorkItem, error) {
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", string(id))
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("could not load %s, %s", wi.ID, db.Error.Error()))
	}

	res, err := r.wrapped.Reorder(ctx, direction, targetID, wi)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Save(&old)
			return db.Error
		})
	}
	return res, err
}

// Delete implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Delete(ctx context.Context, workitemID string, currentUser uuid.UUID) error {
	id, err := strconv.ParseUint(workitemID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return errors.NewNotFoundError("work item", workitemID)
	}

	log.Info(ctx, map[string]interface{}{
		"id": id,
	}, "Loading work iteme")

	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("could not load %s, %s", workitemID, db.Error.Error()))
	}

	err = r.wrapped.Delete(ctx, workitemID, currentUser)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			old.DeletedAt = nil
			db = db.Save(&old)
			return db.Error
		})
	}
	return errs.WithStack(err)
}

// Create implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Create(ctx context.Context, typeID uuid.UUID, fields map[string]interface{}, creator uuid.UUID) (*app.WorkItem, error) {
	result, err := r.wrapped.Create(ctx, typeID, fields, creator)
	if err != nil {
		return result, errs.WithStack(err)
	}
	id, err := strconv.ParseUint(result.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", result.ID)
	}

	toDelete := WorkItem{ID: id}

	r.undo.Append(func(db *gorm.DB) error {
		db = db.Unscoped().Delete(&toDelete)
		return db.Error
	})

	return result, errs.WithStack(err)
}

// List implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error) {
	return r.wrapped.List(ctx, criteria, start, length)
}

// Fetch fetches the (first) work item matching by the given criteria.Expression.
func (r *UndoableWorkItemRepository) Fetch(ctx context.Context, criteria criteria.Expression) (*app.WorkItem, error) {
	return r.wrapped.Fetch(ctx, criteria)
}

func (r *UndoableWorkItemRepository) GetCountsPerIteration(ctx context.Context, spaceId uuid.UUID) (map[string]WICountsPerIteration, error) {
	return map[string]WICountsPerIteration{}, nil
}

func (r *UndoableWorkItemRepository) GetCountsForIteration(ctx context.Context, iterationId uuid.UUID) (map[string]WICountsPerIteration, error) {
	return map[string]WICountsPerIteration{}, nil
}
