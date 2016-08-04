package models

import (
	"github.com/jinzhu/gorm"
)

// TransactionSupport abstracts from concrete transaction implementation
type TransactionSupport interface {
	// Begin starts a transaction
	Begin() error
	// Commit commits a transaction
	Commit() error
	// Rollback rolls back the transaction
	Rollback() error
}

// NewGormTransactionSupport constructs a new instance of GormTransactionSupport
func NewGormTransactionSupport(db *gorm.DB) *GormTransactionSupport {
	return &GormTransactionSupport{db: db}
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
