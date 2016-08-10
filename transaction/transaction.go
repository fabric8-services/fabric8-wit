package transaction

// Support abstracts from concrete transaction implementation
type Support interface {
	// Begin starts a transaction
	Begin() error
	// Commit commits a transaction
	Commit() error
	// Rollback rolls back the transaction
	Rollback() error
}

// Do executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Do(support Support, todo func() error) error {
	if err := support.Begin(); err != nil {
		return err
	}
	if err := todo(); err != nil {
		support.Rollback()
		return err
	}
	return support.Commit()
}
