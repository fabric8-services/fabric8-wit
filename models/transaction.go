package models

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

// Transactional executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Transactional(db *gorm.DB, todo func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := todo(tx); err != nil {
		tx := tx.Rollback()
		if tx.Error != nil {
			log.Error(context.Background(), map[string]interface{}{
				"errRollback": errs.WithStack(tx.Error),
				"err":         errs.WithStack(err),
			}, "failed to rollback transaction: %+v", tx.Error)
		}
		return errs.WithStack(err)
	}
	tx = tx.Commit()
	return tx.Error
}
