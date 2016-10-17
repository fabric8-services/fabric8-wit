package test

// NoTransactionSupport is an empty implementation of TransactionSupport
type NoTransactionSupport struct{}

// Begin implements TransactionSupport
func (NoTransactionSupport) Begin() error {
	return nil
}

// Commit implements TransactionSupport
func (NoTransactionSupport) Commit() error {
	return nil
}

// Rollback implements TransactionSupport
func (NoTransactionSupport) Rollback() error {
	return nil
}
