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

	t.Run("Check for existence", func(t *testing.T) {
		needle := interface{}(3)
		assert.True(t, contains(haystack, needle))
	})

	t.Run("Check for absence", func(t *testing.T) {
		needle := interface{}(42)
		assert.False(t, contains(haystack, needle))
	})
}

func TestEnumTypeContainsAll(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	haystack := []interface{}{1, 2, 3, 4}

	t.Run("Check single value", func(t *testing.T) {
		needles := []interface{}{1}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check subset", func(t *testing.T) {
		needles := []interface{}{1, 3}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check subset ordered", func(t *testing.T) {
		needles := []interface{}{1, 2}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check full set ordered", func(t *testing.T) {
		needles := []interface{}{1, 2, 3, 4}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check empty set (should return true)", func(t *testing.T) {
		needles := []interface{}{}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check for absence, single", func(t *testing.T) {
		needles := []interface{}{42}
		assert.False(t, containsAll(haystack, needles))
	})

	t.Run("Check for absence, multi", func(t *testing.T) {
		needles := []interface{}{42, 23}
		assert.False(t, containsAll(haystack, needles))
	})

	t.Run("Check for different type, simple", func(t *testing.T) {
		haystack := []interface{}{"hello", "world", "!"}
		needles := []interface{}{"world", "!"}
		assert.True(t, containsAll(haystack, needles))
		needles = []interface{}{"none"}
		assert.False(t, containsAll(haystack, needles))
	})

	t.Run("Check for different type, struct", func(t *testing.T) {
		type needle struct {
			id   int
			name string
		}
		haystack := []interface{}{
			needle{id: 1, name: "One"},
			needle{id: 2, name: "Two"},
			needle{id: 3, name: "Three"},
		}
		needles := []interface{}{
			needle{id: 1, name: "One"},
			needle{id: 2, name: "Two"},
		}
		assert.True(t, containsAll(haystack, needles))
		needles = []interface{}{
			needle{id: 4, name: "Four"},
		}
		assert.False(t, containsAll(haystack, needles))
	})
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
