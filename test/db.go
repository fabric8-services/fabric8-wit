package test

import (
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/project"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
)

func NewMockDB() *MockDB {
	return &MockDB{wir: &WorkItemRepository{}}
}

type MockDB struct {
	wir *WorkItemRepository
}

func (db *MockDB) WorkItems() workitem.WorkItemRepository {
	return db.wir
}
func (db *MockDB) WorkItemTypes() workitem.WorkItemTypeRepository {
	return nil
}

func (db *MockDB) Projects() project.Repository {
	return nil
}

func (db *MockDB) Trackers() application.TrackerRepository {
	return nil
}
func (db *MockDB) TrackerQueries() application.TrackerQueryRepository {
	return nil
}
func (db *MockDB) SearchItems() application.SearchRepository {
	return nil
}
func (db *MockDB) Identities() application.IdentityRepository {
	return nil
}
func (db *MockDB) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return nil
}
func (db *MockDB) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return nil
}
func (db *MockDB) WorkItemLinks() link.WorkItemLinkRepository {
	return nil
}
func (db *MockDB) WorkItemComments() comment.Repository {
	return nil
}

func (db *MockDB) Commit() error {
	return nil
}
func (db *MockDB) Rollback() error {
	return nil
}

func (db *MockDB) BeginTransaction() (application.Transaction, error) {
	return db, nil
}
