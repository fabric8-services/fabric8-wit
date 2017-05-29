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
	"github.com/almighty/almighty-core/gormsupport/cleaner"
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

func TestSuiteWorkItemLinkType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkTypeSuite))
}

type workItemLinkTypeSuite struct {
	gormtestsupport.DBTestSuite

	clean                   func()
	linkTypeCtrl            *WorkItemLinkTypeController
	linkTypeCombinationCtrl *WorkItemLinkTypeCombinationController
	typeCtrl                *WorkitemtypeController
	svc                     *goa.Service

	spaceID   uuid.UUID
	wit1ID    uuid.UUID
	wit2ID    uuid.UUID
	linkCatID uuid.UUID
}

func (s *workItemLinkTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *workItemLinkTypeSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)

	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCombinationCtrl = NewWorkItemLinkTypeCombinationController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.typeCtrl = NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	// Create a few resources needed along the way in most tests

	// space
	spaceCtrl := NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	spacePayload := CreateSpacePayload(testsupport.CreateRandomValidTestName("space"), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, spaceCtrl, spacePayload)
	s.spaceID = *space.Data.ID

	// work item types (2x)
	wit1Payload := CreateWorkItemType(uuid.NewV4(), *space.Data.ID)
	wit2Payload := CreateWorkItemType(uuid.NewV4(), s.spaceID)
	_, wit1 := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, s.spaceID, &wit1Payload)
	_, wit2 := test.CreateWorkitemtypeCreated(s.T(), s.svc.Context, s.svc, s.typeCtrl, s.spaceID, &wit2Payload)
	s.wit2ID = *wit2.Data.ID
	s.wit1ID = *wit1.Data.ID

	// link category
	linkCatCtrl := NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
	linkCatPayload := CreateWorkItemLinkCategory(testsupport.CreateRandomValidTestName("link category"))
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, linkCatCtrl, linkCatPayload)
	s.linkCatID = *linkCat.Data.ID
}

func (s *workItemLinkTypeSuite) TearDownTest() {
	s.clean()
}

func (s *workItemLinkTypeSuite) TestCreateAndDelete() {
	// given
	createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("foo"), s.linkCatID, s.spaceID)
	// when
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, createPayload)
	// then
	require.NotNil(s.T(), workItemLinkType)

	s.T().Run("link category is included in the response in the \"included\" array", func(t *testing.T) {
		require.Len(t, workItemLinkType.Included, 2, "The work item link type should include it's work item link category and space.")
		categoryData, ok := workItemLinkType.Included[0].(*app.WorkItemLinkCategoryData)
		require.True(t, ok)
		require.Equal(t, s.linkCatID, *categoryData.ID)
	})

	s.T().Run("space is included in the response in the \"included\" array", func(t *testing.T) {
		spaceData, ok := workItemLinkType.Included[1].(*app.Space)
		require.True(t, ok)
		require.Equal(t, s.spaceID, *spaceData.ID)
	})

	s.T().Run("delete created link type", func(t *testing.T) {
		_ = test.DeleteWorkItemLinkTypeOK(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, *workItemLinkType.Data.ID)
	})
}

func (s *workItemLinkTypeSuite) TestValidateCreatePayload() {
	s.T().Run("all valid", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("empty name"), s.linkCatID, s.spaceID)
		// when
		valid := createPayload.Validate()
		// then
		require.Nil(t, valid)
	})
	s.T().Run("empty name", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("empty name"), s.linkCatID, s.spaceID)
		emptyName := ""
		createPayload.Data.Attributes.Name = &emptyName
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
	})
	s.T().Run("empty topology", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("empty topology"), s.linkCatID, s.spaceID)
		emptyTopology := ""
		createPayload.Data.Attributes.Topology = &emptyTopology
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
	})
	s.T().Run("wrong topology", func(t *testing.T) {
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("wrong topology"), s.linkCatID, s.spaceID)
		wrongTopology := "wrongtopology"
		createPayload.Data.Attributes.Topology = &wrongTopology
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
	})
}

