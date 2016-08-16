package models

import "testing"

var (
	stString   = SimpleType{KindString}
	stInt      = SimpleType{KindInteger}
	stFloat    = SimpleType{KindFloat}
	stDuration = SimpleType{KindDuration}
	stURL      = SimpleType{KindURL}
	stList     = SimpleType{KindList}
)

type input struct {
	t             FieldType
	value         interface{}
	expectedValue interface{}
	errorExpected bool
}

func TestSimpleTypeConversion(t *testing.T) {
	test_data := []input{
		{stString, "hello world", "hello world", false},
		{stString, "", "", false},
		{stString, 100, nil, true},
		{stString, 1.90, nil, true},

		{stInt, 100.0, nil, true},
		{stInt, 100, 100, false},
		{stInt, "100", nil, true},
		{stInt, true, nil, true},

		{stFloat, 1.1, 1.1, false},
		{stFloat, 1, nil, true},
		{stFloat, "a", nil, true},

		{stDuration, 0, 0, false},
		{stDuration, 1.1, nil, true},
		{stDuration, "duration", nil, true},

		{stURL, "http://www.google.com", "http://www.google.com", false},
		{stURL, "", nil, true},
		{stURL, "google", nil, true},
		{stURL, "http://google.com", "http://google.com", false},

		{stList, [4]int{1, 2, 3, 4}, [4]int{1, 2, 3, 4}, false},
		{stList, [2]string{"1", "2"}, [2]string{"1", "2"}, false},
		{stList, "", nil, true},
		// {stList, []int{}, []int{}, false}, need to find out the way for empty array.
		// because slices do not have equality operator.
	}
	for _, inp := range test_data {
		retVal, err := inp.t.ConvertToModel(inp.value)
		if retVal == inp.expectedValue && (err != nil) == inp.errorExpected {
			t.Log("test pass:", inp)
		} else {
			t.Error(retVal, err)
			t.Fail()
		}
	}
}

var (
	stEnum = SimpleType{KindEnum}
	enum   = EnumType{
		BaseType: stEnum,
		// ENUM with same type values
		Values: []interface{}{"new", "triaged", "WIP", "QA", "done"},
	}

	multipleTypeEnum = EnumType{
		BaseType: stEnum,
		// ENUM with different type values.
		Values: []interface{}{100, 1.1, "hello"},
	}
)

func TestEnumTypeConversion(t *testing.T) {
	data := []input{
		{enum, "string", nil, true},
		{enum, "triaged", "triaged", false},
		{enum, "done", "done", false},
		{enum, "", nil, true},
		{enum, 100, nil, true},

		{multipleTypeEnum, "abcd", nil, true},
		{multipleTypeEnum, 100, 100, false},
		{multipleTypeEnum, "hello", "hello", false},
	}
	for _, inp := range data {
		retVal, err := inp.t.ConvertToModel(inp.value)
		if retVal == inp.expectedValue && (err != nil) == inp.errorExpected {
			t.Log("test pass:", inp)
		} else {
			t.Error(retVal, err)
			t.Fail()
		}
	}
}
