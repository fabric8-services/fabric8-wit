package workitem_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/require"

	"github.com/davecgh/go-spew/spew"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
)

var (
	stString      = SimpleType{Kind: KindString}
	stIteration   = SimpleType{Kind: KindIteration}
	stInt         = SimpleType{Kind: KindInteger}
	stFloat       = SimpleType{Kind: KindFloat}
	stDuration    = SimpleType{Kind: KindDuration}
	stURL         = SimpleType{Kind: KindURL}
	stList        = SimpleType{Kind: KindList}
	stMarkup      = SimpleType{Kind: KindMarkup}
	stArea        = SimpleType{Kind: KindArea}
	stBoardColumn = SimpleType{Kind: KindBoardColumn}
	stInstant     = SimpleType{Kind: KindInstant}
)

type input struct {
	t             FieldType
	value         interface{}
	expectedValue interface{}
	errorExpected bool
}

func TestConvertToModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	for kind, validInvalid := range GetFieldTypeTestData(t) {
		fieldType := SimpleType{Kind: kind}
		t.Run(kind.String(), func(t *testing.T) {
			t.Run("legal", func(t *testing.T) {
				for _, inOut := range validInvalid.Valid {
					t.Run(fmt.Sprintf("%+v -> %+v", spew.Sdump(inOut.Input), spew.Sdump(inOut.Storage)), func(t *testing.T) {
						actual, err := fieldType.ConvertToModel(inOut.Input)
						require.NoError(t, err)
						require.Equal(t, inOut.Storage, actual)
					})
				}
				// for _, sample := range validInvalid.InvalidWhenRequired {
				// 	t.Run(fmt.Sprintf("%+v", spew.Sdump(sample)), func(t *testing.T) {
				// 		_, err := typ.ConvertToModel(sample)
				// 		require.NoError(t, err)
				// 	})
				// }
			})
			t.Run("illegal", func(t *testing.T) {
				for _, sample := range validInvalid.Invalid {
					t.Run(spew.Sdump(sample), func(t *testing.T) {
						_, err := fieldType.ConvertToModel(sample)
						require.Error(t, err)
					})
				}
			})
		})
	}
}

func TestConvertFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	for kind, validInvalid := range GetFieldTypeTestData(t) {
		fieldType := SimpleType{Kind: kind}
		t.Run(kind.String(), func(t *testing.T) {
			t.Run("legal", func(t *testing.T) {
				for _, inOut := range validInvalid.Valid {
					t.Run(fmt.Sprintf("%+v -> %+v", spew.Sdump(inOut.Storage), spew.Sdump(inOut.Output)), func(t *testing.T) {
						actual, err := fieldType.ConvertFromModel(inOut.Storage)
						require.NoError(t, err)
						require.Equal(t, inOut.Output, actual)
					})
				}
				// for _, sample := range validInvalid.InvalidWhenRequired {
				// 	t.Run(fmt.Sprintf("%+v", spew.Sdump(sample)), func(t *testing.T) {
				// 		_, err := typ.ConvertFromModel(sample)
				// 		require.NoError(t, err)
				// 	})
				// }
			})
			t.Run("illegal", func(t *testing.T) {
				for _, sample := range validInvalid.Invalid {
					t.Run(spew.Sdump(sample), func(t *testing.T) {
						_, err := fieldType.ConvertFromModel(sample)
						require.Error(t, err)
					})
				}
			})
		})
	}
}

var (
	stEnum = SimpleType{Kind: KindEnum}
	enum   = EnumType{
		SimpleType: stEnum,
		BaseType:   SimpleType{Kind: workitem.KindString},
		// ENUM with same type values
		Values: []interface{}{"new", "triaged", "WIP", "QA", "done"},
	}
)

func TestEnumTypeConversion(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	data := []input{
		{enum, "string", nil, true},
		{enum, "triaged", "triaged", false},
		{enum, "done", "done", false},
		{enum, "", nil, true},
		{enum, 100, nil, true},
	}
	for _, inp := range data {
		retVal, err := inp.t.ConvertToModel(inp.value)
		if retVal == inp.expectedValue && (err != nil) == inp.errorExpected {
			t.Log("test pass for input: ", inp)
		} else {
			t.Error(retVal, err)
			t.Fail()
		}
	}
}

var (
	intList = ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindInteger},
	}
	strList = ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}
)

func TestListTypeConversion(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	data := []input{
		{intList, []int{11, 2}, []interface{}{int32(11), int32(2)}, false},
		{intList, []string{"11", "2", "33.0"}, []interface{}{int32(11), int32(2), int32(33)}, false},
		{intList, []string{"11", "2", "3.141"}, ([]interface{})(nil), true},

		{strList, []string{"11", "2"}, []interface{}{"11", "2"}, false},
		{strList, []int{112, 2}, ([]interface{})(nil), true},
	}

	for _, inp := range data {
		retVal, err := inp.t.ConvertToModel(inp.value)
		require.Equal(t, inp.expectedValue, retVal, "with intput: %+v", inp.value)
		if (err != nil) != inp.errorExpected {
			t.Errorf(`
			In: %+v (%[1]T)
			Out: %+v (%[2]T)
			Expected: %+v (%[3]T)
			Err: %s
		`, inp.value, retVal, inp.expectedValue, err)
			t.Fail()
		}
	}
}
