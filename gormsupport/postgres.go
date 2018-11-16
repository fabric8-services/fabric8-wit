package gormsupport

import "github.com/lib/pq"

const (
	errCheckViolation      = "23514"
	errUniqueViolation     = "23505"
	errForeignKeyViolation = "23503"
	errInvalidCatalogName  = "3D000"
	errDataException       = "22" // This class contains many data representation related errors. See https://www.postgresql.org/docs/current/static/errcodes-appendix.html
)

// IsCheckViolation returns true if the error is a violation of the given check
func IsCheckViolation(err error, constraintName string) bool {
	if err == nil {
		return false
	}
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errCheckViolation && pqError.Constraint == constraintName
}

// IsDataException returns true if the error is of type data exception (eg: type
// mismatch, invalid delimiter, divide by zero, ...)
// See https://www.postgresql.org/docs/current/static/errcodes-appendix.html
func IsDataException(err error) bool {
	if err == nil {
		return false
	}
	pqError, ok := err.(*pq.Error)
	return ok && pqError.Code[:2] == errDataException
}

// IsInvalidCatalogName returns true if the given error says that the catalog
// is ivalid (e.g. database does not exist)
func IsInvalidCatalogName(err error) bool {
	if err == nil {
		return false
	}
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errInvalidCatalogName
}

// IsUniqueViolation returns true if the error is a violation of the given unique index
func IsUniqueViolation(err error, indexName string) bool {
	if err == nil {
		return false
	}
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errUniqueViolation && pqError.Constraint == indexName
}

// IsForeignKeyViolation returns true if the error is a violation of the given foreign key index
func IsForeignKeyViolation(err error, indexName string) bool {
	if err == nil {
		return false
	}
	pqError, ok := err.(*pq.Error)
	if !ok {
		return false
	}
	return pqError.Code == errForeignKeyViolation && pqError.Constraint == indexName
}
