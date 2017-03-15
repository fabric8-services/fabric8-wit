package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The workItemLinkTypeSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemLinkTypeSuite struct {
	suite.Suite
	db           *gorm.DB
	linkTypeCtrl *WorkItemLinkTypeController
	spaceCtrl    *SpaceController
	linkCatCtrl  *WorkItemLinkCategoryController
	typeCtrl     *WorkitemtypeController
	svc          *goa.Service
	spaceName    string
	spaceID      *uuid.UUID
	categoryName string
	linkTypeName string
	linkName     string
}

var wiltConfiguration *config.ConfigurationData

func init() {
	var err error
	wiltConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemLinkTypeSuite) SetupSuite() {
	var err error
	wiltConfiguration, err = config.GetConfigurationData()
	require.Nil(s.T(), err)
	s.db, err = gorm.Open("postgres", wiltConfiguration.GetPostgresConfigString())
	require.Nil(s.T(), err)
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	err = models.Transactional(s.db, func(tx *gorm.DB) error {
		return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
	})
	require.Nil(s.T(), err)
	svc := goa.New("workItemLinkTypeSuite-Service")
	require.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.linkCatCtrl)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.typeCtrl)
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.spaceCtrl)
	s.spaceName = "test-space" + uuid.NewV4().String()
	s.categoryName = "test-workitem-category" + uuid.NewV4().String()
	s.linkTypeName = "test-workitem-link-type" + uuid.NewV4().String()
	s.linkName = "test-workitem-link" + uuid.NewV4().String()
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *workItemLinkTypeSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *workItemLinkTypeSuite) cleanup() {
	db := s.db.Unscoped().Delete(&link.WorkItemLinkType{Name: s.linkTypeName})
	require.Nil(s.T(), db.Error)
	db = s.db.Unscoped().Delete(&link.WorkItemLinkType{Name: s.linkName})
	require.Nil(s.T(), db.Error)
	db = db.Unscoped().Delete(&link.WorkItemLinkCategory{Name: s.categoryName})
	require.Nil(s.T(), db.Error)
	if s.spaceID != nil {
		db = db.Unscoped().Delete(&space.Space{ID: *s.spaceID})
	}
	require.Nil(s.T(), db.Error)
	//db = db.Unscoped().Delete(&link.WorkItemType{Name: "foo.bug"})

}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item link types that we will create already exist.
func (s *workItemLinkTypeSuite) SetupTest() {
	s.cleanup()
}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemLinkTypeSuite) TearDownTest() {
	s.cleanup()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createDemoType creates a demo work item link type of type "name"
func (s *workItemLinkTypeSuite) createDemoLinkType(name string) *app.CreateWorkItemLinkTypePayload {
	//   1. Create a space
	createSpacePayload := CreateSpacePayload(s.spaceName, "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, createSpacePayload)
	s.spaceID = space.Data.ID

	//	 2. Create at least one work item type
	workItemTypePayload := CreateWorkItemType(uuid.NewV4(), *space.Data.ID)
	_, workItemType := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, &workItemTypePayload)
	require.NotNil(s.T(), workItemType)

	//   3. Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory(s.categoryName)
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)

	// 4. Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType(name, *workItemType.Data.ID, *workItemType.Data.ID, *workItemLinkCategory.Data.ID, *space.Data.ID)
	return createLinkTypePayload
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLinkType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkTypeSuite))
}

// TestCreateWorkItemLinkType tests if we can create the s.linkTypeName work item link type
func (s *workItemLinkTypeSuite) TestCreateAndDeleteWorkItemLinkType() {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload)
	require.NotNil(s.T(), workItemLinkType)

	// Check that the link category is included in the response in the "included" array
	require.Len(s.T(), workItemLinkType.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := workItemLinkType.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")

	// Check that the link category is included in the response in the "included" array
	spaceData, ok := workItemLinkType.Included[1].(*app.Space)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.spaceName, *spaceData.Attributes.Name, "The work item link type's space should have the name 'test-space'.")

	_ = test.DeleteWorkItemLinkTypeOK(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *workItemLinkType.Data.ID)
}

//func (s *workItemLinkTypeSuite) TestCreateWorkItemLinkTypeBadRequest() {
//	createPayload := s.createDemoLinkType("") // empty name causes bad request
//	_, _ = test.CreateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//}

