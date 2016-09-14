package models_test

import (
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnumType_Equal(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	stEnum := models.SimpleType{models.KindEnum}
	a := models.EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo", "bar"},
	}

	// Test type inequality
	assert.False(t, a.Equal(models.DummyEqualer{}))

	// Test simple type difference
	stInteger := models.SimpleType{models.KindInteger}
	b := models.EnumType{
		SimpleType: models.SimpleType{models.KindInteger},
		BaseType:   stInteger,
	}
	assert.False(t, a.Equal(b))

	// Test base type difference
	c := models.EnumType{
		BaseType: stInteger,
	}
	assert.False(t, a.Equal(c))

	// Test values difference
	d := models.EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo1", "bar2"},
	}
	assert.False(t, a.Equal(d))
}
