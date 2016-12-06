package test

import (
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/link"
)

func NewMockDB() *MockDB {
	return &MockDB{wir: &WorkItemRepository{}}
}

type MockDB struct {
	wir *WorkItemRepository
}

func (db *MockDB) WorkItems() application.WorkItemRepository {
	return db.wir
}
func (db *MockDB) WorkItems2() application.WorkItem2Repository {
	return nil
}
func (db *MockDB) WorkItemTypes() application.WorkItemTypeRepository {
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
