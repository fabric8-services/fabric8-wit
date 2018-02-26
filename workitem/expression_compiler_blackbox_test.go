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

	// TODO(kwk): I've found out that we currently cannot handle when the field
	// expression is on the right side. This should be fixed
	//
	// expect(t, c.Not(c.Literal("abcd"), c.Field("SpaceID")), "(? != space_id)", []interface{}{"abcd"}, nil)
	// expect(t, c.Equals(c.Literal(23), c.Field("foo.bar")), "(Fields@>'{\"foo.bar\" : 23}')", []interface{}{}, nil)

	t.Run("test iteration join", func(t *testing.T) {
		expect(t, c.Equals(c.Field("iteration.name"), c.Literal("abcd")), `(iter.name = ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"iterations": {
				Active:         true,
				TableName:      "iterations",
				TableAlias:     "iter",
				PrefixTrigger:  "iteration.",
				On:             `fields@> concat('{"system.iteration": "', iter.id, '"}')::jsonb`,
				AllowedColumns: []string{"name"},
			},
		})
		expect(t, c.Not(c.Field("iteration.name"), c.Literal("abcd")), `(iter.name != ?)`, []interface{}{"abcd"}, map[string]workitem.TableJoin{
			"iterations": {
				Active:         true,
				TableName:      "iterations",
				TableAlias:     "iter",
				PrefixTrigger:  "iteration.",
				On:             `fields@> concat('{"system.iteration": "', iter.id, '"}')::jsonb`,
				AllowedColumns: []string{"name"},
			},
		})
		t.Run("test only name field is allowed", func(t *testing.T) {
			// given
			expr := c.Equals(c.Field("iteration.somec.NotAllowedc.Field"), c.Literal("abcd"))
			// when
			_, _, _, compileErrors := workitem.Compile(expr)
			// then
			require.NotEmpty(t, compileErrors)
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
		require.Empty(t, compileErrors)

		assert.Equal(t, 1, len(compileErrors))
		assert.Equal(t, "", where)
	})
}
