package models

import (
	"runtime/debug"
	"testing"

	. "github.com/almighty/almighty-core/criteria"
)

func TestField(t *testing.T) {
	expect(t, Equals(Field("foo"), Literal(23)), "(Fields->'foo' = '23')", 0)
	expect(t, Equals(Field("Type"), Literal("abcd")), "(Type = 'abcd')", 0)
}

func TestParameter(t *testing.T) {
	expect(t, And(Literal(true), Parameter()), "(true and ?)", 1)
}

func TestAndOr(t *testing.T) {
	expect(t, Or(Literal(true), Literal(false)), "(true or false)", 0)

	expect(t, And(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields->'foo' = '\"abcd\"') and (true = false))", 0)
	expect(t, Or(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields->'foo' = '\"abcd\"') or (true = false))", 0)
}

func expect(t *testing.T, expr Expression, expectedClause string, expectedParameters uint16) {
	clause, parameters, err := Compile(expr)
	if len(err) > 0 {
		debug.PrintStack()
		t.Fatal(err[0].Error())
	}
	if clause != expectedClause {
		debug.PrintStack()
		t.Fatalf("clause should be %s but is %s", expectedClause, clause)
	}
	if parameters != expectedParameters {
		debug.PrintStack()
		t.Fatalf("%d parameters instead of %d", parameters, expectedParameters)
	}
}
