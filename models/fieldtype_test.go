package models

import (
	"fmt"
	"testing"
)

var stString = SimpleType{KindString}
var stInt = SimpleType{KindInteger}
var stFloat = SimpleType{KindFloat}
var stDuration = SimpleType{KindDuration}
var stURL = SimpleType{KindURL}

type input struct {
	t             SimpleType
	value         interface{}
	expectedValue interface{}
}

func TestSimpleTypeConversion(t *testing.T) {
	// When expected value is set to nil
	// it means `ConvertToModel` should raise an error
	test_data := []input{
		{stString, "hello world", "hello world"},
		{stString, "", ""},
		{stString, 100, nil},
		{stString, 1.90, nil},

		{stInt, 100.0, nil},
		{stInt, 100, 100},
		{stInt, "100", nil},
		{stInt, true, nil},

		{stFloat, 1.1, 1.1},
		{stFloat, 1, nil},
		{stFloat, "a", nil},

		{stDuration, 0, 0},
		{stDuration, 1.1, nil},
		{stDuration, "duration", nil},

		{stURL, "http://www.google.com", "http://www.google.com"},
		{stURL, "", nil},
		{stURL, "google", nil},
		{stURL, "http://google.com", "http://google.com"},
	}
	for _, inp := range test_data {
		retVal, err := inp.t.ConvertToModel(inp.value)
		if retVal == inp.expectedValue {
			fmt.Println("test pass:", inp)
		} else {
			t.Error(retVal, err)
			t.Fail()
		}
	}
}
