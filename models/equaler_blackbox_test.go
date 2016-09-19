package models_test

import (
	"testing"

	. "github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
)

// foo implements the Equaler interface
type foo struct{}

// Ensure foo implements the Equaler interface
var _ Equaler = foo{}
var _ Equaler = (*foo)(nil)

func (f foo) Equal(u Equaler) bool {
	_, ok := u.(foo)
	if !ok {
		return false
	}
	return true
}

func TestDummyEqualerEqual(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := DummyEqualer{}
	b := DummyEqualer{}

	// Test for type difference
	assert.False(t, a.Equal(foo{}))

	// Test for equality
	assert.True(t, a.Equal(b))
}
