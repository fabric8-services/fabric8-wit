package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumTypeContains(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	haystack := []interface{}{1, 2, 3, 4}

	// Check for existence
	needle := interface{}(3)
	assert.True(t, contains(haystack, needle))

	// Check for absence
	needle = interface{}(42)
	assert.False(t, contains(haystack, needle))
}

func TestEnumTypeDefaultValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// given
	vals := []interface{}{"first", "second", "third"}
	e := EnumType{
		SimpleType: SimpleType{Kind: KindEnum},
		BaseType:   SimpleType{Kind: KindString},
		Values:     vals,
	}

	t.Run("default to first value of enum", func(t *testing.T) {
		t.Parallel()
		// when
		def, err := e.DefaultValue(nil)
		// then
		require.NoError(t, err)
		require.Equal(t, def, vals[0])
	})

	t.Run("return value as is if not nil", func(t *testing.T) {
		t.Parallel()
		// when
		def, err := e.DefaultValue("second")
		// then
		require.NoError(t, err)
		require.Equal(t, def, vals[1])
	})

	t.Run("return value as is (even if it is not one of the permissable values)", func(t *testing.T) {
		t.Parallel()
		// when
		def, err := e.DefaultValue("not existing value")
		// then
		require.NoError(t, err)
		require.Equal(t, def, "not existing value")
	})

	t.Run("return error when values are nil", func(t *testing.T) {
		t.Parallel()
		// given
		a := EnumType{
			SimpleType: SimpleType{Kind: KindEnum},
			BaseType:   SimpleType{Kind: KindString},
		}
		// when
		def, err := a.DefaultValue(nil)
		// then
		require.Error(t, err)
		require.Nil(t, def)
	})

	t.Run("return error when values are empty", func(t *testing.T) {
		t.Parallel()
		// given
		a := EnumType{
			SimpleType: SimpleType{Kind: KindEnum},
			BaseType:   SimpleType{Kind: KindString},
			Values:     []interface{}{},
		}
		// when
		def, err := a.DefaultValue(nil)
		// then
		require.Error(t, err)
		require.Nil(t, def)
	})
}
