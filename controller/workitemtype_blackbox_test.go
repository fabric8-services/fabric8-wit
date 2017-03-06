package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var witbConfiguration *config.ConfigurationData

func init() {
	var err error
	witbConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemTypeSuite struct {
	gormsupport.DBTestSuite
	clean        func()
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
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
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

func (s *workItemTypeSuite) TearDownTest() {
	s.clean()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

var (
	animalID = uuid.FromStringOrNil("729431f2-bca4-4062-9087-c751807b569f")
	personID = uuid.FromStringOrNil("22a1e4f1-7e9d-4ce8-ac87-fe7c79356b16")
)

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *workItemTypeSuite) createWorkItemTypeAnimal() (http.ResponseWriter, *app.WorkItemTypeSingle) {

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
	desc := "Description for 'animal'"
	id := animalID
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			Type: "workitemtypes",
			ID:   &id,
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "animal",
				Description: &desc,
				Icon:        "fa-hand-lizard-o",
				Fields: map[string]*app.FieldDefinition{
					"animal_type": &typeFieldDef,
					"color":       &colorFieldDef,
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	responseWriter, wi := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
	require.NotNil(s.T(), wi)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.True(s.T(), uuid.Equal(animalID, *wi.Data.ID))
	return responseWriter, wi
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *workItemTypeSuite) createWorkItemTypePerson() (http.ResponseWriter, *app.WorkItemTypeSingle) {
	// Create the type for the "color" field
	nameFieldDef := app.FieldDefinition{
		Required: true,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	desc := "Description for 'person'"
	id := personID
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "person",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"name": &nameFieldDef,
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	responseWriter, wi := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, &payload)
	require.NotNil(s.T(), wi)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.True(s.T(), uuid.Equal(personID, *wi.Data.ID))
	return responseWriter, wi
}

func CreateWorkItemType(id uuid.UUID, spaceID uuid.UUID) app.CreateWorkitemtypePayload {
	// Create the type for the "color" field
	nameFieldDef := app.FieldDefinition{
		Required: false,
		Type: &app.FieldType{
			Kind: "string",
		},
	}

	// Use the goa generated code to create a work item type
	desc := "Description for 'person'"
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(spaceID.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "person",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"test": &nameFieldDef,
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: space.NewSpaceRelation(spaceID, spaceSelfURL),
			},
		},
	}

	return payload
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// TestCreateWorkItemType tests if we can create two work item types: "animal" and "person"
func (s *workItemTypeSuite) TestCreateWorkItemType() {
	_, _ = s.createWorkItemTypeAnimal()
	_, _ = s.createWorkItemTypePerson()
}

// TestShowWorkItemType tests if we can fetch the work item type "animal".
func (s *workItemTypeSuite) TestShowWorkItemType() {
	// Create the work item type first and try to read it back in
	_, wit := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	_, wit2 := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, *wit.Data.ID)

	require.NotNil(s.T(), wit2)
	require.EqualValues(s.T(), wit, wit2)
}

// TestListWorkItemType tests if we can find the work item types
// "person" and "animal" in the list of work item types
func (s *workItemTypeSuite) TestListWorkItemType() {
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
	assert.Condition(s.T(), func() bool {
		return (len(witCollection.Data) >= 2)
	}, "At least two work item types must exist (animal and person), but only %d exist.", len(witCollection.Data))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(witCollection.Data) && toBeFound > 0; i++ {
		require.NotNil(s.T(), witCollection.Data[i])
		require.NotNil(s.T(), witCollection.Data[i].ID)
		id := *witCollection.Data[i].ID
		name := witCollection.Data[i].Attributes.Name
		if uuid.Equal(id, personID) || uuid.Equal(id, animalID) {
			s.T().Log("Found work item type in collection: ID=", id, " name=", name)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item types (animal and person) where found.")
}

// TestListSourceAndTargetLinkTypes tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypes() {
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

	// Create work item link type
	animalLinksToBugStr := "animal-links-to-bug"
	linkTypePayload := CreateWorkItemLinkType(animalLinksToBugStr, animalID, workitem.SystemBug, *linkCat.Data.ID, *space.Data.ID)
	_, linkType := test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Create another work item link type
	bugLinksToAnimalStr := "bug-links-to-animal"
	linkTypePayload = CreateWorkItemLinkType(bugLinksToAnimalStr, workitem.SystemBug, animalID, *linkCat.Data.ID, *space.Data.ID)
	_, linkType = test.CreateWorkItemLinkTypeCreated(s.T(), nil, nil, s.linkTypeCtrl, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Fetch source link types
	_, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, animalID)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// Check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	require.Equal(s.T(), animalLinksToBugStr, *wiltCollection.Data[0].Attributes.Name)

	// Fetch target link types
	_, wiltCollection = test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, animalID)
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	// Check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	require.Equal(s.T(), bugLinksToAnimalStr, *wiltCollection.Data[0].Attributes.Name)
}

// TestListSourceAndTargetLinkTypesEmpty tests that no link type is returned for
// WITs that don't have link types associated to them
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesEmpty() {
	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)

	_, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, personID)
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)

	_, wiltCollection = test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, personID)
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)
}

// TestListSourceAndTargetLinkTypesNotFound tests that a NotFound error is
// returned when you query a non existing WIT.
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesNotFound() {
	_, jerrors := test.ListSourceLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, uuid.Nil)
	require.NotNil(s.T(), jerrors)

	_, jerrors = test.ListTargetLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, uuid.Nil)
	require.NotNil(s.T(), jerrors)
}

func getWorkItemTypeTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((witbConfiguration.GetTokenPrivateKey()))
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
			url:                endpointWorkItemTypes + "/2e889d4e-49a9-463b-8cd4-6a3a95155103",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		}, {
			method:             http.MethodGet,
			url:                fmt.Sprintf(endpointWorkItemTypesSourceLinkTypes, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		}, {
			method:             http.MethodGet,
			url:                fmt.Sprintf(endpointWorkItemTypesTargetLinkTypes, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
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
