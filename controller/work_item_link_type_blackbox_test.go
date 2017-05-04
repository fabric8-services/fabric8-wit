package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem/link"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
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
	gormtestsupport.DBTestSuite
	linkTypeCtrl            *WorkItemLinkTypeController
	linkTypeCombinationCtrl *WorkItemLinkTypeCombinationController
	spaceCtrl               *SpaceController
	linkCatCtrl             *WorkItemLinkCategoryController
	typeCtrl                *WorkitemtypeController
	svc                     *goa.Service
	spaceName               string
	spaceID                 *uuid.UUID
	categoryName            string
	linkTypeName            string
	linkName                string
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemLinkTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)

	svc := goa.New("workItemLinkTypeSuite-Service")
	require.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.linkTypeCombinationCtrl = NewWorkItemLinkTypeCombinationController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.linkTypeCombinationCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(s.DB))
	require.NotNil(s.T(), s.linkCatCtrl)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.typeCtrl)

	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.spaceName = "test-space" + uuid.NewV4().String()
	s.categoryName = "test-workitem-category" + uuid.NewV4().String()
	s.linkTypeName = "test-workitem-link-type" + uuid.NewV4().String()
	s.linkName = "test-workitem-link" + uuid.NewV4().String()
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *workItemLinkTypeSuite) TearDownSuite() {
	if s.DB != nil {
		s.DB.Close()
	}
}

// cleanup removes all DB entries that will be created or have been created
// with this test suite. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *workItemLinkTypeSuite) cleanup() {
	db := s.DB.Unscoped().Delete(&link.WorkItemLinkType{Name: s.linkTypeName})
	require.Nil(s.T(), db.Error)
	db = s.DB.Unscoped().Delete(&link.WorkItemLinkType{Name: s.linkName})
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
	svc := goa.New("workItemLinkTypeSuite-Service")
	require.NotNil(s.T(), svc)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.linkTypeCtrl)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(svc, gormapplication.NewGormDB(s.DB))
	require.NotNil(s.T(), s.linkCatCtrl)
	s.typeCtrl = NewWorkitemtypeController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	require.NotNil(s.T(), s.typeCtrl)
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.spaceCtrl = NewSpaceController(svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.spaceName = testsupport.CreateRandomValidTestName("test-space")
	s.categoryName = "test-workitem-category" + uuid.NewV4().String()
	s.linkTypeName = "test-workitem-link-type" + uuid.NewV4().String()
	s.linkName = "test-workitem-link" + uuid.NewV4().String()
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
	_, workItemType := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, s.spaceID.String(), &workItemTypePayload)
	require.NotNil(s.T(), workItemType)

	//   3. Create a work item link category
	createLinkCategoryPayload := CreateWorkItemLinkCategory(s.categoryName)
	_, workItemLinkCategory := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, createLinkCategoryPayload)
	require.NotNil(s.T(), workItemLinkCategory)

	// 4. Create work item link type payload
	createLinkTypePayload := CreateWorkItemLinkType(name, *workItemLinkCategory.Data.ID, *space.Data.ID)

	return createLinkTypePayload
	//  *workItemType.Data.ID, *workItemType.Data.ID
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

func TestNewWorkItemLinkTypeControllerDBNull(t *testing.T) {
	require.Panics(t, func() {
		NewWorkItemLinkTypeController(nil, nil, nil)
	})
}

// TestCreateWorkItemLinkType tests if we can create the s.linkTypeName work item link type
func (s *workItemLinkTypeSuite) TestCreateAndDeleteWorkItemLinkType() {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload.Data.Relationships.Space.Data.ID.String(), createPayload)
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

	_ = test.DeleteWorkItemLinkTypeOK(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, workItemLinkType.Data.Relationships.Space.Data.ID.String(), *workItemLinkType.Data.ID)
}

// func (s *workItemLinkTypeSuite) TestCreateWorkItemLinkTypeBadRequest() {
// 	createPayload := s.createDemoLinkType("") // empty name causes bad request
// 	createPayload.Validate()
// 	_, _ = test.CreateWorkItemLinkTypeBadRequest(s.T(), nil, nil, s.linkTypeCtrl, createPayload)
// }

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
	test.DeleteWorkItemLinkTypeNotFound(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, space.SystemSpace.String(), uuid.FromStringOrNil("1e9a8b53-73a6-40de-b028-5177add79ffa"))
}

