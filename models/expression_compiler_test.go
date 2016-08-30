package models

import (
	"github.com/almighty/almighty-core/test"
	. "github.com/almighty/almighty-core/criteria"
	"reflect"
	"runtime/debug"
	"testing"
)

func TestField(t *testing.T) {
	test.SkiptTestIfNotUnitTest(t)
	expect(t, Equals(Field("foo"), Literal(23)), "(Fields->'foo' = ?::jsonb)", []interface{}{"23"})
	expect(t, Equals(Field("Type"), Literal("abcd")), "(Type = ?)", []interface{}{"abcd"})
}

func TestAndOr(t *testing.T) {
	test.SkiptTestIfNotUnitTest(t)
	expect(t, Or(Literal(true), Literal(false)), "(? or ?)", []interface{}{true, false})

	expect(t, And(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields->'foo' = ?::jsonb) and (? = ?))", []interface{}{"\"abcd\"", true, false})
	expect(t, Or(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields->'foo' = ?::jsonb) or (? = ?))", []interface{}{"\"abcd\"", true, false})
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

	if !reflect.DeepEqual(expectedParameters, parameters) {
		debug.PrintStack()
		t.Fatalf("parameters should be %v but is %v", expectedParameters, parameters)
	}
}
