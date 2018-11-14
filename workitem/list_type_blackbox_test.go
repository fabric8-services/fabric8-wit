package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}

	// Test type incompatibility
	assert.False(t, a.Equal(convert.DummyEqualer{}))

	// Test simple type difference
	b := ListType{
		SimpleType:    SimpleType{Kind: KindString},
		ComponentType: SimpleType{Kind: KindString},
	}
	assert.False(t, a.Equal(b))

	// Test component type difference
	c := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindInteger},
	}
	assert.False(t, a.Equal(c))

	// Test equality
	d := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}
	assert.True(t, d.Equal(a))
	assert.True(t, a.Equal(d)) // test the inverse
}

func TestListType_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		obj     ListType
		wantErr bool
	}{
		{"ok", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
			DefaultValue:  "the",
		}, false},
		{"invalid type", ListType{
			SimpleType:    SimpleType{Kind: KindInteger},
			ComponentType: SimpleType{Kind: KindString},
			DefaultValue:  "the",
		}, true},
		{"invalid component type (enum)", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindEnum},
			DefaultValue:  "the",
		}, true},
		{"invalid component type (list)", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindList},
			DefaultValue:  "the",
		}, true},
		{"invalid default value (string expect, int provided)", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
			DefaultValue:  42,
		}, true},
		{"invalid default value (int expect, string provided)", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindInteger},
			DefaultValue:  "foo",
		}, true},
		{"invalid default value (int expect, array of int provided)", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindInteger},
			DefaultValue:  []int{1, 2, 3},
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

func TestListType_GetDefaultValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		listType ListType
		output   interface{}
	}{
		{"ok - string list", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
			DefaultValue:  "the",
		}, "the"},
		{"ok - default is nil", ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.output, tt.listType.GetDefaultValue())

		})
	}
}

func TestListType_SetDefaultValue(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	tests := []struct {
		name           string
		listType       ListType
		defVal         interface{}
		expectedOutput FieldType
		wantErr        bool
	}{
		{"set default to allowed value",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			"second",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
				DefaultValue:  "second",
			},
			false},
		{"set default to nil",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			nil,
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
				DefaultValue:  nil,
			},
			false},
		{"set default to not-allowed (wrong component type)",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			123,
			nil,
			true},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := tt.listType.SetDefaultValue(tt.defVal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}

func TestListType_ConvertToStringSlice(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	tests := []struct {
		name           string
		list           ListType
		defVal         interface{}
		expectedOutput []string
		wantErr        bool
	}{
		{"convert empty value",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			[]interface{}{},
			[]string{},
			false},
		{"convert single value",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			[]interface{}{"entry"},
			[]string{"entry"},
			false},
		{"convert multiple values",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			[]interface{}{"one", "two", "three"},
			[]string{"one", "two", "three"},
			false},
		{"convert single value - integer",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindInteger},
			},
			[]interface{}{42},
			[]string{"42"},
			false},
		{"convert multiple values - integer",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindInteger},
			},
			[]interface{}{1, 2, 3},
			[]string{"1", "2", "3"},
			false},
		{"convert single value - float",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindFloat},
			},
			[]interface{}{42.5},
			[]string{"42.500000"},
			false},
		{"convert multiple values - float",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindFloat},
			},
			[]interface{}{1.4, 2.3, 3.2},
			[]string{"1.400000", "2.300000", "3.200000"},
			false},
		{"convert single value - label",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindLabel},
			},
			[]interface{}{"somelabel"},
			[]string{"somelabel"},
			false},
		{"convert multiple values - float",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindLabel},
			},
			[]interface{}{"someLabel1", "someLabel2", "someLabel3"},
			[]string{"someLabel1", "someLabel2", "someLabel3"},
			false},
		{"nil value",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			nil,
			[]string{},
			false},
		{"fail - invalid type",
			ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
			1,
			[]string{},
			true},
	}
	for _, tt := range tests {
		tt := tt // needed for parallel running to capture range
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output, err := tt.list.ConvertToStringSlice(tt.defVal)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}
