package models

import (
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := simpleError{message: "foo"}
	assert.Equal(t, "foo", e.Error())
}

func TestBadParameterError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := BadParameterError{parameter: "foo", value: "bar"}
	assert.Equal(t, "Bad value for parameter 'foo': 'bar'", e.Error())
}

func TestNotFoundError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := NotFoundError{entity: "foo", ID: "bar"}
	assert.Equal(t, "foo with id 'bar' not found", e.Error())
}