func (s *workItemLinkTypeSuite) TestDelete() {
	s.T().Run("not found link type", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.NewV4()
		// then
		test.DeleteWorkItemLinkTypeNotFound(t, s.svc.Context, s.svc, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID)
	})
}

func (s *workItemLinkTypeSuite) TestUpdate() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("foo"), s.linkCatID, s.spaceID)
		_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, createPayload)
		require.NotNil(t, workItemLinkType)
		// Specify new description for link type that we just created
		// Wrap data portion in an update payload instead of a create payload
		updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
			Data: workItemLinkType.Data,
		}
		newDescription := "Lalala this is a new description for the work item type"
		updateLinkTypePayload.Data.Attributes.Description = &newDescription
		// when
		_, lt := test.UpdateWorkItemLinkTypeOK(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
		// then
		require.NotNil(t, lt.Data)
		require.NotNil(t, lt.Data.Attributes)
		require.NotNil(t, lt.Data.Attributes.Description)
		require.Equal(t, newDescription, *lt.Data.Attributes.Description)
		// Check that the link categories are included in the response in the "included" array
		require.Len(t, lt.Included, 2, "The work item link type should include it's work item link category and space.")
		categoryData, ok := lt.Included[0].(*app.WorkItemLinkCategoryData)
		require.True(t, ok)
		require.Equal(t, s.linkCatID, *categoryData.ID)
		// Check that the link spaces are included in the response in the "included" array
		spaceData, ok := lt.Included[1].(*app.Space)
		require.True(t, ok)
		require.Equal(t, s.spaceID, *spaceData.ID)
	})

	s.T().Run("not found link type", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("foo"), s.linkCatID, s.spaceID)
		notExistingId := uuid.NewV4()
		createPayload.Data.ID = &notExistingId
		// Wrap Data portion in an update payload instead of a create payload
		updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
			Data: createPayload.Data,
		}
		// then
		test.UpdateWorkItemLinkTypeNotFound(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, notExistingId, updateLinkTypePayload)
	})

	s.T().Run("conflict", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("foo"), s.linkCatID, s.spaceID)
		_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, createPayload)
		require.NotNil(t, workItemLinkType)
		// Specify new description for link type that we just created
		// Wrap data portion in an update payload instead of a create payload
		updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
			Data: workItemLinkType.Data,
		}
		newDescription := "Lalala this is a new description for the work item type"
		updateLinkTypePayload.Data.Attributes.Description = &newDescription
		version := 123456
		updateLinkTypePayload.Data.Attributes.Version = &version
		// when/then
		test.UpdateWorkItemLinkTypeConflict(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
	})
}

func (s *workItemLinkTypeSuite) TestShow() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, nil, nil)
		// then
		assertWorkItemLinkType(t, createdWorkItemLinkType, readWorkItemLinkType)
		assertResponseHeaders(t, res)
	})
	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(createdWorkItemLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, &ifModifiedSinceHeader, nil)
		// then
		assertWorkItemLinkType(t, createdWorkItemLinkType, readWorkItemLinkType)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired ExpiredIfNoneMatch header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		ifNoneMatch := "foo"
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, nil, &ifNoneMatch)
		// then
		assertWorkItemLinkType(t, createdWorkItemLinkType, readWorkItemLinkType)
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using expired IfModifiedSince header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(*createdWorkItemLinkType.Data.Attributes.UpdatedAt)
		res := test.ShowWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, &ifModifiedSinceHeader, nil)
		// then
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		createdWorkItemLinkTypeModel, err := ConvertWorkItemLinkTypeToModel(*createdWorkItemLinkType)
		require.Nil(t, err)
		ifNoneMatch := app.GenerateEntityTag(createdWorkItemLinkTypeModel)
		res := test.ShowWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(t, res)
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.NewV4()
		// then
		test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID, nil, nil)
	})
}

