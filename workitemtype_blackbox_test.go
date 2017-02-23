package main_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemTypeSuite struct {
	gormsupport.DBTestSuite
	typeCtrl     *WorkitemtypeController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController

	svcSpace *goa.Service
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeSuite{
		DBTestSuite: gormsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeSuite) SetupTest() {
	svc := goa.New("workItemTypeSuite-Service")
	assert.NotNil(s.T(), svc)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.DB))
	assert.NotNil(s.T(), s.typeCtrl)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.linkCatCtrl)

	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svcSpace = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(DB))
	require.NotNil(s.T(), s.spaceCtrl)
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *workItemTypeSuite) createWorkItemTypeAnimal() (http.ResponseWriter, *app.WorkItemType) {

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
	payload := app.CreateWorkItemTypePayload{
		Fields: map[string]*app.FieldDefinition{
			"animal_type": &typeFieldDef,
			"color":       &colorFieldDef,
		},
		Name: "animal",
	}

	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *workItemTypeSuite) createWorkItemTypePerson() (http.ResponseWriter, *app.WorkItemType) {

	// Create the type for the "color" field
	nameFieldDef := app.FieldDefinition{
		Required: true,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	payload := app.CreateWorkItemTypePayload{
		Fields: map[string]*app.FieldDefinition{
			"name": &nameFieldDef,
		},
		Name: "person",
	}

	return test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// TestCreateWorkItemType tests if we can create two work item types: "animal" and "person"
func (s *workItemTypeSuite) TestCreateWorkItemType() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	_, wit := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), wit)
	require.Equal(s.T(), "animal", wit.Name)

	_, wit = s.createWorkItemTypePerson()
	require.NotNil(s.T(), wit)
	require.Equal(s.T(), "person", wit.Name)
}

// TestShowWorkItemType tests if we can fetch the work item type "animal".
func (s *workItemTypeSuite) TestShowWorkItemType() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), wit)

	_, wit2 := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, wit.Name)

	require.NotNil(s.T(), wit2)
	require.EqualValues(s.T(), wit, wit2)
}

// TestListWorkItemType tests if we can find the work item types
// "person" and "animal" in the list of work item types
func (s *workItemTypeSuite) TestListWorkItemType() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create the work item type first and try to read it back in
	_, witAnimal := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), witAnimal)
	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)

	// Fetch a single work item type
	// Paging in the format <start>,<limit>"
	page := "0,-1"
	_, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, &page)

	require.NotNil(s.T(), witCollection)
	require.Nil(s.T(), witCollection.Validate())

	// Check the number of found work item types
	require.Condition(s.T(), func() bool {
		return (len(witCollection) >= 2)
	}, "At least two work item types must exist (animal and person), but only %d exist.", len(witCollection))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(witCollection) && toBeFound > 0; i++ {
		if witCollection[i].Name == "person" || witCollection[i].Name == "animal" {
			s.T().Log("Found work item type in collection: ", witCollection[i].Name)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item types (animal and person) where found.")
}

// TestListSourceAndTargetLinkTypes tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypes() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	// Create the work item type first and try to read it back in
	_, witAnimal := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), witAnimal)
	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)

	// Create work item link category
	linkCatPayload := CreateWorkItemLinkCategory("some-link-category")
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), nil, nil, s.linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)

	// Create work item link space
	spacePayload := CreateSpacePayload("some-link-space", "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svcSpace.Context, s.svcSpace, s.spaceCtrl, spacePayload)
	require.NotNil(s.T(), space)

	// Create work item link type
	animalLinksToBugStr := "animal-links-to-bug"
	linkTypePayload := CreateWorkItemLinkType(animalLinksToBugStr, "animal", workitem.SystemBug, *linkCat.Data.ID, space.Data.ID.String())
	_, linkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Create another work item link type
	bugLinksToAnimalStr := "bug-links-to-animal"
	linkTypePayload = CreateWorkItemLinkType(bugLinksToAnimalStr, workitem.SystemBug, "animal", *linkCat.Data.ID, space.Data.ID.String())
	_, linkType = test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Fetch source link types
	_, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, "animal")
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// Check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	require.Equal(s.T(), animalLinksToBugStr, *wiltCollection.Data[0].Attributes.Name)

	// Fetch target link types
	_, wiltCollection = test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, "animal")
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	// Check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	require.Equal(s.T(), bugLinksToAnimalStr, *wiltCollection.Data[0].Attributes.Name)
}

// TestListSourceAndTargetLinkTypesEmpty tests that no link type is returned for
// WITs that don't have link types associated to them
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesEmpty() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)

	_, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, "person")
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)

	_, wiltCollection = test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, "person")
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)
}

// TestListSourceAndTargetLinkTypesNotFound tests that a NotFound error is
// returned when you query a non existing WIT.
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesNotFound() {
	defer cleaner.DeleteCreatedEntities(s.DB)()

	_, jerrors := test.ListSourceLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, "not-existing-WIT")
	require.NotNil(s.T(), jerrors)

	_, jerrors = test.ListTargetLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, "not-existing-WIT")
	require.NotNil(s.T(), jerrors)
}

func getWorkItemTypeTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.Nil(t, err)

	createWITPayloadString := bytes.NewBuffer([]byte(`{"fields": {"system.administrator": {"Required": true,"Type": {"Kind": "string"}}},"name": "Epic"}`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItemTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWITPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWITPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWITPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWITPayloadString,
			jwtToken:           "",
		},
		// Try fetching a random work Item Type
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItemTypes + "/someRandomTestWIT8712",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		}, {
			method:             http.MethodGet,
			url:                fmt.Sprintf(endpointWorkItemTypesSourceLinkTypes, "someNotExistingWIT"),
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		}, {
			method:             http.MethodGet,
			url:                fmt.Sprintf(endpointWorkItemTypesTargetLinkTypes, "someNotExistingWIT"),
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeWorkItemTypeCreate(t *testing.T) {
	UnauthorizeCreateUpdateDeleteTest(t, getWorkItemTypeTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWIT-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemtypeController(service, gormapplication.NewGormDB(DB))
		app.MountWorkitemtypeController(service, controller)
		return nil
	})
}
