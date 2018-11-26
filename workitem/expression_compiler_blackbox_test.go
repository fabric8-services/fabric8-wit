package workitem_test

import (
	"testing"

	c "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	defJoins := workitem.DefaultTableJoins()

	wiTbl := workitem.WorkItemStorage{}.TableName()
	expect(t, c.Equals(c.Field("foo_bar"), c.Literal(23)), `(`+workitem.Column(wiTbl, "fields")+` @> '{"foo_bar" : 23}')`, []interface{}{}, nil)
	expect(t, c.Equals(c.Field("foo.bar"), c.Literal(23)), `(`+workitem.Column(wiTbl, "foo.bar")+` = ?)`, []interface{}{23}, nil)
	expect(t, c.Equals(c.Field("foo"), c.Literal(23)), `(`+workitem.Column(wiTbl, "foo")+` = ?)`, []interface{}{23}, nil)
	expect(t, c.Equals(c.Field("Type"), c.Literal("abcd")), `(`+workitem.Column(wiTbl, "type")+` = ?)`, []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Type"), c.Literal("abcd")), `(`+workitem.Column(wiTbl, "type")+` != ?)`, []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Version"), c.Literal("abcd")), `(`+workitem.Column(wiTbl, "version")+` != ?)`, []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("Number"), c.Literal("abcd")), `(`+workitem.Column(wiTbl, "number")+` != ?)`, []interface{}{"abcd"}, nil)
	expect(t, c.Not(c.Field("SpaceID"), c.Literal("abcd")), `(`+workitem.Column(wiTbl, "space_id")+` != ?)`, []interface{}{"abcd"}, nil)

	t.Run("test join", func(t *testing.T) {
		t.Run("iteration", func(t *testing.T) {
			j := *defJoins["iteration"]
			j.Active = true
			j.HandledFields = []string{"name"}
			expect(t, c.Equals(c.Field("iteration.name"), c.Literal("abcd")), `(`+workitem.Column("iter", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("area", func(t *testing.T) {
			j := *defJoins["area"]
			j.Active = true
			j.HandledFields = []string{"name"}
			expect(t, c.Equals(c.Field("area.name"), c.Literal("abcd")), `(`+workitem.Column("ar", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("codebase", func(t *testing.T) {
			j := *defJoins["codebase"]
			j.Active = true
			j.HandledFields = []string{"url"}
			expect(t, c.Equals(c.Field("codebase.url"), c.Literal("abcd")), `(`+workitem.Column("cb", "url")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("work item type", func(t *testing.T) {
			j := *defJoins["work_item_type"]
			j.Active = true
			j.HandledFields = []string{"name"}
			expect(t, c.Equals(c.Field("wit.name"), c.Literal("abcd")), `(`+workitem.Column("wit", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
			expect(t, c.Equals(c.Field("work_item_type.name"), c.Literal("abcd")), `(`+workitem.Column("wit", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
			expect(t, c.Equals(c.Field("type.name"), c.Literal("abcd")), `(`+workitem.Column("wit", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("space", func(t *testing.T) {
			j := *defJoins["space"]
			j.Active = true
			j.HandledFields = []string{"name"}
			expect(t, c.Equals(c.Field("space.name"), c.Literal("abcd")), `(`+workitem.Column("space", "name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("creator", func(t *testing.T) {
			j := *defJoins["creator"]
			j.Active = true
			j.HandledFields = []string{"full_name"}
			expect(t, c.Equals(c.Field("creator.full_name"), c.Literal("abcd")), `(`+workitem.Column("creator", "full_name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
			expect(t, c.Equals(c.Field("author.full_name"), c.Literal("abcd")), `(`+workitem.Column("creator", "full_name")+` = ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
			expect(t, c.Not(c.Field("author.full_name"), c.Literal("abcd")), `(`+workitem.Column("creator", "full_name")+` != ?)`, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
		})
		t.Run("custom1 + custom2", func(t *testing.T) {
			oldTableJoins := workitem.DefaultTableJoins
			defer func() {
				workitem.DefaultTableJoins = oldTableJoins
			}()

			joins := workitem.TableJoinMap{
				"custom1": &workitem.TableJoin{
					TableName:        "custom1",
					TableAlias:       "cust1",
					PrefixActivators: []string{"custom1."},
				},
				"custom2": &workitem.TableJoin{
					TableName:          "custom2",
					TableAlias:         "cust2",
					PrefixActivators:   []string{"custom2."},
					ActivateOtherJoins: []string{"custom1"},
				},
			}
			workitem.DefaultTableJoins = func() workitem.TableJoinMap {
				return joins
			}
			j := *joins["custom1"]
			j.Active = true
			j.HandledFields = []string{"name"}
			k := *joins["custom2"]
			k.Active = true
			k.HandledFields = []string{"name"}
			k.ActivateOtherJoins = []string{"custom1"}
			expect(t, c.Or(
				c.Equals(c.Field("custom1.name"), c.Literal("abcd")),
				c.Equals(c.Field("custom2.name"), c.Literal("xyz")),
			), `((`+workitem.Column("cust1", "name")+` = ?) OR (`+workitem.Column("cust2", "name")+` = ?))`, []interface{}{"abcd", "xyz"}, []*workitem.TableJoin{&j, &k})
		})
		t.Run("iteration with two fields", func(t *testing.T) {
			j := *defJoins["iteration"]
			j.Active = true
			j.HandledFields = []string{"name", "created_at"}
			expect(t, c.Or(
				c.Equals(c.Field("iteration.name"), c.Literal("abcd")),
				c.Equals(c.Field("iteration.created_at"), c.Literal("123")),
			), `((`+workitem.Column("iter", "name")+` = ?) OR (`+workitem.Column("iter", "created_at")+` = ?))`, []interface{}{"abcd", "123"}, []*workitem.TableJoin{&j})
		})
		t.Run("board by id", func(t *testing.T) {

			columns := *defJoins["boardcolumns"]
			columns.Active = true
			columns.HandledFields = []string{"id"}
			expect(t,
				c.Equals(c.Field("board.id"), c.Literal("c20882bd-3a70-48a4-9784-3d6735992a43")),
				`(`+workitem.Column("boardcolumns", "id")+` = ?)`, []interface{}{"c20882bd-3a70-48a4-9784-3d6735992a43"}, []*workitem.TableJoin{&columns})
		})

		t.Run("parent", func(t *testing.T) {
			t.Run("by id", func(t *testing.T) {
				parent := *defJoins["parent"]
				parent.Active = true
				parent.HandledFields = []string{"id"}
				parent_link := *defJoins["parent_link"]
				parent_link.Active = true
				expect(t,
					c.Equals(c.Field("parent.id"), c.Literal("c20882bd-3a70-48a4-9784-3d6735992a43")),
					`(`+workitem.Column("parent", "id")+` = ?)`, []interface{}{"c20882bd-3a70-48a4-9784-3d6735992a43"}, []*workitem.TableJoin{&parent_link, &parent})
			})
			t.Run("by number", func(t *testing.T) {
				parent := *defJoins["parent"]
				parent.Active = true
				parent.HandledFields = []string{"number"}
				parent_link := *defJoins["parent_link"]
				parent_link.Active = true
				expect(t,
					c.Equals(c.Field("parent.number"), c.Literal("1234")),
					`(`+workitem.Column("parent", "number")+` = ?)`, []interface{}{"1234"}, []*workitem.TableJoin{&parent_link, &parent})
			})
			t.Run("by number or id", func(t *testing.T) {
				parent := *defJoins["parent"]
				parent.Active = true
				parent.HandledFields = []string{"number", "id"}
				parent_link := *defJoins["parent_link"]
				parent_link.Active = true
				expect(t,
					c.Or(
						c.Equals(c.Field("parent.number"), c.Literal("1234")),
						c.Equals(c.Field("parent.id"), c.Literal("5feea506-b0ab-4913-a08b-fe6a5234fa69")),
					),
					`((`+workitem.Column("parent", "number")+` = ?) OR (`+workitem.Column("parent", "id")+` = ?))`, []interface{}{"1234", "5feea506-b0ab-4913-a08b-fe6a5234fa69"}, []*workitem.TableJoin{&parent_link, &parent})
			})
		})
		t.Run("label name", func(t *testing.T) {
			columns := *defJoins["label"]
			columns.Active = true
			columns.HandledFields = []string{"name"}
			expect(t,
				c.Equals(c.Field("label.name"), c.Literal("planner")),
				`("lbl"."name" = ?)`, []interface{}{"planner"}, []*workitem.TableJoin{&columns})
		})
	})
	t.Run("test illegal field name", func(t *testing.T) {
		t.Run("double quote", func(t *testing.T) {
			_, _, _, compileErrors := workitem.Compile(c.Equals(c.Field(`foo"bar`), c.Literal(23)))
			require.NotEmpty(t, compileErrors)
			require.Contains(t, compileErrors[0].Error(), "field name must not contain double quotes")
		})
		t.Run("single quote", func(t *testing.T) {
			_, _, _, compileErrors := workitem.Compile(c.Equals(c.Field(`foo'bar`), c.Literal(23)))
			require.NotEmpty(t, compileErrors)
			require.Contains(t, compileErrors[0].Error(), "field name must not contain single quotes")
		})
	})
}

func TestAndOr(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, c.Or(c.Literal(true), c.Literal(false)), "(? OR ?)", []interface{}{true, false}, nil)

	wiTbl := workitem.WorkItemStorage{}.TableName()

	expect(t, c.And(c.Not(c.Field("foo_bar"), c.Literal("abcd")), c.Not(c.Literal(true), c.Literal(false))), `(NOT (`+workitem.Column(wiTbl, "fields")+` @> '{"foo_bar" : "abcd"}') AND (? != ?))`, []interface{}{true, false}, nil)
	expect(t, c.And(c.Equals(c.Field("foo_bar"), c.Literal("abcd")), c.Equals(c.Literal(true), c.Literal(false))), `((`+workitem.Column(wiTbl, "fields")+` @> '{"foo_bar" : "abcd"}') AND (? = ?))`, []interface{}{true, false}, nil)
	expect(t, c.Or(c.Equals(c.Field("foo_bar"), c.Literal("abcd")), c.Equals(c.Literal(true), c.Literal(false))), `((`+workitem.Column(wiTbl, "fields")+` @> '{"foo_bar" : "abcd"}') OR (? = ?))`, []interface{}{true, false}, nil)
}

func TestIsNull(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	wiTbl := workitem.WorkItemStorage{}.TableName()
	expect(t, c.IsNull("system_assignees"), `(`+workitem.Column(wiTbl, "fields")+`->>'system_assignees' IS NULL)`, []interface{}{}, nil)
	expect(t, c.IsNull("ID"), `(`+workitem.Column(wiTbl, "id")+` IS NULL)`, []interface{}{}, nil)
	expect(t, c.IsNull("Type"), `(`+workitem.Column(wiTbl, "type")+` IS NULL)`, []interface{}{}, nil)
	expect(t, c.IsNull("Version"), `(`+workitem.Column(wiTbl, "version")+` IS NULL)`, []interface{}{}, nil)
	expect(t, c.IsNull("Number"), `(`+workitem.Column(wiTbl, "number")+` IS NULL)`, []interface{}{}, nil)
	expect(t, c.IsNull("SpaceID"), `(`+workitem.Column(wiTbl, "space_id")+` IS NULL)`, []interface{}{}, nil)
}

func TestChild(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	defJoins := workitem.DefaultTableJoins()
	t.Run("iteration", func(t *testing.T) {
		j := *defJoins["iteration"]
		j.Active = true
		e := "(uuid(\"work_items\".fields->>'system_iteration') IN (\n\t\t\t\tSELECT iter.id\n\t\t\t\t\tWHERE\n\t\t\t\t\t\t(SELECT j.path\n\t\t\t\t\t\t\tFROM iterations j\n\t\t\t\t\t\t\tWHERE j.space_id = \"work_items\".\"space_id\" AND j.id = ? \n\t\t\t\t\t\t) @> iter.path\n\t\t\t\t\t\t\t  ))"
		expect(t, c.Child(c.Field("system_iteration"), c.Literal("abcd")), e, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
	})
	t.Run("area", func(t *testing.T) {
		j := *defJoins["area"]
		j.Active = true
		e := "(uuid(\"work_items\".fields->>'system_area') IN (\n\t\t\t\tSELECT ar.id\n\t\t\t\t\tWHERE\n\t\t\t\t\t\t(SELECT j.path\n\t\t\t\t\t\t\tFROM areas j\n\t\t\t\t\t\t\tWHERE j.space_id = \"work_items\".\"space_id\" AND j.id = ? \n\t\t\t\t\t\t) @> ar.path\n\t\t\t\t\t\t\t  ))"
		expect(t, c.Child(c.Field("system_area"), c.Literal("abcd")), e, []interface{}{"abcd"}, []*workitem.TableJoin{&j})
	})
	t.Run("label - error", func(t *testing.T) {
		j := *defJoins["label"]
		j.Active = true
		expr := c.Child(c.Field("system_label"), c.Literal("abcd"))
		_, _, _, compileErrors := workitem.Compile(expr)
		require.EqualError(t, compileErrors[0], "invalid field name for child expression: system_label")
	})

}

func expect(t *testing.T, expr c.Expression, expectedClause string, expectedParameters []interface{}, expectedJoins []*workitem.TableJoin) {
	clause, parameters, joins, compileErrors := workitem.Compile(expr)
	t.Run("check for compile errors", func(t *testing.T) {
		require.Empty(t, compileErrors, "compile error")
	})
	t.Run("check clause", func(t *testing.T) {
		require.Equal(t, expectedClause, clause, "clause mismatch")
	})
	t.Run("check parameters", func(t *testing.T) {
		require.Equal(t, expectedParameters, parameters, "parameters mismatch")
	})
	t.Run("check joins", func(t *testing.T) {
		// We could just use `require.Equal` on the two join array but that is
		// much harder to debug, therefore we do it manually.
		require.Len(t, joins, len(expectedJoins), "number of joins not matching the expected number of joins")
		for i, expected := range expectedJoins {
			require.Equal(t, expected, joins[i], "join at index #%d is not matching", i)
		}
	})
}

func TestArray(t *testing.T) {
	assignees := []string{"1", "2", "3"}

	exp := c.Equals(c.Field("system_assignees"), c.Literal(assignees))
	where, _, _, compileErrors := workitem.Compile(exp)
	require.Empty(t, compileErrors)
	wiTbl := workitem.WorkItemStorage{}.TableName()
	assert.Equal(t, `(`+workitem.Column(wiTbl, "fields")+` @> '{"system_assignees" : ["1","2","3"]}')`, where)
}

func TestSubstring(t *testing.T) {
	wiTbl := workitem.WorkItemStorage{}.TableName()
	t.Run("system_title with simple text", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system_title"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, workitem.Column(wiTbl, "fields")+`->>'system_title' ILIKE ?`, where)
	})
	t.Run("system_title with SQL injection text", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system_title;DELETE FROM work_items"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.Empty(t, compileErrors)

		assert.Equal(t, workitem.Column(wiTbl, "fields")+`->>'system_title;DELETE FROM work_items' ILIKE ?`, where)
	})

	t.Run("system_title with SQL injection text single quote", func(t *testing.T) {
		title := "some title"

		exp := c.Substring(c.Field("system_title'DELETE FROM work_items"), c.Literal(title))
		where, _, _, compileErrors := workitem.Compile(exp)
		require.NotEmpty(t, compileErrors)
		assert.Len(t, compileErrors, 1)
		assert.Equal(t, "", where)
	})
}
