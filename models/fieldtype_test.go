package models

import "testing"

var stString = SimpleType{KindString}
var stInt = SimpleType{KindInteger}
var stFloat = SimpleType{KindFloat}
var stDuration = SimpleType{KindDuration}
var stURL = SimpleType{KindURL}
var stList = SimpleType{KindList}

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
