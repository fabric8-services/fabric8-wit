package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"
)

func TestSimpleType_EqualAndEqualValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	t.Run("type difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindString}
		require.False(t, a.Equal(convert.DummyEqualer{}))
		require.False(t, a.EqualValue(convert.DummyEqualer{}))
	})

	t.Run("kind difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindString}
		b := SimpleType{Kind: KindInteger}
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
	})

	t.Run("default difference", func(t *testing.T) {
		t.Parallel()
		a := SimpleType{Kind: KindInteger, DefaultValue: 1}
		b := SimpleType{Kind: KindInteger}
		require.False(t, a.Equal(b))
		require.False(t, a.EqualValue(b))
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

type testData struct {
	name string

	initialValue interface{}
	targetValue  interface{}

	initialFieldType FieldType
	targetFieldType  FieldType

	fieldConvertible bool
}

func getFieldTypeConversionTestData() []testData {
	k := KindString
	return []testData{
		// valid conversions
		{"ok - simple type to simple type",
			"foo1",
			"foo1",
			SimpleType{Kind: k},
			SimpleType{Kind: k},
			true},
		{"ok - simple type to list",
			"foo2",
			[]interface{}{"foo2"},
			SimpleType{Kind: k},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			true},
		{"ok - simple type to enum",
			"foo3",
			"foo3",
			SimpleType{Kind: k},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"red", "foo3", "blue"}},
			true},
		{"ok - list to list",
			[]interface{}{"foo4", "foo5"},
			[]interface{}{"foo4", "foo5"},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			true},
		{"ok - list to simple type",
			[]interface{}{"foo6"},
			"foo6",
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			SimpleType{Kind: k},
			true},
		{"ok - list to enum",
			[]interface{}{"foo7"},
			"foo7",
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"yellow", "foo7", "cyan"}},
			true},
		{"ok - enum to enum",
			"foo8",
			"foo8",
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Bach", "foo8", "Chapdelaine"}},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Kant", "Hume", "foo8", "Aristoteles"}},
			true},
		{"ok - enum to simple type",
			"foo9",
			"foo9",
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Schopenhauer", "foo9", "Duerer"}},
			SimpleType{Kind: k},
			true},
		{"ok - enum to list",
			"foo10",
			[]interface{}{"foo10"},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Sokrates", "foo10", "Fromm"}},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			true},
		// invalid conversions
		{"err - simple type (string) to simple type (int)",
			"foo11",
			nil,
			SimpleType{Kind: KindString},
			SimpleType{Kind: KindInteger},
			false},
		{"err - simple type (string) to list (integer)",
			"foo2",
			([]interface{})(nil),
			SimpleType{Kind: k},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: KindInteger}},
			false},
		{"err - simple type (string) to enum (float)",
			"foo3",
			11.1,
			SimpleType{Kind: k},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: KindFloat}, Values: []interface{}{11.1, 22.2, 33.3}},
			false},
		{"err - list (string) to list (float)",
			[]interface{}{"foo4", "foo5"},
			([]interface{})(nil),
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: KindFloat}},
			false},
		{"err - list (string) to simple type (int)",
			[]interface{}{"foo6"},
			nil,
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			SimpleType{Kind: KindInteger},
			false},
		{"err - list (string) to enum (float)",
			[]interface{}{"foo7"},
			11.1,
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: k}},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: KindFloat}, Values: []interface{}{11.1, 22.2, 33.3}},
			false},
		{"err - enum (string) to enum (float)",
			"foo8",
			11.1,
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Bach", "foo8", "Chapdelaine"}},
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: KindFloat}, Values: []interface{}{11.1, 22.2, 33.3}},
			false},
		{"err - enum (string) to simple type (float)",
			"foo9",
			nil,
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Schopenhauer", "foo9", "Duerer"}},
			SimpleType{Kind: KindFloat},
			false},
		{"err - enum (string) to list (float)",
			"foo10",
			([]interface{})(nil),
			EnumType{SimpleType: SimpleType{Kind: KindEnum}, BaseType: SimpleType{Kind: k}, Values: []interface{}{"Sokrates", "foo10", "Fromm"}},
			ListType{SimpleType: SimpleType{Kind: KindList}, ComponentType: SimpleType{Kind: KindFloat}},
			false},
	}
}

func TestConvertToModelWithType(t *testing.T) {
	for _, d := range getFieldTypeConversionTestData() {
		t.Run(d.name, func(t *testing.T) {
			convertedVal, err := d.initialFieldType.ConvertToModelWithType(d.targetFieldType, d.initialValue)
			if !d.fieldConvertible {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, convertedVal, d.targetValue)
		})
	}
}
