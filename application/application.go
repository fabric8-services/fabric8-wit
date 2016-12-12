package application

import (
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/project"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
)

//An Application stands for a particular implementation of the business logic of our application
type Application interface {
	WorkItems() workitem.WorkItemRepository
	WorkItemTypes() workitem.WorkItemTypeRepository
	Trackers() TrackerRepository
	TrackerQueries() TrackerQueryRepository
	SearchItems() SearchRepository
	Identities() IdentityRepository
	WorkItemLinkCategories() link.WorkItemLinkCategoryRepository
	WorkItemLinkTypes() link.WorkItemLinkTypeRepository
	WorkItemLinks() link.WorkItemLinkRepository
	Comments() comment.Repository
	Projects() project.Repository
	Users() account.IdentityRepository
}

// A Transaction abstracts a database transaction. The repositories created for the transaction object make changes inside the the transaction
type Transaction interface {
	Application
	Commit() error
	Rollback() error
}

// A DB stands for a particular database (or a mock/fake thereof). It also includes "Application" for creating transactionless repositories
type DB interface {
	Application
	BeginTransaction() (Transaction, error)
}
