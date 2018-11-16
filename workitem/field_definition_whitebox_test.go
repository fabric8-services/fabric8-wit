package workitem

import (
	"encoding/json"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompatibleFields(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := FieldDefinition{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}

	t.Run("compatible field definition", func(t *testing.T) {
		t.Parallel()
		// given
		b := FieldDefinition{
			Label:       "b",
			Description: "description for 'b'",
			Required:    true,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
		}
		// then
		assert.True(t, compatibleFields(a, b), "fields %+v and %+v are not detected as being compatible", a, b)
	})
	t.Run("incompatible field definition (incompatible fields)", func(t *testing.T) {
		t.Parallel()
		// given
		c := FieldDefinition{
			Label:       "c",
			Description: "description for 'c'",
			Required:    true,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindInteger},
			},
		}
		// then
		assert.False(t, compatibleFields(a, c), "fields %+v and %+v are not detected as being incompatible", a, c)
	})
	t.Run("incompatible field definition (different required field)", func(t *testing.T) {
		t.Parallel()
		// given
		d := FieldDefinition{
			Label:       "c",
			Description: "description for 'd'",
			Required:    false,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
		}
		// then
		assert.False(t, compatibleFields(a, d), "fields %+v and %+v are not detected as being incompatible", a, d)
	})
}

func TestFieldDefinition_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// given
	a := FieldDefinition{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		ReadOnly:    false,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
		require.True(t, b.Equal(a))
		require.True(t, a.EqualValue(b))
		require.True(t, b.EqualValue(a))
	})
	t.Run("label", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Label = "b"
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("description", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Description = "description for b"
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("required", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Required = !a.Required
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("read-only", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ReadOnly = !a.ReadOnly
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("type", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Type = SimpleType{Kind: KindInteger}
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
}

func TestRawFieldDef_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// given
	a := rawFieldDef{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		ReadOnly:    false,
		Type:        &json.RawMessage{},
	}

	t.Run("equality", func(t *testing.T) {
		t.Parallel()
		b := a
		require.True(t, a.Equal(b))
		require.True(t, b.Equal(a))
		require.True(t, a.EqualValue(b))
		require.True(t, b.EqualValue(a))
	})
	t.Run("label", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Label = "b"
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("description", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Description = "description for b"
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("required", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Required = !a.Required
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("is read-only", func(t *testing.T) {
		t.Parallel()
		b := a
		b.ReadOnly = !a.ReadOnly
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
	t.Run("type", func(t *testing.T) {
		t.Parallel()
		b := a
		b.Type = nil
		require.False(t, a.Equal(b))
		require.False(t, b.Equal(a))
		require.False(t, a.EqualValue(b))
		require.False(t, b.EqualValue(a))
	})
}
