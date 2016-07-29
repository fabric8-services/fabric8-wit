package models

import (
	"runtime/debug"
	"testing"

	. "github.com/almighty/almighty-core/models/criteria"
)

func TestField(t *testing.T) {
	expect(t, Equals(Field("foo"), Value(23)), "(Fields->'foo' = ?)", []interface{}{23})
	expect(t, Equals(Field("Type"), Value(23)), "(Type = ?)", []interface{}{23})
}

func TestAndOr(t *testing.T) {
	expect(t, Or(Value(true), Value(false)), "(? or ?)", []interface{}{true, false})

	expect(t, And(Equals(Field("foo"), Value("abcd")), Equals(Value(true), Value(false))), "((Fields->'foo' = ?) and (? = ?))", []interface{}{"abcd", true, false})
	expect(t, Or(Equals(Field("foo"), Value("abcd")), Equals(Value(true), Value(false))), "((Fields->'foo' = ?) or (? = ?))", []interface{}{"abcd", true, false})
}

func expect(t *testing.T, expr Expression, expectedClause string, expectedParameters []interface{}) {
	clause, parameters, err := Compile(expr)
	if len(err) > 0 {
		debug.PrintStack()
		t.Fatal(err[0].Error())
	}
	if clause != expectedClause {
		debug.PrintStack()
		t.Fatalf("clause should be %s but is %s", expectedClause, clause)
	}
	if len(parameters) != len(expectedParameters) {
		debug.PrintStack()
		t.Fatalf("%d parameters instead of %d", len(parameters), len(expectedParameters))
	}

	for index, param := range expectedParameters {
		if param != parameters[index] {
			debug.PrintStack()
			t.Errorf("parameter %d should be %v, but is %v", index, param, parameters[index])
		}
	}
}
