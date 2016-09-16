package remoteworkitem

import (
	"github.com/jinzhu/gorm"
)

// NewGormTransactionSupport constructs a new instance of GormTransactionSupport
func NewGormTransactionSupport(db *gorm.DB) *GormTransactionSupport {
	return &GormTransactionSupport{db: db, tx: nil}
}

// GormTransactionSupport implements TransactionSupport for gorm
type GormTransactionSupport struct {
	tx *gorm.DB
	db *gorm.DB
}

// Begin implements TransactionSupport
func (g *GormTransactionSupport) Begin() error {
	g.tx = g.db.Begin()
	return g.db.Error
}

// Commit implements TransactionSupport
func (g *GormTransactionSupport) Commit() error {
	err := g.tx.Commit().Error
	g.tx = nil
	return err
}

// Rollback implements TransactionSupport
func (g *GormTransactionSupport) Rollback() error {
	err := g.tx.Rollback().Error
	g.tx = nil
	return err
}
