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
		log.LoggerRuntimeContext().WithFields(map[string]interface{}{
			"err": err,
		}).Errorln("Database BeginTransaction failed!")

		return errors.WithStack(err)
	}

	if err := todo(tx); err != nil {
		log.Logger().WithFields(map[string]interface{}{
			"pkg": "application",
		}).Debugln("Rolling back the transaction...")

		tx.Rollback()

		log.LoggerRuntimeContext().WithFields(map[string]interface{}{
			"err": err,
		}).Errorln("Database transaction failed!")
		return errors.WithStack(err)
	}

	log.Logger().WithFields(map[string]interface{}{
		"pkg": "application",
	}).Debugln("Commit the transaction!")

	return tx.Commit()
}