func (s *workItemLinkTypeSuite) TestUpdateWorkItemLinkTypeNotFound() {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	notExistingId := uuid.FromStringOrNil("46bbce9c-8219-4364-a450-dfd1b501654e") // This ID does not exist
	createPayload.Data.ID = &notExistingId
	// Wrap data portion in an update payload instead of a create payload
	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
		Data: createPayload.Data,
	}
	test.UpdateWorkItemLinkTypeNotFound(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, updateLinkTypePayload.Data.Relationships.Space.Data.ID.String(), *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
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
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload.Data.Relationships.Space.Data.ID.String(), createPayload)
	require.NotNil(s.T(), workItemLinkType)
	// Specify new description for link type that we just created
	// Wrap data portion in an update payload instead of a create payload
	updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
		Data: workItemLinkType.Data,
	}
	newDescription := "Lalala this is a new description for the work item type"
	updateLinkTypePayload.Data.Attributes.Description = &newDescription
	_, lt := test.UpdateWorkItemLinkTypeOK(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, updateLinkTypePayload.Data.Relationships.Space.Data.ID.String(), *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
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

func (s *workItemLinkTypeSuite) createWorkItemLinkType() *app.WorkItemLinkTypeSingle {
	createPayload := s.createDemoLinkType(s.linkTypeName)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, createPayload.Data.Relationships.Space.Data.ID.String(), createPayload)
	require.NotNil(s.T(), workItemLinkType)
	return workItemLinkType
}

