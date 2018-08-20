package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	w "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
)

func TestEnumType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := w.EnumType{
		SimpleType:       w.SimpleType{Kind: w.KindEnum},
		BaseType:         w.SimpleType{Kind: w.KindString},
		Values:           []interface{}{"foo", "bar"},
		RewritableValues: false,
		DefaultValue:     "fooooooobar",
	}
	t.Run("type inequality", func(t *testing.T) {
		require.False(t, a.Equal(convert.DummyEqualer{}))
	})

	t.Run("simple type difference", func(t *testing.T) {
		b := a
		b.SimpleType = w.SimpleType{Kind: w.KindArea}
		require.False(t, a.Equal(b))
	})

	t.Run("base type difference", func(t *testing.T) {
		b := a
		b.BaseType = w.SimpleType{Kind: w.KindInteger}
		require.False(t, a.Equal(b))
	})

	t.Run("default value difference", func(t *testing.T) {
		b := a
		b.DefaultValue = "foo"
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

func TestEnumType_GetDefaultValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	tests := []struct {
		name           string
		enum           w.EnumType
		input          interface{}
		expectedOutput interface{}
		wantErr        bool
	}{
		{"return first value of enum when input is nil", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindString},
			Values:     []interface{}{"first", "second", "third"},
		}, nil, "first", false},
		{"return input value as is in list of allowed values", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindString},
			Values:     []interface{}{"first", "second", "third"},
		}, "second", "second", false},
		{"return error when input value is not in list of allowed values", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindString},
			Values:     []interface{}{"first", "second", "third"},
		}, "fourth", nil, true},
		{"return error when input value is of wrong type", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindString},
			Values:     []interface{}{"first", "second", "third"},
		}, 123, nil, true},
		{"return input value converted to output type if possible", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindFloat},
			Values:     []interface{}{111.3, 123.0, 222.1},
		}, 123, 123.0, false},
		{"return error when input value cannot be converted to output type", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindInteger},
			Values:     []interface{}{111, 222, 333},
		}, 222.0, 222, true},
		{"return custom default when input value is nil", w.EnumType{
			SimpleType:   w.SimpleType{Kind: w.KindEnum},
			BaseType:     w.SimpleType{Kind: w.KindInteger},
			Values:       []interface{}{111, 222, 333},
			DefaultValue: 222,
		}, nil, 222, false},
		{"return error when custom default is of wrong type", w.EnumType{
			SimpleType:   w.SimpleType{Kind: w.KindEnum},
			BaseType:     w.SimpleType{Kind: w.KindInteger},
			Values:       []interface{}{111, 222, 333},
			DefaultValue: 222.0,
		}, nil, nil, true},
		{"return error when values are empty", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindInteger},
			Values:     []interface{}{},
		}, nil, nil, true},
		{"return error when values are nil", w.EnumType{
			SimpleType: w.SimpleType{Kind: w.KindEnum},
			BaseType:   w.SimpleType{Kind: w.KindInteger},
			Values:     nil,
		}, nil, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := tt.enum.GetDefaultValue(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}
func TestEnumType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		obj     w.EnumType
		wantErr bool
	}{
		{"ok", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindString},
			Values:           []interface{}{"who", "let", "the", "dogs", "out"},
			RewritableValues: false,
			DefaultValue:     "the",
		}, false},
		{"error - empty values", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindString},
			Values:           []interface{}{},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"error - nil values", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindString},
			Values:           nil,
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid type", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindString},
			BaseType:         w.SimpleType{Kind: w.KindString},
			Values:           []interface{}{"who", "let", "the", "dogs", "out"},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid base type (list)", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindList},
			Values:           []interface{}{"who", "let", "the", "dogs", "out"},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid base type (enum)", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindEnum},
			Values:           []interface{}{"who", "let", "the", "dogs", "out"},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid string values", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindString},
			Values:           []interface{}{"who", 1, "the", "dogs", "out"},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid integer values", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindInteger},
			Values:           []interface{}{1, 2, "the", 4, 5},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid default value (wrong type)", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindInteger},
			Values:           []interface{}{1, 2, 3, 4, 5},
			RewritableValues: false,
			DefaultValue:     "the",
		}, true},
		{"invalid default value (not in allowed values)", w.EnumType{
			SimpleType:       w.SimpleType{Kind: w.KindEnum},
			BaseType:         w.SimpleType{Kind: w.KindInteger},
			Values:           []interface{}{1, 2, 3, 4, 5},
			RewritableValues: false,
			DefaultValue:     42,
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.obj.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEnumType_EqualEnclosing(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := w.EnumType{
		SimpleType:       w.SimpleType{Kind: w.KindEnum},
		BaseType:         w.SimpleType{Kind: w.KindString},
		Values:           []interface{}{"foo", "bar", "baz"},
		RewritableValues: false,
	}

	t.Run("simple type difference", func(t *testing.T) {
		b := a
		b.SimpleType = w.SimpleType{Kind: w.KindArea}
		require.False(t, a.EqualEnclosing(b))
	})

	t.Run("base type difference", func(t *testing.T) {
		b := a
		b.BaseType = w.SimpleType{Kind: w.KindInteger}
		require.False(t, a.EqualEnclosing(b))
	})

	t.Run("value difference", func(t *testing.T) {
		t.Run("not equal", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo1", "bar2"}
			require.False(t, a.EqualEnclosing(b))
		})

		t.Run("new type has subset values", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo", "bar"}
			require.False(t, b.EqualEnclosing(a))
		})

		t.Run("new type has more than subset values but not all of old set", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo", "bar", "hello"}
			require.False(t, b.EqualEnclosing(a))
		})

		t.Run("new type has more than subset values", func(t *testing.T) {
			b := a
			b.Values = []interface{}{"foo", "bar", "baz", "hello"}
			require.True(t, b.EqualEnclosing(a))
		})

		t.Run("new type has empty values", func(t *testing.T) {
			b := a
			b.Values = []interface{}{}
			require.False(t, b.EqualEnclosing(a))
		})
	})
}
