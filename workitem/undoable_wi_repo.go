package workitem

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/jinzhu/gorm"
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
func (r *UndoableWorkItemRepository) Load(ctx context.Context, ID string) (*app.WorkItem, error) {
	return r.wrapped.Load(ctx, ID)
}

// Save implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error) {
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("work item", wi.ID)
	}

	log.Printf("loading work item %d", id)
	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.Error != nil {
		return nil, errors.NewInternalError(fmt.Sprintf("could not load %s, %s", wi.ID, db.Error.Error()))
	}

	res, err := r.wrapped.Save(ctx, wi)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Save(&old)
			return db.Error
		})
	}
	return res, err
}

// Delete implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Delete(ctx context.Context, ID string) error {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return errors.NewNotFoundError("work item", ID)
	}

	log.Printf("loading work item %d", id)
	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.Error != nil {
		return errors.NewInternalError(fmt.Sprintf("could not load %s, %s", ID, db.Error.Error()))
	}

	err = r.wrapped.Delete(ctx, ID)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			old.DeletedAt = nil
			db = db.Save(&old)
			return db.Error
		})
	}
	return err
}

// Create implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error) {
	result, err := r.wrapped.Create(ctx, typeID, fields, creator)
	if err != nil {
		return result, err
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

	return result, err
}

// List implements application.WorkItemRepository
func (r *UndoableWorkItemRepository) List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error) {
	return r.wrapped.List(ctx, criteria, start, length)
}
