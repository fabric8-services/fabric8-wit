package search

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
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
	expected := &Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
		&Query{Name: "space", Value: &openshiftio, Negate: false, Children: nil},
		&Query{Name: "status", Value: &status, Negate: false, Children: nil}},
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
	expected := &Query{Name: "OR", Value: nil, Negate: false, Children: &[]*Query{
		&Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
			&Query{Name: "space", Value: &openshiftio, Children: nil},
			&Query{Name: "area", Value: &area, Children: nil}}},
		&Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
			&Query{Name: "space", Value: &rhel, Children: nil}}},
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
	expected := &Query{Name: "OR", Value: nil, Negate: false, Children: &[]*Query{
		&Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
			&Query{Name: "space", Value: &openshiftio, Children: nil},
			&Query{Name: "area", Value: &area, Children: nil}}},
		&Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
			&Query{Name: "space", Value: &rhel, Negate: true, Children: nil}}},
	}}
	assert.Equal(s.T(), expected, q)
}

func criteriaExpect(t *testing.T, expr criteria.Expression, expectedClause string, expectedParameters []interface{}) {
	clause, parameters, err := workitem.Compile(expr)
	if len(err) > 0 {
		debug.PrintStack()
		t.Fatal(err[0].Error())
	}
	fmt.Printf("clause: %#v\n expectedClause %#v\n", clause, expectedClause)
	if clause != expectedClause {
		debug.PrintStack()
		t.Fatalf("clause should be %s but is %s", expectedClause, clause)
	}

	if !reflect.DeepEqual(expectedParameters, parameters) {
		debug.PrintStack()
		t.Fatalf("parameters should be %v but is %v", expectedParameters, parameters)
	}
}

func (s *queryLanguageWhiteboxTest) TestMinimalANDExpression() {
	openshiftio := "openshiftio"
	status := "NEW"
	q := Query{Name: "AND", Value: nil, Negate: false, Children: &[]*Query{
		&Query{Name: "space", Value: &openshiftio, Negate: false, Children: nil},
		&Query{Name: "status", Value: &status, Negate: false, Children: nil}},
	}
	//var result *criteria.Expression
	//criteriaExpression(q, result)
	result := generateExpression2(&q)

	expectedExpression := `((Fields@>'{"space" : ["openshiftio"]}') and (Fields@>'{"status" : ["NEW"]}'))`

	criteriaExpect(s.T(), result, expectedExpression, []interface{}{})
}