//func (s *workItemLinkTypeSuite) TestCreateWorkItemLinkTypeBadRequestDueToEmptyTopology() {
//	createPayload := s.createDemoLinkType(s.linkTypeName)
//	emptyTopology := ""
//	createPayload.Data.Attributes.Topology = &emptyTopology
//	_, _ = test.CreateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//}

//func (s *workItemLinkTypeSuite) TestCreateWorkItemLinkTypeBadRequestDueToWrongTopology() {
//	createPayload := s.createDemoLinkType(s.linkTypeName)
//	wrongTopology := "wrongtopology"
//	createPayload.Data.Attributes.Topology = &wrongTopology
//	_, _ = test.CreateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
//}

func (s *workItemLinkTypeSuite) TestDeleteWorkItemLinkTypeNotFound() {
	test.DeleteWorkItemLinkTypeNotFound(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, uuid.FromStringOrNil("1e9a8b53-73a6-40de-b028-5177add79ffa"))
}

func (s *workItemLinkTypeSuite) TestUpdateWorkItemLinkTypeNotFound() {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	notExistingId := uuid.FromStringOrNil("46bbce9c-8219-4364-a450-dfd1b501654e") // This ID does not exist
	createPayload.Data.ID = &notExistingId
	// Wrap data portion in an update payload instead of a create payload
	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
		Data: createPayload.Data,
	}
	test.UpdateWorkItemLinkTypeNotFound(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
}

// func (s *workItemLinkTypeSuite) TestUpdateWorkItemLinkTypeBadRequestDueToBadID() {
// 	createPayload := s.createDemoLinkType(s.linkTypeName)
// 	notExistingId := "something that is not a UUID" // This ID does not exist
// 	createPayload.Data.ID = &notExistingId
// 	// Wrap data portion in an update payload instead of a create payload
// 	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
// 		Data: createPayload.Data,
// 	}
// 	test.UpdateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
// }

func (s *workItemLinkTypeSuite) TestUpdateWorkItemLinkTypeOK() {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload)
	require.NotNil(s.T(), workItemLinkType)
	// Specify new description for link type that we just created
	// Wrap data portion in an update payload instead of a create payload
	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
		Data: workItemLinkType.Data,
	}
	newDescription := "Lalala this is a new description for the work item type"
	updateLinkTypePayload.Data.Attributes.Description = &newDescription
	_, lt := test.UpdateWorkItemLinkTypeOK(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
	require.NotNil(s.T(), lt.Data)
	require.NotNil(s.T(), lt.Data.Attributes)
	require.NotNil(s.T(), lt.Data.Attributes.Description)
	require.Equal(s.T(), newDescription, *lt.Data.Attributes.Description)
	// Check that the link categories are included in the response in the "included" array
	require.Len(s.T(), lt.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := lt.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")

	// Check that the link spaces are included in the response in the "included" array
	spaceData, ok := lt.Included[1].(*app.Space)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.spaceName, *spaceData.Attributes.Name, "The work item link type's space should have the name 'test-space'.")
}

// func (s *workItemLinkTypeSuite) TestUpdateWorkItemLinkTypeBadRequest() {
// 	createPayload := s.createDemoLinkType(s.linkTypeName)
// 	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
// 		Data: createPayload.Data,
// 	}
// 	updateLinkTypePayload.Data.Type = "This should be workitemlinktypes" // Causes bad request
// 	test.UpdateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
// }

// TestShowWorkItemLinkTypeOK tests if we can fetch the "system" work item link type
func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeOK() {
	// Create the work item link type first and try to read it back in
	createPayload := s.createDemoLinkType(s.linkTypeName)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload)
	require.NotNil(s.T(), workItemLinkType)
	_, readIn := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, *workItemLinkType.Data.ID)
	require.NotNil(s.T(), readIn)
	// Convert to model space and use equal function
	expected := link.WorkItemLinkType{}
	actual := link.WorkItemLinkType{}
	require.Nil(s.T(), link.ConvertLinkTypeToModel(*workItemLinkType, &expected))
	require.Nil(s.T(), link.ConvertLinkTypeToModel(*readIn, &actual))
	require.True(s.T(), expected.Equal(actual))
	// Check that the link category is included in the response in the "included" array
	require.Len(s.T(), readIn.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := readIn.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")

	// Check that the link space is included in the response in the "included" array
	spaceData, ok := readIn.Included[1].(*app.Space)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.spaceName, *spaceData.Attributes.Name, "The work item link type's space should have the name 'test-space'.")

	require.NotNil(s.T(), readIn.Data.Links, "The link type MUST include a self link")
	require.NotEmpty(s.T(), readIn.Data.Links.Self, "The link type MUST include a self link that's not empty")
}

