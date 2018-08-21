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
		name   string
		obj    SimpleType
		output interface{}
	}{
		{"ok - int field: output 333", SimpleType{Kind: KindInteger, DefaultValue: 333}, 333},
		{"ok - float field: output 33.3", SimpleType{Kind: KindFloat, DefaultValue: 33.3}, 33.3},
		{"ok - string field: output \"foo\"", SimpleType{Kind: KindString, DefaultValue: "foo"}, "foo"},
		{"ok - string field nil default", SimpleType{Kind: KindString}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.output, tt.obj.GetDefaultValue())
		})
	}
}

func TestSimpleType_SetDefaultValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	tests := []struct {
		name           string
		enum           SimpleType
		defVal         interface{}
		expectedOutput FieldType
		wantErr        bool
	}{
		{"set default to allowed value",
			SimpleType{Kind: KindString},
			"foo",
			&SimpleType{Kind: KindString, DefaultValue: "foo"},
			false},
		{"set default to nil",
			SimpleType{Kind: KindString},
			nil,
			&SimpleType{Kind: KindString, DefaultValue: nil},
			false},
		{"set default to not-allowed value",
			SimpleType{Kind: KindString},
			123,
			nil,
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := tt.enum.SetDefaultValue(tt.defVal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}
