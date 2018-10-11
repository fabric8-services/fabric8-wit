package search

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"testing"

	c "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMap(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	t.Run(EQ, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"space": { "%s": "openshiftio"}}`, EQ)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		expectedQuery := Query{Name: "space", Value: &openshiftio}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("$SUBSTR", func(t *testing.T) {
		t.Parallel()
		// given
		substr := "openshiftio"
		input := fmt.Sprintf(`{"title": { "$SUBSTR": "%s"}}`, substr)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		expectedQuery := Query{Name: "title", Value: &substr, Substring: true}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("$SUBSTR within $AND", func(t *testing.T) {
		t.Parallel()
		// given
		openshiftio := "openshiftio"
		title := "sometitle"
		input := fmt.Sprintf(`{"$AND": [{"space": "%s"}, {"title": { "$SUBSTR": "%s"}}]}`, openshiftio, title)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		expectedQuery := Query{Name: AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "title", Value: &title, Substring: true}},
		}

		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("Equality with NULL value", func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"assignee": null}`)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		expectedQuery := Query{Name: "assignee", Value: nil}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run(EQ+" with NULL value", func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"label": { "%s": null}}`, EQ)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		expectedQuery := Query{Name: "label", Value: nil}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run(NE, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"space": { "%s": "openshiftio"}}`, NE)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		expectedQuery := Query{Name: "space", Value: &openshiftio, Negate: true}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run(AND, func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + AND + `": [{"space": "openshiftio"}, {"status": "NEW"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		expectedQuery := Query{Name: AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("Multiple "+AND, func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + AND + `": [{"space": "openshiftio"}, {"status": "NEW"}, {"title": "hello"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		title := "hello"
		expectedQuery := Query{Name: AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status},
			{Name: "title", Value: &title}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("Multiple "+OR, func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + OR + `": [{"space": "openshiftio"}, {"status": "NEW"}, {"title": "hello"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		title := "hello"
		expectedQuery := Query{Name: OR, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status},
			{Name: "title", Value: &title}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("AND with EQ", func(t *testing.T) {
		t.Parallel()
		// given
		input := `{"` + AND + `": [{"space": {"$EQ": "openshiftio"}}, {"status": "NEW"}]}`
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		openshiftio := "openshiftio"
		status := "NEW"
		expectedQuery := Query{Name: AND, Children: []Query{
			{Name: "space", Value: &openshiftio},
			{Name: "status", Value: &status}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run("Minimal OR and AND operation", func(t *testing.T) {
		t.Parallel()
		input := `
			{"` + OR + `": [{"` + AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
	        {"` + AND + `": [{"space": "rhel"}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: OR, Children: []Query{
			{Name: AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area, Child: true}}},
			{Name: AND, Children: []Query{
				{Name: "space", Value: &rhel}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run("minimal OR and AND and Negate operation", func(t *testing.T) {
		t.Parallel()
		input := `
		{"` + OR + `": [{"` + AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
			 {"` + AND + `": [{"space": "rhel", "negate": true}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: OR, Children: []Query{
			{Name: AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area, Child: true}}},
			{Name: AND, Children: []Query{
				{Name: "space", Value: &rhel, Negate: true}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run("minimal OR and AND and Negate operation with EQ", func(t *testing.T) {
		t.Parallel()
		input := `
		{"` + OR + `": [{"` + AND + `": [{"space": "openshiftio"},
                         {"area": "planner"}]},
			 {"` + AND + `": [{"space": {"$EQ": "rhel"}, "negate": true}]}]}`
		fm := map[string]interface{}{}

		// Parsing/Unmarshalling JSON encoding/json
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		q := &Query{}

		parseMap(fm, q)

		openshiftio := "openshiftio"
		area := "planner"
		rhel := "rhel"
		expected := &Query{Name: OR, Children: []Query{
			{Name: AND, Children: []Query{
				{Name: "space", Value: &openshiftio},
				{Name: "area", Value: &area, Child: true}}},
			{Name: AND, Children: []Query{
				{Name: "space", Value: &rhel, Negate: true}}},
		}}
		assert.Equal(t, expected, q)
	})

	t.Run(IN, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"status": { "%s": ["NEW", "OPEN"]}}`, IN)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualQuery := Query{}
		parseMap(fm, &actualQuery)
		// then
		new := "NEW"
		open := "OPEN"
		expectedQuery := Query{Name: OR, Children: []Query{
			{Name: "status", Value: &new},
			{Name: "status", Value: &open}},
		}
		assert.Equal(t, expectedQuery, actualQuery)
	})

	t.Run(OPTS, func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"%s": {"parent-exists": true, "tree-view": true}}`, OPTS)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		actualOptions := parseOptions(fm)
		// then
		expectedOptions := &QueryOptions{ParentExists: true, TreeView: true}
		assert.Equal(t, expectedOptions, actualOptions)
	})
	t.Run(OPTS+" complex query", func(t *testing.T) {
		t.Parallel()
		// given
		input := fmt.Sprintf(`{"%s":[{"title":"some"},{"state":"new"}],"%s": {"parent-exists": true, "tree-view": true}}`, AND, OPTS)
		// Parsing/Unmarshalling JSON encoding/json
		fm := map[string]interface{}{}
		err := json.Unmarshal([]byte(input), &fm)
		require.NoError(t, err)
		// when
		options := parseOptions(fm)
		actualQuery := Query{Options: options}

		// then
		expectedQuery := Query{Options: &QueryOptions{ParentExists: true, TreeView: true}}
		assert.Equal(t, expectedQuery, actualQuery)

		parseMap(fm, &actualQuery)
		title := "some"
		state := "new"
		expectedQuery = Query{Options: &QueryOptions{ParentExists: true, TreeView: true},
			Name: AND, Children: []Query{
				{Name: "title", Value: &title},
				{Name: "state", Value: &state}},
		}

		assert.Equal(t, expectedQuery, actualQuery)
	})

}

func TestParseFilterString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	t.Run("OPTS with other query", func(t *testing.T) {

		input := fmt.Sprintf(`{"$AND":[{"title":"some"},{"state":"new"}],"%s": {"parent-exists": true, "tree-view": true}}`, OPTS)
		actualExpr, options, err := ParseFilterString(context.Background(), input)
		expectedExpr := c.And(
			c.Equals(
				c.Field("system_title"),
				c.Literal("some"),
			),
			c.Equals(
				c.Field("system_state"),
				c.Literal("new"),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
		assert.Nil(t, err)
		expectedOptions := &QueryOptions{ParentExists: true, TreeView: true}
		assert.Equal(t, expectedOptions, options)
	})
}

func TestGenerateExpression(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	t.Run("Equals (top-level)", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Equals(
			c.Field("SpaceID"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(NOT+" (top-level)", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName, Negate: true}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Not(
			c.Field("SpaceID"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})
	t.Run(AND, func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system_state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(OR, func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: OR,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.Or(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system_state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run(NOT+" (nested)", func(t *testing.T) {
		t.Parallel()
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: AND,
			Children: []Query{
				{Name: "space", Value: &spaceName, Negate: true},
				{Name: "state", Value: &statusName},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Not(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("system_state"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{
			Name: AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "assignee", Value: nil},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),

			c.IsNull("system_assignees"),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value at top-level", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{
			Name: "assignee", Value: nil,
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.IsNull("system_assignees")

		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NULL value at top-level with Negate", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{
			Name: "assignee", Value: nil, Negate: true,
		}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.Error(t, err)
		require.Nil(t, actualExpr)
		assert.Contains(t, err.Error(), "negate for null not supported")
	})

	t.Run("NULL value with Negate", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{
			Name: AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "assignee", Value: nil, Negate: true},
			},
		}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.Error(t, err)
		require.Nil(t, actualExpr)
		assert.Contains(t, err.Error(), "negate for null not supported")
	})

}

func expectEqualExpr(t *testing.T, expectedExpr, actualExpr c.Expression) {
	require.NotNil(t, expectedExpr)
	require.NotNil(t, actualExpr)
	actualClause, actualParameters, actualJoins, actualErrs := workitem.Compile(actualExpr)
	require.Empty(t, actualErrs, "failed to compile actual expression. stack: %s", string(debug.Stack()))
	exprectedClause, expectedParameters, expectedJoins, expectedErrs := workitem.Compile(expectedExpr)
	require.Empty(t, expectedErrs, "failed to compile expected expression. stack: %s", string(debug.Stack()))
	require.Equal(t, exprectedClause, actualClause, "where clause differs")
	require.Equal(t, expectedParameters, actualParameters, "parameters differ")
	require.Equal(t, expectedJoins, actualJoins, "joins differ")
}

func TestWorkItemNumber(t *testing.T) {
	t.Run("search by number", func(t *testing.T) {
		// given
		spaceName := "openshiftio"
		wiNumber := "1"

		q := Query{
			Name: AND,
			Children: []Query{
				{Name: "space", Value: &spaceName},
				{Name: "number", Value: &wiNumber},
			},
		}
		// when
		actualExpr, _ := q.generateExpression()
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("SpaceID"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("Number"),
				c.Literal(wiNumber),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})
}

func TestGenerateExpressionWithNonExistingKey(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	t.Run("Empty query", func(t *testing.T) {
		t.Parallel()
		// given
		q := Query{}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.Error(t, err)
		require.Nil(t, actualExpr)
	})
	t.Run("Empty name", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "", Value: &spaceName}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.Error(t, err)
		require.Nil(t, actualExpr)
	})

	t.Run("No existing key", func(t *testing.T) {
		t.Parallel()
		// given
		spaceName := "openshiftio"
		q := Query{Name: "nonexistingkey", Value: &spaceName}
		// when
		actualExpr, err := q.generateExpression()
		// then
		require.Error(t, err)
		require.Nil(t, actualExpr)
	})
}
