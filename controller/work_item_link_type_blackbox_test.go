package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
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
	linkCatID uuid.UUID
	testDir   string
}

func (s *workItemLinkTypeSuite) SetupSuite() {
	log.Info(nil, nil, "----- BEGIN Setup Suite -----")
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
	s.testDir = filepath.Join("test-files", "work_item_link_type")
	log.Info(nil, nil, "----- END Setup Suite -----")
}

func (s *workItemLinkTypeSuite) SetupTest() {
	log.Info(nil, nil, "----- BEGIN Setup Test -----")
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)

	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCombinationCtrl = NewWorkItemLinkTypeCombinationController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.typeCtrl = NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	// Create a few resources needed along the way in most tests

	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	// space
	s.spaceID = uuid.FromStringOrNil("26efe317-fbc2-4b4b-8369-bf5cf2325424")
	spaceCtrl := NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	spacePayload := CreateSpacePayloadWithID(s.spaceID, "space "+s.spaceID.String(), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, spaceCtrl, spacePayload)
	require.NotNil(s.T(), space)

	// link category
	s.linkCatID = uuid.FromStringOrNil("cca21f05-48e8-45be-b506-a7ddfd3bf520")
	linkCatCtrl := NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
	linkCatPayload := CreateWorkItemLinkCategoryWithID(s.linkCatID, "link category "+s.linkCatID.String())
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)
	log.Info(nil, nil, "----- END Setup Test -----")
}

func (s *workItemLinkTypeSuite) TearDownTest() {
	log.Info(nil, nil, "----- START Tear down test -----")
	s.clean()
	log.Info(nil, nil, "----- END Tear down test -----")
}

func (s *workItemLinkTypeSuite) TestCreateAndDelete() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	// given
	id := uuid.FromStringOrNil("a764bfe2-f824-4b60-992a-275825bd9400")

	s.T().Run("ok", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkTypeWithID(id, "link type "+id.String(), s.linkCatID, s.spaceID)
		// when
		_, workItemLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, createPayload)
		// then
		require.NotNil(s.T(), workItemLinkType)
		compareWithGolden(t, filepath.Join(s.testDir, "create", "ok.golden"), workItemLinkType)
	})

	s.T().Run("delete created link type", func(t *testing.T) {
		_ = test.DeleteWorkItemLinkTypeOK(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, id)
	})
}

func (s *workItemLinkTypeSuite) TestValidateCreatePayload() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("all valid", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType("empty name", s.linkCatID, s.spaceID)
		// when
		valid := createPayload.Validate()
		// then
		require.Nil(t, valid)
	})

	s.T().Run("empty name", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType("to be replaced later", s.linkCatID, s.spaceID)
		emptyName := ""
		createPayload.Data.Attributes.Name = &emptyName
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "empty_name.golden"), valid)
	})

	s.T().Run("empty topology", func(t *testing.T) {
		// given
		createPayload := CreateWorkItemLinkType("empty topology", s.linkCatID, s.spaceID)
		emptyTopology := ""
		createPayload.Data.Attributes.Topology = &emptyTopology
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "empty_topology.golden"), valid)
	})

	s.T().Run("wrong topology", func(t *testing.T) {
		createPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("wrong topology"), s.linkCatID, s.spaceID)
		wrongTopology := "wrongtopology"
		createPayload.Data.Attributes.Topology = &wrongTopology
		// when
		valid := createPayload.Validate()
		// then
		require.NotNil(t, valid)
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "wrong_topology.golden"), valid)
	})
}

func (s *workItemLinkTypeSuite) TestDelete() {
	s.T().Run("not found link type", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.FromStringOrNil("724d3f54-bb7b-4993-957e-e503c0ba6376")
		// whem
		_, jerrs := test.DeleteWorkItemLinkTypeNotFound(t, s.svc.Context, s.svc, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID)
		// then
		require.NotNil(t, jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "delete", "not_found_link_type.golden"), jerrs)
	})
}

