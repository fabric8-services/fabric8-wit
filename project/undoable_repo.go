package project

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

var _ Repository = &UndoableRepository{}

// NewUndoableRepository creates a new UndoableRepository
func NewUndoableRepository(wrapped *GormRepository, undoScript *gormsupport.DBScript) *UndoableRepository {
	return &UndoableRepository{wrapped, undoScript}
}

// An UndoableRepository is a wrapper that appends inverse operations to an undo
// script for every operation and then calls the wrapped repo
type UndoableRepository struct {
	wrapped *GormRepository
	undo    *gormsupport.DBScript
}

// Load implements application.ProjectRepository
func (r *UndoableRepository) Load(ctx context.Context, id satoriuuid.UUID) (*Project, error) {
	return r.wrapped.Load(ctx, id)
}

// List implements application.ProjectRepository
func (r *UndoableRepository) List(ctx context.Context, start *int, length *int) ([]Project, uint64, error) {
	return r.wrapped.List(ctx, start, length)
}

// Create implements application.ProjectRepository
func (r *UndoableRepository) Create(ctx context.Context, name string) (*Project, error) {
	res, err := r.wrapped.Create(ctx, name)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Unscoped().Delete(&Project{ID: res.ID})
			return db.Error
		})
	}
	return res, err
}

// Save implements application.ProjectRepository
func (r *UndoableRepository) Save(ctx context.Context, p Project) (*Project, error) {

	old := Project{}
	db := r.wrapped.db.First(&old, p.ID)
	if db.Error != nil {
		return nil, errors.NewNotFoundError("project", p.ID.String())
	}

	res, err := r.wrapped.Save(ctx, p)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Save(&old)
			return db.Error
		})
	}
	return res, err
}

// Delete implements application.WorkItemRepository
func (r *UndoableRepository) Delete(ctx context.Context, ID satoriuuid.UUID) error {
	old := Project{}
	db := r.wrapped.db.First(&old, ID)
	if db.Error != nil {
		return errors.NewNotFoundError("project", ID.String())
	}

	err := r.wrapped.Delete(ctx, ID)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			old.DeletedAt = nil
			db = db.Save(&old)
			return db.Error
		})
	}
	return err
}