func assertWorkItemLinkType(t *testing.T, expected *app.WorkItemLinkTypeSingle, spaceName, categoryName string, actual *app.WorkItemLinkTypeSingle) {
	require.NotNil(t, actual)
	expectedModel, err := ConvertWorkItemLinkTypeToModel(*expected)
	require.Nil(t, err)
	actualModel, err := ConvertWorkItemLinkTypeToModel(*actual)
	require.Nil(t, err)
	require.Equal(t, expectedModel.ID, actualModel.ID)
	// Check that the link category is included in the response in the "included" array
	require.Len(t, actual.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := actual.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(t, ok)
	require.Equal(t, categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")

	// Check that the link space is included in the response in the "included" array
	spaceData, ok := actual.Included[1].(*app.Space)
	require.True(t, ok)
	require.Equal(t, spaceName, *spaceData.Attributes.Name, "The work item link type's space should have the name 'test-space'.")

	require.NotNil(t, actual.Data.Links, "The link type MUST include a self link")
	require.NotEmpty(t, actual.Data.Links.Self, "The link type MUST include a self link that's not empty")
}

// TestShowWorkItemLinkTypeOK tests if we can fetch the "system" work item link type
func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeOK() {
	// given
	createdWorkItemLinkType := s.createWorkItemLinkType()
	// when
	res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), *createdWorkItemLinkType.Data.ID, nil, nil)
	// then
	assertWorkItemLinkType(s.T(), createdWorkItemLinkType, s.spaceName, s.categoryName, readWorkItemLinkType)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeOKUsingExpiredIfModifiedSinceHeader() {
	// given
	createdWorkItemLinkType := s.createWorkItemLinkType()
	// when
	ifModifiedSinceHeader := app.ToHTTPTime(createdWorkItemLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), *createdWorkItemLinkType.Data.ID, &ifModifiedSinceHeader, nil)
	// then
	assertWorkItemLinkType(s.T(), createdWorkItemLinkType, s.spaceName, s.categoryName, readWorkItemLinkType)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeOKUsingExpiredIfNoneMatchHeader() {
	// given
	createdWorkItemLinkType := s.createWorkItemLinkType()
	// when
	ifNoneMatch := "foo"
	spaceID := createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String()
	res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, spaceID, *createdWorkItemLinkType.Data.ID, nil, &ifNoneMatch)
	// then
	assertWorkItemLinkType(s.T(), createdWorkItemLinkType, s.spaceName, s.categoryName, readWorkItemLinkType)
	assertResponseHeaders(s.T(), res)
}
func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeNotModifiedUsingIfModifiedSinceHeader() {
	// given
	createdWorkItemLinkType := s.createWorkItemLinkType()
	// when
	ifModifiedSinceHeader := app.ToHTTPTime(*createdWorkItemLinkType.Data.Attributes.UpdatedAt)
	spaceID := createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String()
	res := test.ShowWorkItemLinkTypeNotModified(s.T(), nil, nil, s.linkTypeCtrl, spaceID, *createdWorkItemLinkType.Data.ID, &ifModifiedSinceHeader, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeNotModifiedUsingIfNoneMatchHeader() {
	// given
	createdWorkItemLinkType := s.createWorkItemLinkType()
	// when
	createdWorkItemLinkTypeModel, err := ConvertWorkItemLinkTypeToModel(*createdWorkItemLinkType)
	require.Nil(s.T(), err)
	ifNoneMatch := app.GenerateEntityTag(createdWorkItemLinkTypeModel)
	res := test.ShowWorkItemLinkTypeNotModified(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), *createdWorkItemLinkType.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

// TestShowWorkItemLinkTypeNotFound tests if we can fetch a non existing work item link type
func (s *workItemLinkTypeSuite) TestShowWorkItemLinkTypeNotFound() {
	wiltID := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19")
	test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, space.SystemSpace.String(), wiltID, nil, nil)
}
func (s *workItemLinkTypeSuite) createWorkItemLinkTypes() (*app.WorkItemTypeSingle, *app.WorkItemLinkTypeSingle) {
	bugBlockerPayload := s.createDemoLinkType(s.linkTypeName)
	_, bugBlockerType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, bugBlockerPayload.Data.Relationships.Space.Data.ID.String(), bugBlockerPayload)
	require.NotNil(s.T(), bugBlockerType)

	spaceID := *bugBlockerType.Data.Relationships.Space.Data.ID
	categoryID := bugBlockerType.Data.Relationships.LinkCategory.Data.ID

	workItemTypePayload := CreateWorkItemType(uuid.NewV4(), *s.spaceID)
	_, workItemType := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, spaceID.String(), &workItemTypePayload)
	require.NotNil(s.T(), workItemType)

	relatedPayload := CreateWorkItemLinkType(s.linkName, categoryID, spaceID)
	_, relatedType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, spaceID.String(), relatedPayload)
	require.NotNil(s.T(), relatedType)

	wiltcPayload, err := CreateWorkItemLinkTypeCombination(spaceID, *relatedType.Data.ID, *workItemType.Data.ID, *workItemType.Data.ID)
	require.Nil(s.T(), err)
	_, wiltcCreated := test.CreateWorkItemLinkTypeCombinationCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCombinationCtrl, spaceID.String(), wiltcPayload)
	require.NotNil(s.T(), wiltcCreated)

	return workItemType, relatedType

}

func assertWorkItemLinkTypes(t *testing.T, spaceName, categoryName, expectedLinkTypeName, expectedLinkName string, linkTypes *app.WorkItemLinkTypeList) {
	require.NotNil(t, linkTypes)
	require.Nil(t, linkTypes.Validate())
	// Check the number of found work item link types
	require.NotNil(t, linkTypes.Data)
	require.Condition(t, func() bool {
		return (len(linkTypes.Data) >= 2)
	}, "At least two work item link types must exist (bug-blocker and related), but only %d exist.", len(linkTypes.Data))
	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkTypes.Data) && toBeFound > 0; i++ {
		if *linkTypes.Data[i].Attributes.Name == expectedLinkTypeName || *linkTypes.Data[i].Attributes.Name == expectedLinkName {
			t.Log("Found work item link type in collection: ", *linkTypes.Data[i].Attributes.Name)
			toBeFound--
		}
	}
	require.Exactly(t, 0, toBeFound, "Not all required work item link types (bug-blocker and related) where found.")
	// Check that the link categories are included in the response in the "included" array
	require.Len(t, linkTypes.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := linkTypes.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(t, ok)
	require.Equal(t, categoryName, *categoryData.Attributes.Name, "The work item link type's category should have the name 'test-user'.")
	// Check that the link spaces are included in the response in the "included" array
	spaceData, ok := linkTypes.Included[1].(*app.Space)
	require.True(t, ok)
	require.Equal(t, spaceName, *spaceData.Attributes.Name, "The work item link type's category should have the name 'test-space'.")
}

