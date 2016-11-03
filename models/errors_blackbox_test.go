package models_test

import (
	"testing"

	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
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
