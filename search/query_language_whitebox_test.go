package search

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"testing"

	c "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	w "github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestQueryLanguageWhiteboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &queryLanguageWhiteboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type queryLanguageWhiteboxTest struct {
	gormtestsupport.DBTestSuite
	clean      func()
	modifierID uuid.UUID
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *queryLanguageWhiteboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *queryLanguageWhiteboxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.modifierID = testIdentity.ID
}

func (s *queryLanguageWhiteboxTest) TearDownTest() {
	s.clean()
}

func (s *queryLanguageWhiteboxTest) TestMinimalANDOperation() {
	input := `
	{"AND": [{"space": "openshiftio"},
                 {"status": "NEW"}
	]}`
	fm := map[string]interface{}{}

	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(input), &fm)

	if err != nil {
		panic(err)
	}
	q := &Query{}

	parseMap(fm, q)

	openshiftio := "openshiftio"
	status := "NEW"
	expected := &Query{Name: "AND", Children: []Query{
		Query{Name: "space", Value: &openshiftio},
		Query{Name: "status", Value: &status}},
	}
	assert.Equal(s.T(), expected, q)
}

func (s *queryLanguageWhiteboxTest) TestMinimalORandANDOperation() {
	input := `
	{"OR": [{"AND": [{"space": "openshiftio"},
                         {"area": "planner"}]},
	        {"AND": [{"space": "rhel"}]}]}`
	fm := map[string]interface{}{}

	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(input), &fm)

	if err != nil {
		panic(err)
	}
	q := &Query{}

	parseMap(fm, q)

	openshiftio := "openshiftio"
	area := "planner"
	rhel := "rhel"
	expected := &Query{Name: "OR", Children: []Query{
		Query{Name: "AND", Children: []Query{
			Query{Name: "space", Value: &openshiftio},
			Query{Name: "area", Value: &area}}},
		Query{Name: "AND", Children: []Query{
			Query{Name: "space", Value: &rhel}}},
	}}
	assert.Equal(s.T(), expected, q)
}

func (s *queryLanguageWhiteboxTest) TestMinimalORandANDandNegateOperation() {
	input := `
	{"OR": [{"AND": [{"space": "openshiftio"},
                         {"area": "planner"}]},
			 {"AND": [{"space": "rhel", "negate": true}]}]}`
	fm := map[string]interface{}{}

	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(input), &fm)

	if err != nil {
		panic(err)
	}
	q := &Query{}

	parseMap(fm, q)

	openshiftio := "openshiftio"
	area := "planner"
	rhel := "rhel"
	expected := &Query{Name: "OR", Children: []Query{
		Query{Name: "AND", Children: []Query{
			Query{Name: "space", Value: &openshiftio},
			Query{Name: "area", Value: &area}}},
		Query{Name: "AND", Children: []Query{
			Query{Name: "space", Value: &rhel, Negate: true}}},
	}}
	assert.Equal(s.T(), expected, q)
}

func expectEqualExpr(t *testing.T, expectedExpr, actualExpr c.Expression) {
	actualClause, actualParameters, actualErrs := w.Compile(actualExpr)
	if len(actualErrs) > 0 {
		debug.PrintStack()
		require.Nil(t, actualErrs, "failed to compile actual expression")
	}
	exprectedClause, expectedParameters, expectedErrs := w.Compile(expectedExpr)
	if len(expectedErrs) > 0 {
		debug.PrintStack()
		require.Nil(t, expectedErrs, "failed to compile expected expression")
	}
	require.Equal(t, exprectedClause, actualClause, "where clause differs")
	require.Equal(t, expectedParameters, actualParameters, "parameters differ")
}

func TestGenerateExpression(t *testing.T) {
	t.Run("EQUAL (top-level)", func(t *testing.T) {
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName}
		// when
		actualExpr := generateExpression(q)
		// then
		expectedExpr := c.Equals(
			c.Field("space"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NOT (top-level)", func(t *testing.T) {
		// given
		spaceName := "openshiftio"
		q := Query{Name: "space", Value: &spaceName, Negate: true}
		// when
		actualExpr := generateExpression(q)
		// then
		expectedExpr := c.Not(
			c.Field("space"),
			c.Literal(spaceName),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("AND", func(t *testing.T) {
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: "AND",
			Children: []Query{
				Query{Name: "space", Value: &spaceName},
				Query{Name: "status", Value: &statusName},
			},
		}
		// when
		actualExpr := generateExpression(q)
		// then
		expectedExpr := c.And(
			c.Equals(
				c.Field("space"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("status"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("OR", func(t *testing.T) {
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: "OR",
			Children: []Query{
				Query{Name: "space", Value: &spaceName},
				Query{Name: "status", Value: &statusName},
			},
		}
		// when
		actualExpr := generateExpression(q)
		// then
		expectedExpr := c.Or(
			c.Equals(
				c.Field("space"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("status"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})

	t.Run("NOT (nested)", func(t *testing.T) {
		// given
		statusName := "NEW"
		spaceName := "openshiftio"
		q := Query{
			Name: "AND",
			Children: []Query{
				Query{Name: "space", Value: &spaceName, Negate: true},
				Query{Name: "status", Value: &statusName},
			},
		}
		// when
		actualExpr := generateExpression(q)
		// then
		expectedExpr := c.And(
			c.Not(
				c.Field("space"),
				c.Literal(spaceName),
			),
			c.Equals(
				c.Field("status"),
				c.Literal(statusName),
			),
		)
		expectEqualExpr(t, expectedExpr, actualExpr)
	})
}