// TestListWorkItemLinkTypeOK tests if we can find the work item link types
// s.linkTypeName and s.linkName in the list of work item link types
func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeOK() {
	// given
	_, createdWorkItemLinkType := s.createWorkItemLinkTypes()
	// when fetching all work item link type in a give space
	res, linkTypes := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), nil, nil)
	// then
	assertWorkItemLinkTypes(s.T(), s.spaceName, s.categoryName, s.linkTypeName, s.linkName, linkTypes)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, createdWorkItemLinkType := s.createWorkItemLinkTypes()
	// when fetching all work item link type in a give space
	ifModifiedSinceHeader := app.ToHTTPTime(createdWorkItemLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, linkTypes := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), &ifModifiedSinceHeader, nil)
	// then
	assertWorkItemLinkTypes(s.T(), s.spaceName, s.categoryName, s.linkTypeName, s.linkName, linkTypes)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, createdWorkItemLinkType := s.createWorkItemLinkTypes()
	// when fetching all work item link type in a give space
	ifNoneMatch := "foo"
	res, linkTypes := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), nil, &ifNoneMatch)
	// then
	assertWorkItemLinkTypes(s.T(), s.spaceName, s.categoryName, s.linkTypeName, s.linkName, linkTypes)
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeNotModifiedUsingIfModifiedSinceHeader() {
	// given
	_, workItemLinkType := s.createWorkItemLinkTypes()
	// when fetching all work item link type in a give space
	ifModifiedSinceHeader := app.ToHTTPTime(*workItemLinkType.Data.Attributes.UpdatedAt)
	res := test.ListWorkItemLinkTypeNotModified(s.T(), nil, nil, s.linkTypeCtrl, workItemLinkType.Data.Relationships.Space.Data.ID.String(), &ifModifiedSinceHeader, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) TestListWorkItemLinkTypeNotModifiedUsingIfNoneMatchHeader() {
	// given
	_, createdWorkItemLinkType := s.createWorkItemLinkTypes()
	_, existingLinkTypes := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), nil, nil)
	// when fetching all work item link type in a give space
	createdWorkItemLinkTypeModels := make([]app.ConditionalResponseEntity, len(existingLinkTypes.Data))
	for i, linkTypeData := range existingLinkTypes.Data {
		createdWorkItemLinkTypeModel, err := ConvertWorkItemLinkTypeToModel(
			app.WorkItemLinkTypeSingle{
				Data: linkTypeData,
			},
		)
		require.Nil(s.T(), err)
		createdWorkItemLinkTypeModels[i] = *createdWorkItemLinkTypeModel
	}
	ifNoneMatch := app.GenerateEntitiesTag(createdWorkItemLinkTypeModels)
	res := test.ListWorkItemLinkTypeNotModified(s.T(), nil, nil, s.linkTypeCtrl, createdWorkItemLinkType.Data.Relationships.Space.Data.ID.String(), nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *workItemLinkTypeSuite) getWorkItemLinkTypeTestDataFunc() func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {

		privatekey, err := jwt.ParseRSAPrivateKeyFromPEM(s.Configuration.GetTokenPrivateKey())
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
					"space": {"data": {"type":"spaces", "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"}, "links":{"self": "http://localhost:8080/api/spaces/6ba7b810-9dad-11d1-80b4-00c04fd430c8"}}
				}
			}
		}
		`))
		return []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           "",
			},
			// Update Work Item API with different parameters
			{
				method:             http.MethodPatch,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPatch,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypePayloadString,
				jwtToken:           "",
			},
			// Delete Work Item API with different parameters
			{
				method:             http.MethodDelete,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodDelete,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           "",
			},
			// Try fetching a random work item link type
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemLinkTypes, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkTypeSuite) TestUnauthorizeWorkItemLinkTypeCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemLinkTypeTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLinkType-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkTypeController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemLinkTypeController(service, controller)
		return nil
	})
}
