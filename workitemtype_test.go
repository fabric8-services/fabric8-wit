// +build integration

package main

import (
	"testing"

	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/transaction"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
	"os"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type WorkItemTypeSuite struct {
	suite.Suite
	db       *gorm.DB
	ts       *models.GormTransactionSupport
	witRepo  *models.GormWorkItemTypeRepository
	typeCtrl WorkitemtypeController
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *WorkItemTypeSuite) SetupSuite() {
	fmt.Println("--- Setting up test suite WorkItemTypeSuite ---")

	dbHost := os.Getenv("ALMIGHTY_DB_HOST")
	if "" == dbHost {
		panic("The environment variable ALMIGHTY_DB_HOST is not specified or empty.")
	}

	var err error
	s.db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", dbHost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	s.ts = models.NewGormTransactionSupport(s.db)
	s.witRepo = models.NewWorkItemTypeRepository(s.ts)
	s.typeCtrl = WorkitemtypeController{ts: s.ts, witRepository: s.witRepo}

	// Migrate the schema
	if err := transaction.Do(s.ts, func() error {
		return migration.Perform(context.Background(), s.ts.TX(), s.witRepo)
	}); err != nil {
		panic(err.Error())
	}
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *WorkItemTypeSuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemTypeSuite ---")
	if s.db != nil {
		s.db.Close()
	}
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that non of the work item types that we will create already exist.
func (s *WorkItemTypeSuite) SetupTest() {
	s.T().Log("--- Running SetupTest for ---")
	// Remove all work item types from the db that will be created during these tests.
	s.db.Where("name = ?", "animal").Delete(models.WorkItemType{})
}

// The TearDownTest method will be run after every test in the suite.
func (s *WorkItemTypeSuite) TearDownTest() {
	s.T().Log("--- Running TearDownTest ---")
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *WorkItemTypeSuite) createWorkItemTypeAnimal() (http.ResponseWriter, *app.WorkItemType) {

	// Create an enumeration of animal names
	typeStrings := []string{"elephant", "blue whale", "Tyrannosaurus rex"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"
	typeFieldDef := app.FieldDefinition{
		Required: true,
		Type: &app.FieldType{
			BaseType: &stString,
			Kind:     "enum",
			Values:   typeEnum,
		},
	}

	// Create the type for the "color" field
	colorFieldDef := app.FieldDefinition{
		Required: false,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	payload := app.CreateWorkitemtypePayload{
		Fields: map[string]*app.FieldDefinition{
			"animal_type": &typeFieldDef,
			"color":       &colorFieldDef,
		},
		Name: "animal",
	}

	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, &s.typeCtrl, &payload)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

func (s *WorkItemTypeSuite) TestCreateWorkItemType() {
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)
}

func (s *WorkItemTypeSuite) TestShowWorkItemType() {
	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)

	_, wit2 := test.ShowWorkitemtypeOK(s.T(), nil, nil, &s.typeCtrl, wit.Name)

	assert.NotNil(s.T(), wit2)
	assert.EqualValues(s.T(), wit, wit2)
}

/*func (s *WorkItemTypeSuite) TestListWorkItemType() {
	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)

	_, wit2 := test.ListWorkitemtypeOK(s.T(), nil, nil, &s.typeCtrl, wit.Name)

	assert.NotNil(s.T(), wit2)
	assert.EqualValues(s.T(), wit, wit2)
}*/

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	suite.Run(t, new(WorkItemTypeSuite))
}
