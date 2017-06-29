package search

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
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
	expected := &Query{Name: "", Value: nil, Children: []*Query(nil)}
	assert.Equal(s.T(), expected, q)
}
