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
		log.LogError(nil, map[string]interface{}{
			"err": err,
		}, "Database BeginTransaction failed!")

		return errors.WithStack(err)
	}

	if err := todo(tx); err != nil {
		log.LogDebug(nil, map[string]interface{}{
			"pkg": "application",
		}, "Rolling back the transaction...")

		tx.Rollback()

		log.LogError(nil, map[string]interface{}{
			"err": err,
		}, "Database transaction failed!")
		return errors.WithStack(err)
	}

	log.LogDebug(nil, map[string]interface{}{
		"pkg": "application",
	}, "Commit the transaction!")

	return tx.Commit()
}
