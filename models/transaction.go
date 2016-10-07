package models

import (
	"fmt"
	"strconv"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/transaction"
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

// NewGormTransactionSupport constructs a new instance of GormTransactionSupport
func NewGormTransactionSupport(db *gorm.DB) *GormTransactionSupport {
	return &GormTransactionSupport{db: db}
}

// GormTransactionSupport implements TransactionSupport for gorm
type GormTransactionSupport struct {
	tx         *gorm.DB
	db         *gorm.DB
	txIsoLevel string
}

// SetTransactionIsolationLevel sets the isolation level for
// See also https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
func (g *GormTransactionSupport) SetTransactionIsolationLevel(level TXIsoLevel) error {
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

// TX returns the transaction object
func (g *GormTransactionSupport) TX() *gorm.DB {
	return g.tx
}

// Begin implements TransactionSupport
func (g *GormTransactionSupport) Begin() error {
	g.tx = g.db.Begin()
	if g.db.Error != nil {
		return g.db.Error
	}
	if len(g.txIsoLevel) != 0 {
		db := g.tx.Exec(fmt.Sprintf("set transaction isolation level %s", g.txIsoLevel))
		return db.Error
	}
	return nil

}

// Commit implements TransactionSupport
func (g *GormTransactionSupport) Commit(tx transaction.Transaction) error {
	err := tx.(*gorm.DB).Commit().Error
	return err
}

// Rollback implements TransactionSupport
func (g *GormTransactionSupport) Rollback(tx transaction.Transaction) error {
	err := tx.(*gorm.DB).Rollback().Error
	return err
}

// CurrentTX returns the current gorm transaction or nil
func CurrentTX(ctx context.Context) *gorm.DB {
	return transaction.Current(ctx).(*gorm.DB)
}
