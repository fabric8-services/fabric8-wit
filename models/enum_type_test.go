package models

import (
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnumType_Equal(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	stEnum := SimpleType{KindEnum}
	a := EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo", "bar"},
	}

	// Test type inequality
	assert.False(t, a.Equal(DummyEqualer{}))

	// Test simple type difference
	stInteger := SimpleType{KindInteger}
	b := EnumType{
		SimpleType: SimpleType{KindInteger},
		BaseType: stInteger,
	}
	assert.False(t, a.Equal(b))

	// Test base type difference
	c := EnumType{
		BaseType: stInteger,
	}
	assert.False(t, a.Equal(c))

	// Test values difference
	d := EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo1", "bar2"},
	}
	assert.False(t, a.Equal(d))
}

func TestEnumTypeContains(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	haystack := []interface{}{1,2,3,4}

	// Check for existence
	needle := interface{}(3)
	assert.True(t, contains(haystack, needle))

	// Check for absence
	needle = interface{}(42)
	assert.False(t, contains(haystack, needle))

}
