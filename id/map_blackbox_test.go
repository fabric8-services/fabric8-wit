package id_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/id"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
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

func TestMapToString(t *testing.T) {
	// given
	a := uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94")
	b := uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8")
	c := uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809")
	m := id.Map{a: {}, b: {}, c: {}}
	// when
	res := m.ToString("; ", func(ID uuid.UUID) string { return fmt.Sprintf("(%s)", ID) })
	// then
	assert.Equal(t, 1, strings.Count(res, fmt.Sprintf("(%s)", a)))
	assert.Equal(t, 1, strings.Count(res, fmt.Sprintf("(%s)", b)))
	assert.Equal(t, 1, strings.Count(res, fmt.Sprintf("(%s)", c)))
	assert.Equal(t, 2, strings.Count(res, "; "))
}

func TestMapString(t *testing.T) {
	// given
	a := uuid.FromStringOrNil("9afc7d5c-9f4e-4a04-8359-71d72e5eed94")
	b := uuid.FromStringOrNil("4ce8076c-4997-4565-8272-9a3cb4d7a1a8")
	c := uuid.FromStringOrNil("0403d2cb-02d9-466f-88cd-65dc9247f809")
	m := id.Map{a: {}, b: {}, c: {}, c: {}}
	// when
	res := m.String()
	// then
	assert.Equal(t, 1, strings.Count(res, a.String()))
	assert.Equal(t, 1, strings.Count(res, b.String()))
	assert.Equal(t, 1, strings.Count(res, c.String()))
	assert.Equal(t, 2, strings.Count(res, ", "))
}
