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
	return &GormTransactionSupport{db}
}

// GormTransactionSupport implements TransactionSupport for gorm
type GormTransactionSupport struct {
	db *gorm.DB
}

// Begin implements TransactionSupport
func (g *GormTransactionSupport) Begin() error {
	g.db = g.db.Begin()
	return g.db.Error
}

// Commit implements TransactionSupport
func (g *GormTransactionSupport) Commit() error {
	g.db = g.db.Commit()
	return g.db.Error
}

// Rollback implements TransactionSupport
func (g *GormTransactionSupport) Rollback() error {
	g.db = g.db.Rollback()
	return g.db.Error
}
