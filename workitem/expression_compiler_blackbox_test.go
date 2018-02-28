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
	expect(t, c.Equals(c.Field("foo"), c.Literal(23)), "(work_items.foo = ?)", []interface{}{23}, nil)
	expect(t, c.Equals(c.Field("Type"), c.Literal("abcd")), "(work_items.type = ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Type"), c.Literal("abcd")), "(work_items.type != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Version"), c.Literal("abcd")), "(work_items.version != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Number"), c.Literal("abcd")), "(work_items.number != ?)", []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("SpaceID"), c.Literal("abcd")), "(work_items.space_id != ?)", []interface{}{"abcd"}, nil)

	t.Run("test join", func(t *testing.T) {
		expect(t, c.Equals(c.Field("iteration.name"), c.Literal("abcd")), `(iter.name = ?)`, []interface{}{"abcd"}, []string{"iteration"})
		expect(t, c.Equals(c.Field("area.name"), c.Literal("abcd")), `(ar.name = ?)`, []interface{}{"abcd"}, []string{"area"})
		expect(t, c.Equals(c.Field("codebase.url"), c.Literal("abcd")), `(cb.url = ?)`, []interface{}{"abcd"}, []string{"codebase"})
		expect(t, c.Equals(c.Field("wit.name"), c.Literal("abcd")), `(wit.name = ?)`, []interface{}{"abcd"}, []string{"work_item_type"})
		expect(t, c.Equals(c.Field("work_item_type.name"), c.Literal("abcd")), `(wit.name = ?)`, []interface{}{"abcd"}, []string{"work_item_type"})
		expect(t, c.Equals(c.Field("type.name"), c.Literal("abcd")), `(wit.name = ?)`, []interface{}{"abcd"}, []string{"work_item_type"})
		expect(t, c.Equals(c.Field("space.name"), c.Literal("abcd")), `(space.name = ?)`, []interface{}{"abcd"}, []string{"space"})
		expect(t, c.Equals(c.Field("creator.full_name"), c.Literal("abcd")), `(creator.full_name = ?)`, []interface{}{"abcd"}, []string{"creator"})
		expect(t, c.Equals(c.Field("author.full_name"), c.Literal("abcd")), `(creator.full_name = ?)`, []interface{}{"abcd"}, []string{"creator"})
		expect(t, c.Not(c.Field("author.full_name"), c.Literal("abcd")), `(creator.full_name != ?)`, []interface{}{"abcd"}, []string{"creator"})

		expect(t, c.Or(
			c.Equals(c.Field("iteration.name"), c.Literal("abcd")),
			c.Equals(c.Field("area.name"), c.Literal("xyz")),
		), `((iter.name = ?) or (ar.name = ?))`, []interface{}{"abcd", "xyz"}, []string{"iteration", "area"})

		expect(t, c.Or(
			c.Equals(c.Field("iteration.name"), c.Literal("abcd")),
			c.Equals(c.Field("iteration.created_at"), c.Literal("123")),
		), `((iter.name = ?) or (iter.created_at = ?))`, []interface{}{"abcd", "123"}, []string{"iteration"})
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
	expect(t, c.IsNull("ID"), "(work_items.id IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Type"), "(work_items.type IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Version"), "(work_items.version IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("Number"), "(work_items.number IS NULL)", []interface{}{}, nil)
	expect(t, c.IsNull("SpaceID"), "(work_items.space_id IS NULL)", []interface{}{}, nil)
}

func expect(t *testing.T, expr c.Expression, expectedClause string, expectedParameters []interface{}, expectedJoins []string) {
	clause, parameters, joins, compileErrors := workitem.Compile(expr)
	t.Run(expectedClause, func(t *testing.T) {
		t.Run("check for compile errors", func(t *testing.T) {
			require.Empty(t, compileErrors, "compile error. stack: %s", string(debug.Stack()))
		})
		t.Run("check clause", func(t *testing.T) {
			require.Equal(t, expectedClause, clause, "clause mismatch. stack: %s", string(debug.Stack()))
		})
		t.Run("check parameters", func(t *testing.T) {
			require.Equal(t, expectedParameters, parameters, "parameters mismatch. stack: %s", string(debug.Stack()))
		})
		t.Run("check joins", func(t *testing.T) {
			for _, k := range expectedJoins {
				_, ok := joins[k]
				require.True(t, ok, `joins is missing "%s". stack: %s`, k, string(debug.Stack()))
			}
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