func (s *workItemLinkTypeSuite) TestList() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		type1 := s.createRandomWorkItemLinkType(t)
		type2 := s.createRandomWorkItemLinkType(t)
		// when
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, nil, nil)
		// then
		require.Nil(t, linkTypes.Validate(), "list of work item link types is not valid")
		s.requireMinNumberOfListElements(t, 2, linkTypes)
		s.requireIDsInList(t, linkTypes, *type1.Data.ID, *type2.Data.ID)
		s.requireIncluded(t, linkTypes, s.spaceID, s.linkCatID)
		assertResponseHeaders(t, res)
	})
	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		type1 := s.createRandomWorkItemLinkType(t)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(type1.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, &ifModifiedSinceHeader, nil)
		// then
		s.requireMinNumberOfListElements(t, 1, linkTypes)
		s.requireIDsInList(t, linkTypes, *type1.Data.ID)
		s.requireIncluded(t, linkTypes, s.spaceID, s.linkCatID)
		assertResponseHeaders(t, res)
	})
	s.T().Run("ok using expired IfNoneMatch header", func(t *testing.T) {
		// given
		type1 := s.createRandomWorkItemLinkType(t)
		// when
		ifNoneMatch := "foo"
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, nil, &ifNoneMatch)
		// then
		s.requireMinNumberOfListElements(t, 1, linkTypes)
		s.requireIDsInList(t, linkTypes, *type1.Data.ID)
		s.requireIncluded(t, linkTypes, s.spaceID, s.linkCatID)
		assertResponseHeaders(t, res)
	})
	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// given
		type1 := s.createRandomWorkItemLinkType(t)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(*type1.Data.Attributes.UpdatedAt)
		res := test.ListWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, s.spaceID, &ifModifiedSinceHeader, nil)
		// then
		assertResponseHeaders(t, res)
	})
	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// given
		_ = s.createRandomWorkItemLinkType(t)
		_, existingLinkTypes := test.ListWorkItemLinkTypeOK(s.T(), nil, nil, s.linkTypeCtrl, s.spaceID, nil, nil)
		// when
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
		res := test.ListWorkItemLinkTypeNotModified(s.T(), nil, nil, s.linkTypeCtrl, s.spaceID, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(s.T(), res)
	})
}

func (s *workItemLinkTypeSuite) TestListTypeCombinations() {
	s.T().Run("ok", func(t *testing.T) {
		// given
		type1 := s.createRandomWorkItemLinkType(t)
		_, wit1 := createRandomWorkItemType(t, s.typeCtrl, s.spaceID)
		_, wit2 := createRandomWorkItemType(t, s.typeCtrl, s.spaceID)
		combi := link.WorkItemLinkTypeCombination{
			SpaceID:      s.spaceID,
			LinkTypeID:   *type1.Data.ID,
			SourceTypeID: *wit1.Data.ID,
			TargetTypeID: *wit2.Data.ID,
		}
		_, _ = createWorkItemTypeCombination(t, gormapplication.NewGormDB(s.DB), s.linkTypeCombinationCtrl, combi)
		// when
		_, combinationList := test.ListTypeCombinationsWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *type1.Data.ID, nil, nil)
		// then
		require.NotNil(t, combinationList)
		require.Len(t, combinationList.Data, 1)
	})
	s.T().Run("not existing link type", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.NewV4()
		// when
		_, err := test.ListTypeCombinationsWorkItemLinkTypeNotFound(t, nil, nil, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID, nil, nil)
		// then
		require.NotNil(t, err)
	})

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

// createRandomWorkItemLinkType creates a random work item link type
func (s *workItemLinkTypeSuite) createRandomWorkItemLinkType(t *testing.T) *app.WorkItemLinkTypeSingle {
	createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("foo"), s.linkCatID, s.spaceID)
	_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, createPayload)
	require.NotNil(t, workItemLinkType)
	return workItemLinkType
}

