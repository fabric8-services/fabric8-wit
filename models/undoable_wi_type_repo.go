package models

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/jinzhu/gorm"
)

var _ application.WorkItemTypeRepository = &UndoableWorkItemTypeRepository{}

func NewUndoableWorkItemTypeRepository(wrapped *GormWorkItemTypeRepository, undoScript *DBScript) *UndoableWorkItemTypeRepository {
	return &UndoableWorkItemTypeRepository{wrapped, undoScript}
}

type UndoableWorkItemTypeRepository struct {
	wrapped *GormWorkItemTypeRepository
	undo    *DBScript
}

func (r *UndoableWorkItemTypeRepository) Load(ctx context.Context, name string) (*app.WorkItemType, error) {
	return r.wrapped.Load(ctx, name)
}
func (r *UndoableWorkItemTypeRepository) List(ctx context.Context, start *int, length *int) ([]*app.WorkItemType, error) {
	return r.wrapped.List(ctx, start, length)
}
func (r *UndoableWorkItemTypeRepository) Create(ctx context.Context, extendedTypeID *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error) {
	res, err := r.wrapped.Create(ctx, extendedTypeID, name, fields)
	if err == nil {
		r.undo.Append(func(db *gorm.DB) error {
			db = db.Unscoped().Delete(WorkItemType{Name: name})
			return db.Error
		})
	}
	return res, err
}
