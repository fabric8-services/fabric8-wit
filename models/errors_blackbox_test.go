package models_test

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

func TestNewInternalError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := models.NewInternalError("System disk could not be read")

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewConversionError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := models.NewConversionError("Couldn't convert workitem")

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewBadParameterError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := 10
	expectedValue := 11
	err := models.NewBadParameterError(param, value)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v'", param, value), err.Error())
	err = models.NewBadParameterError(param, value).Expected(expectedValue)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v' (expected: '%v')", param, value, expectedValue), err.Error())
}

func TestNewNotFoundError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := "10"
	err := models.NewNotFoundError(param, value)
	assert.Equal(t, fmt.Sprintf("%s with id '%s' not found", param, value), err.Error())
}