// requireIncludedCheck checks that all given IDs are included in the given
// included list.
func (s *workItemLinkTypeSuite) requireIncluded(t *testing.T, list *app.WorkItemLinkTypeList, IDs ...uuid.UUID) {
	for _, id := range IDs {
		found := false
		for _, included := range list.Included {
			switch data := included.(type) {
			case *app.WorkItemLinkCategoryData:
				if *data.ID == id {
					found = true
				}
			case *app.Space:
				if *data.ID == id {
					found = true
				}
			}
		}
		require.True(t, found, "failed to find element")
	}
}

// requireMinNumberOfListElements checks that the given list has at least minNum
// elements.
func (s *workItemLinkTypeSuite) requireMinNumberOfListElements(t *testing.T, minNum int, list *app.WorkItemLinkTypeList) {
	require.NotNil(t, list.Data)
	require.Condition(t, func() bool {
		return (len(list.Data) >= minNum)
	}, "list must at least have %d element(s) but it only has %d element(s)", minNum, len(list.Data))
}

// findIDsInList checks that all given IDs can be found in the list
func (s *workItemLinkTypeSuite) requireIDsInList(t *testing.T, list *app.WorkItemLinkTypeList, IDs ...uuid.UUID) {
	for _, id := range IDs {
		found := false
		for _, element := range list.Data {
			if *element.ID == id {
				found = true
			}
		}
		require.True(t, found, "failed to find ID %s in list", id)
	}
}

func assertWorkItemLinkType(t *testing.T, expected *app.WorkItemLinkTypeSingle, actual *app.WorkItemLinkTypeSingle) {
	require.NotNil(t, actual)
	expectedModel, err := ConvertWorkItemLinkTypeToModel(*expected)
	require.Nil(t, err)
	actualModel, err := ConvertWorkItemLinkTypeToModel(*actual)
	require.Nil(t, err)
	require.Equal(t, expectedModel.ID, actualModel.ID)

	// Check that the link category is included in the response in the "included" array
	require.Len(t, actual.Included, 2, "The work item link type should include it's work item link category and space.")
	categoryData, ok := actual.Included[0].(*app.WorkItemLinkCategoryData)
	require.True(t, ok, "work item link category is missing from the \"included\" array in the response")
	require.Equal(t, expectedModel.LinkCategoryID, *categoryData.ID)

	// Check that the link space is included in the response in the "included" array
	spaceData, ok := actual.Included[1].(*app.Space)
	require.True(t, ok, "space is missing from the \"included\" array in the response")
	require.Equal(t, expectedModel.SpaceID, *spaceData.ID)

	require.NotNil(t, actual.Data.Links, "The link type MUST include a self link")
	require.NotEmpty(t, actual.Data.Links.Self, "The link type MUST include a self link that's not empty")
}

func (s *workItemLinkTypeSuite) createWorkItemLinkTypes(t *testing.T) (*app.WorkItemTypeSingle, *app.WorkItemLinkTypeSingle) {
	bugBlockerPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("bug blocker"), s.linkCatID, s.spaceID)
	_, bugBlockerType := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, bugBlockerPayload)
	require.NotNil(t, bugBlockerType)

	workItemTypePayload := CreateWorkItemType(uuid.NewV4(), s.spaceID)
	_, workItemType := test.CreateWorkitemtypeCreated(t, s.svc.Context, s.svc, s.typeCtrl, s.spaceID, &workItemTypePayload)
	require.NotNil(t, workItemType)

	relatedPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("related"), s.linkCatID, s.spaceID)
	_, relatedType := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, relatedPayload)
	require.NotNil(t, relatedType)

	wiltcPayload, err := CreateWorkItemLinkTypeCombination(s.spaceID, *relatedType.Data.ID, *workItemType.Data.ID, *workItemType.Data.ID)
	require.Nil(t, err)
	_, wiltcCreated := test.CreateWorkItemLinkTypeCombinationCreated(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, wiltcPayload)
	require.NotNil(t, wiltcCreated)

	return workItemType, relatedType
}
