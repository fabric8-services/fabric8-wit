package models

import "github.com/lib/pq"

const (
	errCheckViolation  = "23514"
	errUniqueViolation = "23505"
)

func isCheckViolation(err error, constraintName string) bool {
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errCheckViolation && pqError.Constraint == constraintName
}

func isUniqueViolation(err error, indexName string) bool {
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errUniqueViolation && pqError.Constraint == indexName
}
