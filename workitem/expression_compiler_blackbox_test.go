package workitem_test

import (
	"runtime/debug"
	"testing"

	c "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, c.Equals(c.Field("foo.bar"), c.Literal(23)), "(Fields@>'{\"foo.bar\" : 23}')", []interface{}{}, nil)
	expect(t, c.Equals(c.Field("foo"), c.Literal(23)), "(foo = ?)", []interface{}{23}, nil)
	expect(t, c.Equals(c.Field("Type"), c.Literal("abcd")), "(type = ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Type"), c.Literal("abcd")), "(type != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Version"), c.Literal("abcd")), "(version != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Number"), c.Literal("abcd")), "(number != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("SpaceID"), c.Literal("abcd")), "(space_id != ?)", []interface{}{"abcd"}, nil)

	t.Run("test join", func(t *testing.T) {
		expect(t, c.Equals(c.Field("iteration.name"), c.Literal("abcd")), `(iter.name = ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"iterations": {
				Active:        true,
				TableName:     "iterations",
				TableAlias:    "iter",
				PrefixTrigger: "iteration.",
				On:            workitem.JoinOnJSONField(workitem.SystemIteration, "iter.id"),
			},
		})
		expect(t, c.Equals(c.Field("area.name"), c.Literal("abcd")), `(ar.name = ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"areas": {
				Active:        true,
				TableName:     "areas",
				TableAlias:    "ar",
				PrefixTrigger: "area.",
				On:            workitem.JoinOnJSONField(workitem.SystemArea, "ar.id"),
			},
		})
		expect(t, c.Equals(c.Field("codebase.name"), c.Literal("abcd")), `(cb.name = ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"codebases": {
				Active:        true,
				TableName:     "codebases",
				TableAlias:    "cb",
				PrefixTrigger: "codebase.",
				On:            workitem.JoinOnJSONField(workitem.SystemCodebase, "cb.id"),
			},
		})
		expect(t, c.Equals(c.Field("wit.name"), c.Literal("abcd")), `(wit.name = ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"work_item_types": {
				Active:        true,
				TableName:     "work_item_types",
				TableAlias:    "wit",
				PrefixTrigger: "wit.",
				On:            "wit.id = work_items.type",
			},
		})
	})
}

func TestAndOr(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, c.Or(c.Literal(true), c.Literal(false)), "(? or ?)", []interface{}{true, false}, nil)

	expect(t, c.And(c.Not(c.Field("foo.bar"), c.Literal("abcd")), c.Not(c.Literal(true), c.Literal(false))), "(NOT (Fields@>'{\"foo.bar\" : \"abcd\"}') and (? != ?))", []interface{}{true, false}, nil)
	expect(t, c.And(c.Equals(c.Field("foo.bar"), c.Literal("abcd")), c.Equals(c.Literal(true), c.Literal(false))), "((Fields@>'{\"foo.bar\" : \"abcd\"}') and (? = ?))", []interface{}{true, false}, nil)
	expect(t, c.Or(c.Equals(c.Field("foo.bar"), c.Literal("abcd")), c.Equals(c.Literal(true), c.Literal(false))), "((Fields@>'{\"foo.bar\" : \"abcd\"}') or (? = ?))", []interface{}{true, false}, nil)
}

func TestIsNull(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, c.IsNull("system.assignees"), "(Fields->>'system.assignees' IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("ID"), "(id IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Type"), "(type IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Version"), "(version IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Number"), "(number IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("SpaceID"), "(space_id IS NULL)", []interface{}{}, nil)
}

func expect(t *testing.T, expr c.Expression, expectedClause string, expectedParameters []interface{}, expectedJoins map[string]workitem.TableJoin) {
	clause, parameters, joins, compileErrors := workitem.Compile(expr)
	t.Run(expectedClause, func(t *testing.T) {
		t.Run("check for compile errors", func(t *testing.T) {
			require.Empty(t, compileErrors, "compile error. stack: %s", string(debug.Stack()))
		})
		t.Run("check clause", func(t *testing.T) {
			require.Equal(t, expectedClause, clause, "clause mismatch. stack: %s", string(debug.Stack()))
		})
		t.Run("check joins", func(t *testing.T) {
			require.Equal(t, expectedJoins, joins, "joins mismatch. stack: %s", string(debug.Stack()))
		})
		t.Run("check parameters", func(t *testing.T) {
			require.Equal(t, expectedParameters, parameters, "parameters mismatch. stack: %s", string(debug.Stack()))
		})
	})
}

func TestArray(t *testing.T) {
	assignees := []string{"1", "2", "3"}

	exp := c.Equals(c.Field("system.assignees"), c.Literal(assignees))
	where, _, _, compileErrors := workitem.Compile(exp)
	require.Empty(t, compileErrors)

	assert.Equal(t, "(Fields@>'{\"system.assignees\" : [\"1\",\"2\",\"3\"]}')", where)
}

func TestSubstring(t *testing.T) {
	t.Run("system.title with simple text", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system.title"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, "Fields->>'system.title' ILIKE ?", where)
	})
	t.Run("system.title with SQL injection text", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system.title;DELETE FROM work_items"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, "Fields->>'system.title;DELETE FROM work_items' ILIKE ?", where)
	})

	t.Run("system.title with SQL injection text single quote", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system.title'DELETE FROM work_items"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.NotEmpty(t, compileErrors)
		assert.Len(t, compileErrors, 1)
		assert.Equal(t, "", where)
	})
}
