package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
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

	t.Run("Check subset unordered", func(t *testing.T) {
		needles := []interface{}{2, 1}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check full set ordered", func(t *testing.T) {
		needles := []interface{}{1, 2, 3, 4}
		assert.True(t, containsAll(haystack, needles))
	})

	t.Run("Check full set unordered", func(t *testing.T) {
		needles := []interface{}{2, 1, 4, 3}
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
