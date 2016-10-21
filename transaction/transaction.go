package transaction

import "golang.org/x/net/context"

// Transaction is an opaque handle to the transaction. Cast as necessary
type Transaction interface{}

var transactionKey struct{}

// Support abstracts from concrete transaction implementation
type Support interface {
	// Begin starts a transaction
	Begin() (Transaction, error)
	// Commit commits a transaction
	Commit(Transaction) error
	// Rollback rolls back the transaction
	Rollback(Transaction) error
}

// Do executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Do(support Support, ctx context.Context, todo func(context.Context) error) error {
	var tx Transaction
	var err error
	if tx, err = support.Begin(); err != nil {
		return err
	}

	if err := todo(context.WithValue(ctx, transactionKey, tx)); err != nil {
		support.Rollback(tx)
		return err
	}
	return support.Commit(tx)
}

// Current returns the current Transaction or nil if none is active
func Current(ctx context.Context) Transaction {
	return ctx.Value(transactionKey)
}
