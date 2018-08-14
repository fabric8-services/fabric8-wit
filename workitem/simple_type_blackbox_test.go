package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
)

func TestSimpleType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	t.Run("type difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindString}
		require.False(t, a.Equal(convert.DummyEqualer{}))
	})

	t.Run("kind difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindString}
		b := SimpleType{Kind: KindInteger}
		require.False(t, a.Equal(b))
	})

	t.Run("default difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindInteger, DefaultValue: 1}
		b := SimpleType{Kind: KindInteger}
		require.False(t, a.Equal(b))
	})
}

func TestSimpleType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		obj     SimpleType
		wantErr bool
	}{
		{"ok int field", SimpleType{Kind: KindInteger, DefaultValue: 333}, false},
		{"ok string field", SimpleType{Kind: KindString, DefaultValue: "foo"}, false},
		{"invalid default (int given, string expected)", SimpleType{Kind: KindString, DefaultValue: 333}, true},
		{"ok string field", SimpleType{Kind: KindInteger, DefaultValue: "foo"}, true},
		{"invalud kind (enum)", SimpleType{Kind: KindEnum, DefaultValue: "foo"}, true},
		{"invalid kind (list)", SimpleType{Kind: KindList, DefaultValue: "foo"}, true},
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

func TestSimpleType_GetDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		obj     SimpleType
		intput  interface{}
		output  interface{}
		wantErr bool
	}{
		{"ok - int field: input nil output 333", SimpleType{Kind: KindInteger, DefaultValue: 333}, nil, int32(333), false},
		{"ok - float field: input nil output 33.3", SimpleType{Kind: KindFloat, DefaultValue: 33.3}, nil, float64(33.3), false},
		{"ok - string field: input nil output \"foo\"", SimpleType{Kind: KindString, DefaultValue: "foo"}, nil, string("foo"), false},

		{"ok - int field: input 333 output 444", SimpleType{Kind: KindInteger, DefaultValue: 333}, 444, int32(444), false},
		{"ok - float field: input 44.4 output 44.4", SimpleType{Kind: KindFloat, DefaultValue: 33.3}, 44.4, float64(44.4), false},
		{"ok - string field: input \"bar\" output \"bar\"", SimpleType{Kind: KindString, DefaultValue: "foo"}, "bar", string("bar"), false},

		{"error - list field is invalid for a simple type", SimpleType{Kind: KindList, DefaultValue: "foo"}, nil, nil, true},
		{"error - enum field is invalid for a simple type", SimpleType{Kind: KindEnum, DefaultValue: "foo"}, nil, nil, true},

		{"error - input int on string field", SimpleType{Kind: KindString, DefaultValue: "foo"}, 123, nil, true},
		{"ok - input int on float field", SimpleType{Kind: KindFloat, DefaultValue: 123.0}, 22, float64(22.0), false},
		{"ok - input float on int field", SimpleType{Kind: KindInteger, DefaultValue: 123}, 22.0, int32(22), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := tt.obj.GetDefaultValue(tt.intput)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.output, output)
			}
		})
	}
}
