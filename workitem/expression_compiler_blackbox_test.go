package workitem_test

import (
	"reflect"
	"runtime/debug"
	"testing"

	. "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, Equals(Field("foo.bar"), Literal(23)), "(Fields@>'{\"foo.bar\" : 23}')", []interface{}{}, nil)
	expect(t, Equals(Field("foo"), Literal(23)), "(foo = ?)", []interface{}{23}, nil)
	expect(t, Equals(Field("Type"), Literal("abcd")), "(type = ?)", []interface{}{"abcd"}, nil)
	expect(t, Not(Field("Type"), Literal("abcd")), "(type != ?)", []interface{}{"abcd"}, nil)
	expect(t, Not(Field("Version"), Literal("abcd")), "(version != ?)", []interface{}{"abcd"}, nil)
	expect(t, Not(Field("Number"), Literal("abcd")), "(number != ?)", []interface{}{"abcd"}, nil)
	expect(t, Not(Field("SpaceID"), Literal("abcd")), "(space_id != ?)", []interface{}{"abcd"}, nil)

	// test joined tables
	//expect(t, Not(Field("iteration.name"), Literal("abcd")), "(iter.name != ?)", []interface{}{"abcd"}, nil)
	// TODO(kwk): This is the correct Negation syntax IMHO. Implement it
	//expect(t, Not(Field("iteration.name"), Literal("abcd")), "NOT(iter.name = ?)", []interface{}{"abcd"}, nil)
	expect(t, Equals(Field("iteration.name"), Literal("abcd")), `(iter.name = "abcd")`, []interface{}{"abcd"}, nil)
}

func TestAndOr(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, Or(Literal(true), Literal(false)), "(? or ?)", []interface{}{true, false}, nil)

	expect(t, And(Not(Field("foo.bar"), Literal("abcd")), Not(Literal(true), Literal(false))), "(NOT (Fields@>'{\"foo.bar\" : \"abcd\"}') and (? != ?))", []interface{}{true, false}, nil)
	expect(t, And(Equals(Field("foo.bar"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields@>'{\"foo.bar\" : \"abcd\"}') and (? = ?))", []interface{}{true, false}, nil)
	expect(t, Or(Equals(Field("foo.bar"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields@>'{\"foo.bar\" : \"abcd\"}') or (? = ?))", []interface{}{true, false}, nil)
}

func TestIsNull(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, IsNull("system.assignees"), "(Fields->>'system.assignees' IS NULL)", []interface{}{}, nil)
	expect(t, IsNull("ID"), "(id IS NULL)", []interface{}{}, nil)
	expect(t, IsNull("Type"), "(type IS NULL)", []interface{}{}, nil)
	expect(t, IsNull("Version"), "(version IS NULL)", []interface{}{}, nil)
	expect(t, IsNull("Number"), "(number IS NULL)", []interface{}{}, nil)
	expect(t, IsNull("SpaceID"), "(space_id IS NULL)", []interface{}{}, nil)
}

func expect(t *testing.T, expr Expression, expectedClause string, expectedParameters []interface{}, expectedJoins map[string]TableJoin) {
	clause, parameters, _, err := Compile(expr)
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

	// clause, parameters, joins, compileErrors := Compile(expr)
	// require.Empty(t, compileErrors, "compile error. stack: %s", string(debug.Stack()))
	// require.Equal(t, expectedClause, clause, "clause mismatch. stack: %s", string(debug.Stack()))
	// require.Equal(t, expectedJoins, joins, "joins mismatch. stack: %s", string(debug.Stack()))
	// require.Equal(t, expectedParameters, parameters, "parameters mismatch. stack: %s", string(debug.Stack()))
}

func TestArray(t *testing.T) {
	assignees := []string{"1", "2", "3"}

	exp := Equals(Field("system.assignees"), Literal(assignees))
	where, _, _, compileErrors := Compile(exp)
	require.Empty(t, compileErrors)

	assert.Equal(t, "(Fields@>'{\"system.assignees\" : [\"1\",\"2\",\"3\"]}')", where)
}

func TestSubstring(t *testing.T) {
	t.Run("system.title with simple text", func(t *testing.T) {
		title := "some title"

		exp := Substring(Field("system.title"), Literal(title))
		where, _, _, compileErrors := Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, "Fields->>'system.title' ILIKE ?", where)
	})
	t.Run("system.title with SQL injection text", func(t *testing.T) {
		title := "some title"

		exp := Substring(Field("system.title;DELETE FROM work_items"), Literal(title))
		where, _, _, compileErrors := Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, "Fields->>'system.title;DELETE FROM work_items' ILIKE ?", where)
	})

	t.Run("system.title with SQL injection text single quote", func(t *testing.T) {
		title := "some title"

		exp := Substring(Field("system.title'DELETE FROM work_items"), Literal(title))
		where, _, _, compileErrors := Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, 1, len(compileErrors))
		assert.Equal(t, "", where)
	})
}