// TestShowWorkItemLinkTypeNotFound tests if we can fetch a non existing work item link type
func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeNotFound() {
	test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19"))
}

// TestListWorkItemLinkTypeOK tests if we can find the work item link types
// s.linkTypeName and s.linkName in the list of work item link types
func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeOK() {
	bugBlockerPayload := s.createDemoLinkType(s.linkTypeName)
	_, bugBlockerType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, bugBlockerPayload)
	require.NotNil(s.T(), bugBlockerType)

	workItemTypePayload := CreateWorkItemType(uuid.NewV4(), *s.spaceID)
	_, workItemType := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, &workItemTypePayload)
	require.NotNil(s.T(), workItemType)

	relatedPayload := CreateWorkItemLinkType(s.linkName, *workItemType.Data.ID, *workItemType.Data.ID, bugBlockerType.Data.Relationships.LinkCategory.Data.ID, *bugBlockerType.Data.Relationships.Space.Data.ID)
	_, relatedType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, relatedPayload)
	require.NotNil(s.T(), relatedType)

	// Fetch a single work item link type
	_, linkTypeCollection := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl)
	require.NotNil(s.T(), linkTypeCollection)
	require.Nil(s.T(), linkTypeCollection.Validate())
	// Check the number of found work item link types
	require.NotNil(s.T(), linkTypeCollection.Data)
	require.Condition(s.T(), func() bool {
		return (len(linkTypeCollection.Data) >= 2)
	}, "At least two work item link types must exist (bug-blocker and related), but only %d exist.", len(linkTypeCollection.Data))
	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkTypeCollection.Data) && toBeFound > 0; i++ {
		if *linkTypeCollection.Data[i].Attributes.Name == s.linkTypeName || *linkTypeCollection.Data[i].Attributes.Name == s.linkName {
			s.T().Log("Found work item link type in collection: ", *linkTypeCollection.Data[i].Attributes.Name)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item link types (bug-blocker and related) where found.")

	// Check that the link categories are included in the response in the "included" array
	require.Len(s.T(), linkTypeCollection.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := linkTypeCollection.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")

	// Check that the link spaces are included in the response in the "included" array
	spaceData, ok := linkTypeCollection.Included[1].(*app.Space)
	require.True(s.T(), ok)
	require.Equal(s.T(), s.spaceName, *spaceData.Attributes.Name, "The work item link type's category should have the name 'test-space'.")

}

func getWorkItemLinkTypeTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((wiltConfiguration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	if err != nil {
		t.Fatal("Could not parse different private key ", err)
	}

	createWorkItemLinkTypePayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"type": "workitemlinktypes",
				"id": "0270e113-7790-477f-9371-97c37d734d5d",
				"attributes": {
					"name": "sample",
					"description": "A sample work item link type",
					"version": 0,
					"forward_name": "forward string name",
					"reverse_name": "reverse string name"
				},
				"relationships": {
					"link_category": {"data": {"type":"workitemlinkcategories", "id": "a75ea296-6378-4578-8573-90f11b8efb00"}},
					"space": {"data": {"type":"spaces", "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"}, "links":{"self": "http://localhost:8080/api/spaces/6ba7b810-9dad-11d1-80b4-00c04fd430c8"}},
					"source_type": {"data": {"type":"workitemtypes", "id": "e7492516-4d7d-4962-a820-75bea73a322e"}},
					"target_type": {"data": {"type":"workitemtypes", "id": "e7492516-4d7d-4962-a820-75bea73a322e"}}
				}
			}
		}
		`))
	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItemLinkTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkTypes,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkTypePayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkTypes + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work item link type
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItemLinkTypes + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkTypeSuite) TestUnauthorizeWorkItemLinkTypeCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemLinkTypeTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLinkType-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkTypeController(service, gormapplication.NewGormDB(s.db))
		app.MountWorkItemLinkTypeController(service, controller)
		return nil
	})
}

func TestNewWorkItemLinkTypeControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypeController(nil, nil)
	})
}
