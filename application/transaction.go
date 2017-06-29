package application

import (
	"github.com/fabric8-services/fabric8-wit/log"

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

	defer func() {
		switch err {
		case nil:
			log.Debug(nil, map[string]interface{}{}, "Commit the transaction!")
			err = tx.Commit()
		default:
			log.Debug(nil, map[string]interface{}{}, "Rolling back the transaction...")
			_ = tx.Rollback()
			log.Error(nil, map[string]interface{}{
				"err": err,
			}, "database transaction failed!")
			err = errors.WithStack(err)
		}
	}()

	err = todo(tx)
	return err
}
