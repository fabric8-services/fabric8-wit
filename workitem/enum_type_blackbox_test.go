package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
)

func TestEnumType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := workitem.EnumType{
		SimpleType:       workitem.SimpleType{Kind: workitem.KindEnum},
		BaseType:         workitem.SimpleType{Kind: workitem.KindString},
		Values:           []interface{}{"foo", "bar"},
		RewritableValues: false,
	}

	t.Run("type inequality", func(t *testing.T) {
		require.False(t, a.Equal(convert.DummyEqualer{}))
	})

	t.Run("simple type difference", func(t *testing.T) {
		b := a
		b.SimpleType = workitem.SimpleType{Kind: workitem.KindArea}
		require.False(t, a.Equal(b))
	})

	t.Run("base type difference", func(t *testing.T) {
		b := a
		b.BaseType = workitem.SimpleType{Kind: workitem.KindInteger}
		require.False(t, a.Equal(b))
	})

	t.Run("value difference", func(t *testing.T) {
		t.Run("not equal", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo1", "bar2"}
			require.False(t, a.Equal(b))
		})

		t.Run("new type has overwritable values but old not", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo1", "bar2"}
			b.RewritableValues = true
			require.False(t, a.Equal(b))
		})

		t.Run("old type has overwritable values but new not", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo1", "bar2"}
			b.RewritableValues = true
			require.True(t, b.Equal(a))
		})
		t.Run("old and new type have overwritable values", func(t *testing.T) {
			b := a
			b.RewritableValues = true
			b.Values = []interface{}{"foo1", "bar2"}
			c := a
			c.RewritableValues = true
			require.True(t, b.Equal(c))
			require.True(t, c.Equal(b))
		})
	})
}