func (s *workItemLinkTypeSuite) TestUpdate() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("ok", func(t *testing.T) {
		// given
		id := uuid.FromStringOrNil("7c3b29a8-8d70-42f1-9a19-4358e4e22705")
		createPayload := CreateWorkItemLinkTypeWithID(id, "link type "+id.String(), s.linkCatID, s.spaceID)
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
		require.NotNil(t, lt)
		compareWithGolden(t, filepath.Join(s.testDir, "update", "ok.golden"), lt)
	})

	s.T().Run("not found link type", func(t *testing.T) {
		// given
		id := uuid.FromStringOrNil("184f7773-ba54-4afa-8e1e-5d703e2ec517")
		createPayload := CreateWorkItemLinkTypeWithID(id, "link type "+id.String(), s.linkCatID, s.spaceID)
		notExistingId := uuid.FromStringOrNil("3e95d92a-295a-4cf5-8856-659704a709df")
		createPayload.Data.ID = &notExistingId
		// Wrap Data portion in an update payload instead of a create payload
		updateLinkTypePayload := &app.UpdateWorkItemLinkTypePayload{
			Data: createPayload.Data,
		}
		// then
		_, jerrs := test.UpdateWorkItemLinkTypeNotFound(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, notExistingId, updateLinkTypePayload)
		require.NotNil(t, jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "update", "not_found_link_type.golden"), jerrs)
	})

	s.T().Run("conflict", func(t *testing.T) {
		// given
		id := uuid.FromStringOrNil("7b0fa8bd-ee60-4ccb-9df7-ebb1ba67b163")
		createPayload := CreateWorkItemLinkTypeWithID(id, "link type "+id.String(), s.linkCatID, s.spaceID)
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
		// when
		_, jerrs := test.UpdateWorkItemLinkTypeConflict(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, *updateLinkTypePayload.Data.ID, updateLinkTypePayload)
		// then
		require.NotNil(t, jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "update", "conflict.golden"), jerrs)
	})
}

func (s *workItemLinkTypeSuite) TestShow() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("ok", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, nil, nil)
		// then
		require.NotNil(t, readWorkItemLinkType)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok.golden"), readWorkItemLinkType)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(createdWorkItemLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, &ifModifiedSinceHeader, nil)
		// then
		require.NotNil(t, readWorkItemLinkType)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "using_expired_ifmodifiedsince_header.golden"), readWorkItemLinkType)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using IfNoneMatch header", func(t *testing.T) {
		// given
		createdWorkItemLinkType := s.createRandomWorkItemLinkType(t)
		// when
		ifNoneMatch := "foo"
		res, readWorkItemLinkType := test.ShowWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *createdWorkItemLinkType.Data.ID, nil, &ifNoneMatch)
		// then
		require.NotNil(t, readWorkItemLinkType)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "using_ifnonematch_header.golden"), readWorkItemLinkType)
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
		notExistingLinkTypeID := uuid.FromStringOrNil("5560a60d-f506-4fc1-958d-3f9fc0f90684")
		// when
		_, jerrs := test.ShowWorkItemLinkTypeNotFound(s.T(), nil, nil, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID, nil, nil)
		// then
		require.NotNil(t, jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "not_found.golden"), jerrs)
	})
}

