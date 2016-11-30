package models

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/project"
	"github.com/jinzhu/gorm"
	satoriuuid "github.com/satori/go.uuid"
)

var _ application.ProjectRepository = &UndoableProjectRepository{}

// NewUndoableProjectRepository creates a new UndoableProjectRepository
func NewUndoableProjectRepository(wrapped *GormProjectRepository, undoScript *gormsupport.DBScript) *UndoableProjectRepository {
	return &UndoableProjectRepository{wrapped, undoScript}
}

// An UndoableProjectRepository is a wrapper that appends inverse operations to an undo
// script for every operation and then calls the wrapped repo
type UndoableProjectRepository struct {
	wrapped *GormProjectRepository
	undo    *gormsupport.DBScript
}

// Load implements application.ProjectRepository
func (r *UndoableProjectRepository) Load(ctx context.Context, id satoriuuid.UUID) (*project.Project, error) {
	return r.wrapped.Load(ctx, id)
}

// List implements application.ProjectRepository
func (r *UndoableProjectRepository) List(ctx context.Context, start *int, length *int) ([]project.Project, uint64, error) {
	return r.wrapped.List(ctx, start, length)
}

// Create implements application.ProjectRepository
func (r *UndoableProjectRepository) Create(ctx context.Context, name string) (*project.Project, error) {
	res, err := r.wrapped.Create(ctx, name)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Unscoped().Delete(&project.Project{ID: res.ID})
			return db.Error
		})
	}
	return res, err
}

// Save implements application.ProjectRepository
func (r *UndoableProjectRepository) Save(ctx context.Context, p project.Project) (*project.Project, error) {

	old := project.Project{}
	db := r.wrapped.db.First(&old, p.ID)
	if db.Error != nil {
		return nil, NewNotFoundError("project", p.ID.String())
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
func (r *UndoableProjectRepository) Delete(ctx context.Context, ID satoriuuid.UUID) error {
	old := project.Project{}
	db := r.wrapped.db.First(&old, ID)
	if db.Error != nil {
		return NewInternalError(fmt.Sprintf("could not load %s, %s", ID, db.Error.Error()))
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
