package application

import "github.com/pkg/errors"

// Transactional executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Transactional(db DB, todo func(f Application) error) error {
	var tx Transaction
	var err error
	if tx, err = db.BeginTransaction(); err != nil {
		return errors.WithStack(err)
	}
	if err := todo(tx); err != nil {
		tx.Rollback()
		return errors.WithStack(err)
	}
	return tx.Commit()
}
