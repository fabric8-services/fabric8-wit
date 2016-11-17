package models

import (
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/criteria"
	"github.com/jinzhu/gorm"
)

var _ application.WorkItemRepository = &UndoableWorkItemRepository{}

type DBScript struct {
	script []func(db *gorm.DB) error
}

func (s *DBScript) Run(db *gorm.DB) []error {
	var errs []error
	for _, f := range s.script {
		err := f(db)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (s *DBScript) Append(f func(db *gorm.DB) error) {
	s.script = append(s.script, f)
}

func NewUndoableWorkItemRepository(wrapped *GormWorkItemRepository, undoScript *DBScript) *UndoableWorkItemRepository {
	return &UndoableWorkItemRepository{wrapped, undoScript}
}

type UndoableWorkItemRepository struct {
	wrapped *GormWorkItemRepository
	undo    *DBScript
}

func (r *UndoableWorkItemRepository) Load(ctx context.Context, ID string) (*app.WorkItem, error) {
	return r.wrapped.Load(ctx, ID)
}

func (r *UndoableWorkItemRepository) Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error) {
	id, err := strconv.ParseUint(wi.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"work item", wi.ID}
	}

	log.Printf("loading work item %d", id)
	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", old)
		return nil, NotFoundError{"work item", wi.ID}
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

func (r *UndoableWorkItemRepository) Delete(ctx context.Context, ID string) error {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return NotFoundError{"work item", ID}
	}

	log.Printf("loading work item %d", id)
	old := WorkItem{}
	db := r.wrapped.db.First(&old, id)
	if db.RecordNotFound() {
		log.Printf("not found, res=%v", old)
		return NotFoundError{"work item", ID}
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

func (r *UndoableWorkItemRepository) Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error) {
	result, err := r.wrapped.Create(ctx, typeID, fields, creator)
	if err != nil {
		return result, err
	}
	id, err := strconv.ParseUint(result.ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"work item", result.ID}
	}

	toDelete := WorkItem{ID: id}

	r.undo.Append(func(db *gorm.DB) error {
		db = db.Unscoped().Delete(&toDelete)
		return db.Error
	})

	return result, err
}

func (r *UndoableWorkItemRepository) List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error) {
	return r.wrapped.List(ctx, criteria, start, length)
}

func (r *UndoableWorkItemRepository) Undo(db *gorm.DB) []error {
	return r.undo.Run(db)
}
