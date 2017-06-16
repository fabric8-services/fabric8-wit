package search

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Query struct {
	Name     string
	Value    string
	Children []*Query
}

func parseMap(queryMap map[string]interface{}, q *Query) {
	for key, val := range queryMap {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			fmt.Println(key)
			c1 := &Query{}
			c := []c1{}
			q.Children = c
			parseMap(val.(map[string]interface{}), c)
		case []interface{}:
			fmt.Println(key)
		default:
			fmt.Println(key, ":", concreteVal)
			q.Name = key
			q.Value = concreteVal
		}
	}

}

func convert(input string) {
	m := map[string]interface{}{}

	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(input), &m)

	if err != nil {
		panic(err)
	}
	q := &Query{}
	parseMap(m, q)
}

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

func (s *queryLanguageWhiteboxTest) TestSmallestPossibleScenario() {
	q := `
	{"space": "openshiftio",
    "status": "NEW"
	}`
	qo := &Query{}
	convert(q)
}
