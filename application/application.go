package application

import "github.com/almighty/almighty-core/comment"

//An Application stands for a particular implementation of the business logic of our application
type Application interface {
	WorkItems() WorkItemRepository
	WorkItems2() WorkItem2Repository
	WorkItemTypes() WorkItemTypeRepository
	Trackers() TrackerRepository
	TrackerQueries() TrackerQueryRepository
	SearchItems() SearchRepository
	Identities() IdentityRepository
	WorkItemLinkCategories() WorkItemLinkCategoryRepository
	WorkItemLinkTypes() WorkItemLinkTypeRepository
	WorkItemLinks() WorkItemLinkRepository
	WorkItemComments() comment.Repository
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
