package gormapplication

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/project"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/search"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/jinzhu/gorm"
)

// A TXIsoLevel specifies the characteristics of the transaction
// See https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
type TXIsoLevel int8

const (
	// TXIsoLevelDefault doesn't specify any transaction isolation level, instead the connection
	// based setting will be used.
	TXIsoLevelDefault TXIsoLevel = iota

	// TXIsoLevelReadCommitted means "A statement can only see rows committed before it began. This is the default."
	TXIsoLevelReadCommitted

	// TXIsoLevelRepeatableRead means "All statements of the current transaction can only see rows committed before the
	// first query or data-modification statement was executed in this transaction."
	TXIsoLevelRepeatableRead

	// TXIsoLevelSerializable means "All statements of the current transaction can only see rows committed
	// before the first query or data-modification statement was executed in this transaction.
	// If a pattern of reads and writes among concurrent serializable transactions would create a
	// situation which could not have occurred for any serial (one-at-a-time) execution of those
	// transactions, one of them will be rolled back with a serialization_failure error."
	TXIsoLevelSerializable
)

var x application.Application = &GormDB{}

var y application.Application = &GormTransaction{}

func NewGormDB(db *gorm.DB) *GormDB {
	return &GormDB{GormBase{db}, ""}
}

// GormBase is a base struct for gorm implementations of db & transaction
type GormBase struct {
	db *gorm.DB
}

type GormTransaction struct {
	GormBase
}

type GormDB struct {
	GormBase
	txIsoLevel string
}

func (g *GormBase) WorkItems() workitem.WorkItemRepository {
	return workitem.NewWorkItemRepository(g.db)
}

func (g *GormBase) WorkItemTypes() workitem.WorkItemTypeRepository {
	return workitem.NewWorkItemTypeRepository(g.db)
}

func (g *GormBase) Projects() project.Repository {
	return project.NewRepository(g.db)
}

func (g *GormBase) Trackers() application.TrackerRepository {
	return remoteworkitem.NewTrackerRepository(g.db)
}
func (g *GormBase) TrackerQueries() application.TrackerQueryRepository {
	return remoteworkitem.NewTrackerQueryRepository(g.db)
}

func (g *GormBase) SearchItems() application.SearchRepository {
	return search.NewGormSearchRepository(g.db)
}

func (g *GormBase) Identities() application.IdentityRepository {
	return account.NewIdentityRepository(g.db)
}

// WorkItemLinkCategories returns a work item link category repository
func (g *GormBase) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return link.NewWorkItemLinkCategoryRepository(g.db)
}

// WorkItemLinkTypes returns a work item link type repository
func (g *GormBase) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return link.NewWorkItemLinkTypeRepository(g.db)
}

// WorkItemLinks returns a work item link repository
func (g *GormBase) WorkItemLinks() link.WorkItemLinkRepository {
	return link.NewWorkItemLinkRepository(g.db)
}

// WorkItemComments returns a work item comments repository
func (g *GormBase) WorkItemComments() comment.Repository {
	return comment.NewCommentRepository(g.db)
}

func (g *GormBase) DB() *gorm.DB {
	return g.db
}

// SetTransactionIsolationLevel sets the isolation level for
// See also https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
func (g *GormDB) SetTransactionIsolationLevel(level TXIsoLevel) error {
	switch level {
	case TXIsoLevelReadCommitted:
		g.txIsoLevel = "READ COMMITTED"
	case TXIsoLevelRepeatableRead:
		g.txIsoLevel = "REPEATABLE READ"
	case TXIsoLevelSerializable:
		g.txIsoLevel = "SERIALIZABLE"
	case TXIsoLevelDefault:
		g.txIsoLevel = ""
	default:
		return fmt.Errorf("Unknown transaction isolation level: " + strconv.FormatInt(int64(level), 10))
	}
	return nil
}

// Begin implements TransactionSupport
func (g *GormDB) BeginTransaction() (application.Transaction, error) {
	tx := g.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if len(g.txIsoLevel) != 0 {
		tx := tx.Exec(fmt.Sprintf("set transaction isolation level %s", g.txIsoLevel))
		if tx.Error != nil {
			return nil, tx.Error
		}
		return &GormTransaction{GormBase{tx}}, nil
	}
	return &GormTransaction{GormBase{tx}}, nil
}

// Commit implements TransactionSupport
func (g *GormTransaction) Commit() error {
	err := g.db.Commit().Error
	g.db = nil
	return err
}

// Rollback implements TransactionSupport
func (g *GormTransaction) Rollback() error {
	err := g.db.Rollback().Error
	g.db = nil
	return err
}
