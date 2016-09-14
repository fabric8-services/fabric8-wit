package models

import (
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
)

func TestSimpleError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := simpleError{message: "foo"}
	assert.Equal(t, "foo", e.Error())
}

func TestBadParameterError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := BadParameterError{parameter: "foo", value: "bar"}
	assert.Equal(t, fmt.Sprintf(stBadParameterErrorMsg, e.parameter, e.value), e.Error())
}

func TestNotFoundError_Error(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	e := NotFoundError{entity: "foo", ID: "bar"}
	assert.Equal(t, fmt.Sprintf(stNotFoundErrorMsg, e.entity, e.ID), e.Error())
}
