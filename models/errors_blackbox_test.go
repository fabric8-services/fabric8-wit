package models_test

import (
	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
)

func TestNewSimpleError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := models.NewSimpleError("Error reading database values")

	// not sure what assertion to do here
	t.Log(err)
}

func TestNewInternalError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := models.NewInternalError("System disk could not be read")

	// not sure what assertion to do here.
	t.Log(err)
}
