package workitem

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var _ WorkItemTypeRepository = &UndoableWorkItemTypeRepository{}

// NewUndoableWorkItemTypeRepository creates a new UndoableWorkItemTypeRepository
func NewUndoableWorkItemTypeRepository(wrapped *GormWorkItemTypeRepository, undoScript *gormsupport.DBScript) *UndoableWorkItemTypeRepository {
	return &UndoableWorkItemTypeRepository{wrapped, undoScript}
}

// An UndoableWorkItemTypeRepository is a wrapper that appends inverse operations to an undo
// script for every operation and then calls the wrapped repo
type UndoableWorkItemTypeRepository struct {
	wrapped *GormWorkItemTypeRepository
	undo    *gormsupport.DBScript
}

// Load implements application.WorkItemTypeRepository
func (r *UndoableWorkItemTypeRepository) Load(ctx context.Context, id uuid.UUID) (*app.WorkItemTypeSingle, error) {
	return r.wrapped.Load(ctx, id)
}

// List implements application.WorkItemTypeRepository
func (r *UndoableWorkItemTypeRepository) List(ctx context.Context, start *int, length *int) (*app.WorkItemTypeList, error) {
	return r.wrapped.List(ctx, start, length)
}

// Create implements application.WorkItemTypeRepository
func (r *UndoableWorkItemTypeRepository) Create(ctx context.Context, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, fields map[string]app.FieldDefinition) (*app.WorkItemTypeSingle, error) {
	res, err := r.wrapped.Create(ctx, id, extendedTypeID, name, description, fields)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Unscoped().Delete(&WorkItemType{ID: *res.Data.ID})
			return db.Error
		})
	}
	return res, errors.WithStack(err)
}
