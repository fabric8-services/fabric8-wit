package application

import (
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
)

//An Application stands for a particular implementation of the business logic of our application
type Application interface {
	WorkItems() workitem.WorkItemRepository
	WorkItemTypes() workitem.WorkItemTypeRepository
	Trackers() remoteworkitem.TrackerRepository
	TrackerQueries() remoteworkitem.TrackerQueryRepository
	SearchItems() SearchRepository
	Identities() account.IdentityRepository
	WorkItemLinkCategories() link.WorkItemLinkCategoryRepository
	WorkItemLinkTypes() link.WorkItemLinkTypeRepository
	WorkItemLinks() link.WorkItemLinkRepository
	Comments() comment.Repository
	Spaces() space.Repository
	Iterations() iteration.Repository
	Users() account.UserRepository
	Areas() area.Repository
	Codebases() codebase.Repository
	Labels() label.Repository
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
