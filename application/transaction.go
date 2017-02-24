package application

import (
	"github.com/almighty/almighty-core/log"

	"github.com/pkg/errors"
)

// Transactional executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Transactional(db DB, todo func(f Application) error) error {
	var tx Transaction
	var err error
	if tx, err = db.BeginTransaction(); err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "database BeginTransaction failed!")

		return errors.WithStack(err)
	}

	if err := todo(tx); err != nil {
		log.Debug(nil, map[string]interface{}{}, "Rolling back the transaction...")

		tx.Rollback()

		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "database transaction failed!")
		return errors.WithStack(err)
	}

	log.Debug(nil, map[string]interface{}{}, "Commit the transaction!")

	return tx.Commit()
}