func (s *workItemLinkTypeSuite) TestList() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("ok", func(t *testing.T) {
		// given
		type1ID := uuid.FromStringOrNil("4a507999-6381-4e14-980a-2cd604697da3")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)

		type2ID := uuid.FromStringOrNil("cc3168c2-adfc-45af-8abf-c1c17bbd523c")
		type2Payload := CreateWorkItemLinkTypeWithID(type2ID, "type2 "+type2ID.String(), s.linkCatID, s.spaceID)
		_, type2 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type2Payload)
		require.NotNil(t, type2)

		// when
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, nil, nil)
		// then
		require.NotNil(t, linkTypes)
		compareWithGolden(t, filepath.Join(s.testDir, "list", "ok.golden"), linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// given
		type1ID := uuid.FromStringOrNil("f4f24ea4-be81-4290-a01d-71952b312fad")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(type1.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, &ifModifiedSinceHeader, nil)
		// then
		compareWithGolden(t, filepath.Join(s.testDir, "list", "ok_using_expired_ifmodifiedsince_header.golden"), linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("ok using IfNoneMatch header", func(t *testing.T) {
		// given
		type1ID := uuid.FromStringOrNil("01dd8598-3771-4ecc-bb4d-85ac1d21be7e")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)
		// when
		ifNoneMatch := "foo"
		res, linkTypes := test.ListWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, nil, &ifNoneMatch)
		// then
		compareWithGolden(t, filepath.Join(s.testDir, "list", "ok_using_ifnonematch_header.golden"), linkTypes)
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfModifiedSince header", func(t *testing.T) {
		// given
		type1ID := uuid.FromStringOrNil("7739816b-2a77-40a5-b8e8-79fdd8e680ca")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(*type1.Data.Attributes.UpdatedAt)
		res := test.ListWorkItemLinkTypeNotModified(t, nil, nil, s.linkTypeCtrl, s.spaceID, &ifModifiedSinceHeader, nil)
		// then
		assertResponseHeaders(t, res)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// given
		type1ID := uuid.FromStringOrNil("2b5254b1-3ae0-4a97-8dc1-4e6a9f1a3aa0")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)
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
		type1ID := uuid.FromStringOrNil("2625819b-be5b-4841-9f2b-665fbd535261")
		type1Payload := CreateWorkItemLinkTypeWithID(type1ID, "type1 "+type1ID.String(), s.linkCatID, s.spaceID)
		_, type1 := test.CreateWorkItemLinkTypeCreated(t, s.svc.Context, s.svc, s.linkTypeCtrl, s.spaceID, type1Payload)
		require.NotNil(t, type1)
		_, wit1 := createRandomWorkItemTypeWithID(t, uuid.FromStringOrNil("cfb588a1-23a7-4a13-b4df-a09c9c6b32f7"), s.typeCtrl, s.spaceID)
		_, wit2 := createRandomWorkItemTypeWithID(t, uuid.FromStringOrNil("587a1208-2daa-420f-b4b9-ebb9a7fa42fe"), s.typeCtrl, s.spaceID)
		createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
			ID:           uuid.FromStringOrNil("ec4e1f0e-e33a-4328-835f-e2a56804eaa8"),
			SpaceID:      s.spaceID,
			LinkTypeID:   *type1.Data.ID,
			SourceTypeID: *wit1.Data.ID,
			TargetTypeID: *wit2.Data.ID,
		})
		require.Nil(t, err)
		_, combiCreated := test.CreateWorkItemLinkTypeCombinationCreated(t, context.Background(), nil, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
		require.NotNil(t, combiCreated)
		// when
		_, combinationList := test.ListTypeCombinationsWorkItemLinkTypeOK(t, nil, nil, s.linkTypeCtrl, s.spaceID, *type1.Data.ID, nil, nil)
		// then
		require.NotNil(t, combinationList)
		require.Len(t, combinationList.Data, 1)
		compareWithGolden(t, filepath.Join(s.testDir, "list_type_combinations", "ok.golden"), combinationList)
	})
	s.T().Run("not existing link type", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.FromStringOrNil("acca2c11-95c3-40bb-a18e-be88b80d91c6")
		// when
		_, jerr := test.ListTypeCombinationsWorkItemLinkTypeNotFound(t, nil, nil, s.linkTypeCtrl, space.SystemSpace, notExistingLinkTypeID, nil, nil)
		// then
		require.NotNil(t, jerr)
		compareWithGolden(t, filepath.Join(s.testDir, "list_type_combinations", "not_existing_link_type.golden"), jerr)
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

	wiltcPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
		SpaceID:      s.spaceID,
		LinkTypeID:   *relatedType.Data.ID,
		SourceTypeID: *workItemType.Data.ID,
		TargetTypeID: *workItemType.Data.ID,
	})
	require.Nil(t, err)
	_, wiltcCreated := test.CreateWorkItemLinkTypeCombinationCreated(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, wiltcPayload)
	require.NotNil(t, wiltcCreated)

	return workItemType, relatedType
}
