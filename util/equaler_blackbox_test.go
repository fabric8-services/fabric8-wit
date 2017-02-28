package util_test

import (
	"testing"

	"github.com/almighty/almighty-core/test/resource"
	"github.com/almighty/almighty-core/util"
	"github.com/stretchr/testify/assert"
)

// foo implements the Equaler interface
type foo struct{}

// Ensure foo implements the Equaler interface
var _ util.Equaler = foo{}
var _ util.Equaler = (*foo)(nil)

func (f foo) Equal(u util.Equaler) bool {
	_, ok := u.(foo)
	if !ok {
		return false
	}
	return true
}

func TestDummyEqualerEqual(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := util.DummyEqualer{}
	b := util.DummyEqualer{}

	// Test for type difference
	assert.False(t, a.Equal(foo{}))

	// Test for equality
	assert.True(t, a.Equal(b))
}
