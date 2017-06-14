package errors_test

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/resource"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewInternalError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := errors.NewInternalError(errs.New("system disk could not be read"))

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewConversionError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := errors.NewConversionError("Couldn't convert workitem")

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewBadParameterError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := 10
	expectedValue := 11
	err := errors.NewBadParameterError(param, value)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v'", param, value), err.Error())
	err = errors.NewBadParameterError(param, value).Expected(expectedValue)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v' (expected: '%v')", param, value, expectedValue), err.Error())
}

func TestNewNotFoundError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := "10"
	err := errors.NewNotFoundError(param, value)
	assert.Equal(t, fmt.Sprintf("%s with id '%s' not found", param, value), err.Error())
}

func TestNewUnauthorizedError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	msg := "Invalid token"
	err := errors.NewUnauthorizedError(msg)

	assert.Equal(t, msg, err.Error())
}

func TestNewForbiddenError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	msg := "Forbidden"
	err := errors.NewForbiddenError(msg)

	assert.Equal(t, msg, err.Error())
}
