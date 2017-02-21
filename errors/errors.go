package errors

import "fmt"

const (
	stBadParameterErrorMsg         = "Bad value for parameter '%s': '%v'"
	stBadParameterErrorExpectedMsg = "Bad value for parameter '%s': '%v' (expected: '%v')"
	stNotFoundErrorMsg             = "%s with id '%s' not found"
)

// Constants that can be used to identify internal server errors
const (
	ErrInternalDatabase = "database_error"
)

type simpleError struct {
	message string
}

func (err simpleError) Error() string {
	return err.message
}

// NewInternalError returns the custom defined error of type InternalError.
func NewInternalError(msg string) InternalError {
	return InternalError{simpleError{msg}}
}

// NewUnauthorizedError returns the custom defined error of type UnauthorizedError.
func NewUnauthorizedError(msg string) UnauthorizedError {
	return UnauthorizedError{simpleError{msg}}
}

// InternalError means that the operation failed for some internal, unexpected reason
type InternalError struct {
	simpleError
}

// UnauthorizedError means that the operation is unauthorized
type UnauthorizedError struct {
	simpleError
}

// VersionConflictError means that the version was not as expected in an update operation
type VersionConflictError struct {
	simpleError
}

// NewVersionConflictError returns the custom defined error of type VersionConflictError.
func NewVersionConflictError(msg string) VersionConflictError {
	return VersionConflictError{simpleError{msg}}
}

// BadParameterError means that a parameter was not as required
type BadParameterError struct {
	parameter        string
	value            interface{}
	expectedValue    interface{}
	hasExpectedValue bool
}

// Error implements the error interface
func (err BadParameterError) Error() string {
	if err.hasExpectedValue {
		return fmt.Sprintf(stBadParameterErrorExpectedMsg, err.parameter, err.value, err.expectedValue)
	}
	return fmt.Sprintf(stBadParameterErrorMsg, err.parameter, err.value)

}

// Expected sets the optional expectedValue parameter on the BadParameterError
func (err BadParameterError) Expected(expexcted interface{}) BadParameterError {
	err.expectedValue = expexcted
	err.hasExpectedValue = true
	return err
}

// NewBadParameterError returns the custom defined error of type NewBadParameterError.
func NewBadParameterError(param string, actual interface{}) BadParameterError {
	return BadParameterError{parameter: param, value: actual}
}

// NewConversionError returns the custom defined error of type NewConversionError.
func NewConversionError(msg string) ConversionError {
	return ConversionError{simpleError{msg}}
}

// ConversionError error means something went wrong converting between different representations
type ConversionError struct {
	simpleError
}

// NotFoundError means the object specified for the operation does not exist
type NotFoundError struct {
	entity string
	ID     string
}

func (err NotFoundError) Error() string {
	return fmt.Sprintf(stNotFoundErrorMsg, err.entity, err.ID)
}

// NewNotFoundError returns the custom defined error of type NewNotFoundError.
func NewNotFoundError(entity string, id string) NotFoundError {
	return NotFoundError{entity: entity, ID: id}
}
