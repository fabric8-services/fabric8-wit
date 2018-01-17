package id_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func TestMapFromSlice(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		require.Equal(t, id.Map{}, id.MapFromSlice(id.Slice{}))
	})
	a := uuid.FromStringOrNil("4c6ed1b1-8de6-4ebc-8238-95e2e28ac0a6")
	b := uuid.FromStringOrNil("a5bb3827-432c-4ab1-9ce2-4de8c7261a2b")
	t.Run("w/o duplicates", func(t *testing.T) {
		require.Equal(t, id.Map{a: {}, b: {}}, id.MapFromSlice(id.Slice{a, b}))
	})
	t.Run("with duplicates", func(t *testing.T) {
		require.Equal(t, id.Map{a: {}, b: {}}, id.MapFromSlice(id.Slice{a, b, a, b}))
	})
}

func TestMapToSlice(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		require.Equal(t, id.Slice{}, id.Map{}.ToSlice())
	})
	a := uuid.FromStringOrNil("4c6ed1b1-8de6-4ebc-8238-95e2e28ac0a6")
	b := uuid.FromStringOrNil("a5bb3827-432c-4ab1-9ce2-4de8c7261a2b")
	t.Run("with values", func(t *testing.T) {
		s := id.Map{a: {}, b: {}}.ToSlice()
		toBeFound := id.Map{a: {}, b: {}}
		for _, ID := range s {
			delete(toBeFound, ID)
		}
		require.Empty(t, toBeFound)
	})
}
