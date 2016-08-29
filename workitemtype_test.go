// +build integration

package main

import (
	"strconv"
	"testing"
	"time"
	//"strings"

	"fmt"
	"net/http"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func reandomUniqueString() string {
	// generates random string using timestamp
	now := time.Now().UnixNano() / 1000000
	return strconv.FormatInt(now, 10)
}

func TestCreateShowWorkItemType(t *testing.T) {
	// test-1
	// create a WIT (using workitem_repo)
	// try to create anohter WIT with same name - should Fail
	// remove the WIT (using GORM raw query)

	ts := models.NewGormTransactionSupport(db)
	witr := models.NewWorkItemTypeRepository(ts)
	controller := WorkitemtypeController{ts: ts, witRepository: witr}
	name := reandomUniqueString()

	st := app.FieldType{
		Kind: "user",
	}
	fd := app.FieldDefinition{
		Type:     &st,
		Required: true,
	}
	fields := map[string]*app.FieldDefinition{
		"system.owner": &fd,
	}
	payload := app.CreateWorkitemtypePayload{
		ExtendedTypeName: nil,
		Name:             name,
		Fields:           fields,
	}
	t.Log("creating WIT now.")
	_, created := test.CreateWorkitemtypeCreated(t, nil, nil, &controller, &payload)

	if created.Name == "" {
		t.Error("no Name")
	}
	t.Log("WIT created, Name=", created.Name)

	t.Log("Fetch recently created WIT")
	_, showWIT := test.ShowWorkitemtypeOK(t, nil, nil, &controller, name)

	if showWIT == nil {
		t.Error("Can not fetch WIT", name)
	}

	t.Log("Started cleanup for ", created.Name)
	db.Table("work_item_types").Where("name=?", name).Delete(&models.WorkItemType{})
	t.Log("Cleanup complete")
}

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
func (s *WorkItemTypeSuite) SetupSuite() {
	fmt.Println("--- Setting up test suite WorkItemTypeSuite ---")

	// for cleaner setup, assign the global db handle to the suite
	// TODO: (kwk) clean overall usage of globals in tests
	s.db = db

	s.ts = models.NewGormTransactionSupport(s.db)
	s.witRepo = models.NewWorkItemTypeRepository(s.ts)
	s.typeCtrl = WorkitemtypeController{ts: s.ts, witRepository: s.witRepo}
}

// The TearDownSuite method will run after all the tests in the suite have been run
func (s *WorkItemTypeSuite) TearDownSuite() {
	fmt.Println("--- Tearing down test suite WorkItemTypeSuite ---")
	s.T().Log("--- Running TearDownSuite ---")
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

func (s *WorkItemTypeSuite) TestGetWorkItemType() {
	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	assert.NotNil(s.T(), wit)

	_, wit2 := test.ShowWorkitemtypeOK(s.T(), nil, nil, &s.typeCtrl, wit.Name)

	assert.NotNil(s.T(), wit2)
	assert.EqualValues(s.T(), wit, wit2)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	suite.Run(t, new(WorkItemTypeSuite))
}
