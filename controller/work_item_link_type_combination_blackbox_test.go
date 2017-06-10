package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"path/filepath"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem/link"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CreateWorkItemLinkTypeCombination defines a work item link type combination
func CreateWorkItemLinkTypeCombinationPayload(tc link.WorkItemLinkTypeCombination) (*app.CreateWorkItemLinkTypeCombinationPayload, error) {
	appl := new(application.Application)
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	payload, err := ConvertWorkItemLinkTypeCombinationFromModel(*appl, reqLong, tc)
	if err != nil {
		return nil, err
	}
	// The create payload is required during creation. Simply copy data over.
	res := &app.CreateWorkItemLinkTypeCombinationPayload{
		Data: payload,
	}
	return res, nil
}

func TestSuiteWorkItemLinkTypeCombination(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkTypeCombinationSuite))
}

type workItemLinkTypeCombinationSuite struct {
	gormtestsupport.DBTestSuite

	clean                   func()
	linkTypeCombinationCtrl *WorkItemLinkTypeCombinationController
	typeCtrl                *WorkitemtypeController
	svc                     *goa.Service

	spaceID    uuid.UUID
	wit1ID     uuid.UUID
	wit2ID     uuid.UUID
	linkTypeID uuid.UUID
	testDir    string
}

func (s *workItemLinkTypeCombinationSuite) SetupSuite() {
	log.Info(nil, nil, "----- BEGIN Setup Suite -----")
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
	s.testDir = filepath.Join("test-files", "work_item_link_type_combination")
	log.Info(nil, nil, "----- END Setup Suite -----")
}

func (s *workItemLinkTypeCombinationSuite) SetupTest() {
	log.Info(nil, nil, "----- BEGIN Setup Test -----")
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)

	s.linkTypeCombinationCtrl = NewWorkItemLinkTypeCombinationController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	// Create a few resources needed along the way in most tests

	// space
	s.spaceID = uuid.FromStringOrNil("38f6a5e5-c241-4477-894b-530461636056")
	spaceCtrl := NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	spacePayload := CreateSpacePayloadWithID(s.spaceID, "space "+s.spaceID.String(), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, spaceCtrl, spacePayload)
	require.NotNil(s.T(), space)

	// WIT 1
	s.wit1ID = uuid.FromStringOrNil("f3b1d121-04ad-496d-a9c1-4cbea99185a3")
	witCtrl := NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	_, wit1 := createRandomWorkItemTypeWithID(s.T(), s.wit1ID, witCtrl, s.spaceID)
	require.NotNil(s.T(), wit1)

	// WIT 2
	s.wit2ID = uuid.FromStringOrNil("143befee-a646-4ce6-a192-b38134db4075")
	_, wit2 := createRandomWorkItemTypeWithID(s.T(), s.wit2ID, witCtrl, s.spaceID)
	require.NotNil(s.T(), wit2)

	// link category
	linkCatCtrl := NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
	linkCatPayload := CreateWorkItemLinkCategory(testsupport.CreateRandomValidTestName("link category"))
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)
	linkCatID := *linkCat.Data.ID

	// link type
	linkTypeCtrl := NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	wiltPayload := CreateWorkItemLinkType(testsupport.CreateRandomValidTestName("parenting"), linkCatID, s.spaceID)
	s.linkTypeID = uuid.FromStringOrNil("53ca887a-025b-4be7-9a79-79e0c8e28fa3")
	wiltPayload.Data.ID = &s.linkTypeID
	_, linkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, linkTypeCtrl, s.spaceID, wiltPayload)
	require.NotNil(s.T(), linkType)
	log.Info(nil, nil, "----- END Setup Test -----")
}

func (s *workItemLinkTypeCombinationSuite) TearDownTest() {
	log.Info(nil, nil, "----- START Tear down test -----")
	s.clean()
	log.Info(nil, nil, "----- END Tear down test -----")
}

func (s *workItemLinkTypeCombinationSuite) TestCreate() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("ok", func(t *testing.T) {
		// given
		createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
			ID:           uuid.FromStringOrNil("4c986308-0f19-41f9-b8b3-b904291bda28"),
			SpaceID:      s.spaceID,
			LinkTypeID:   s.linkTypeID,
			SourceTypeID: s.wit1ID,
			TargetTypeID: s.wit2ID,
		})
		require.Nil(t, err)
		// when
		_, combination := test.CreateWorkItemLinkTypeCombinationCreated(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
		// then
		require.NotNil(t, combination)
		goldenFile := filepath.Join(s.testDir, "create", "ok.golden")
		compareWithGolden(t, goldenFile, combination)
	})

	s.T().Run("work item link type not found", func(t *testing.T) {
		// given
		notExistingLinkTypeID := uuid.FromStringOrNil("7b161a5a-9455-4ac4-923d-c98cd3d3546e")
		createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
			ID:           uuid.FromStringOrNil("bd517a2-54a4-4a73-98b6-79a34ac00b32"),
			SpaceID:      s.spaceID,
			LinkTypeID:   notExistingLinkTypeID,
			SourceTypeID: s.wit1ID,
			TargetTypeID: s.wit2ID,
		})
		require.Nil(t, err)
		// when
		_, jerr := test.CreateWorkItemLinkTypeCombinationNotFound(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
		// then
		require.NotNil(t, jerr)
		goldenFile := filepath.Join(s.testDir, "create", "link_type_not_found.golden")
		compareWithGolden(t, goldenFile, jerr)
	})

	s.T().Run("source work item type not found", func(t *testing.T) {
		// given
		notExistingSourceWorkItemTypeID := uuid.FromStringOrNil("c7cc661e-a883-4514-8f2a-6fb6ee68a4d4")
		createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
			ID:           uuid.FromStringOrNil("d8dcabcc-feec-4b74-8b53-6f28d6cd59d7"),
			SpaceID:      s.spaceID,
			LinkTypeID:   s.linkTypeID,
			SourceTypeID: notExistingSourceWorkItemTypeID,
			TargetTypeID: s.wit2ID,
		})
		require.Nil(t, err)
		// when
		_, jerr := test.CreateWorkItemLinkTypeCombinationNotFound(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
		// then
		require.NotNil(t, jerr)
		goldenFile := filepath.Join(s.testDir, "create", "source_work_item_type_not_found.golden")
		compareWithGolden(t, goldenFile, jerr)
	})

	s.T().Run("target work item type not found", func(t *testing.T) {
		// given
		notExistingTargetWorkItemTypeID := uuid.FromStringOrNil("959f3ee6-4171-44ff-9200-ed592d5a8722")
		createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
			ID:           uuid.FromStringOrNil("ff43e74a-00c3-4d81-ab91-a4c736f71163"),
			SpaceID:      s.spaceID,
			LinkTypeID:   s.linkTypeID,
			SourceTypeID: s.wit1ID,
			TargetTypeID: notExistingTargetWorkItemTypeID,
		})
		require.Nil(t, err)
		// when
		_, jerr := test.CreateWorkItemLinkTypeCombinationNotFound(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
		// then
		require.NotNil(t, jerr)
		goldenFile := filepath.Join(s.testDir, "create", "target_work_item_type_not_found.golden")
		compareWithGolden(t, goldenFile, jerr)
	})
}

func (s *workItemLinkTypeCombinationSuite) TestShow() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	createdAt := time.Date(2016, time.January, 2, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2017, time.April, 27, 0, 0, 0, 0, time.UTC)

	// given
	id := uuid.FromStringOrNil("c1bc83fc-25c7-4698-90fe-e6a59a74cc06")

	createPayload, err := CreateWorkItemLinkTypeCombinationPayload(link.WorkItemLinkTypeCombination{
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ID:           id,
		SpaceID:      s.spaceID,
		LinkTypeID:   s.linkTypeID,
		SourceTypeID: s.wit1ID,
		TargetTypeID: s.wit2ID,
	})
	require.Nil(s.T(), err)
	_, combination := test.CreateWorkItemLinkTypeCombinationCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, createPayload)
	require.NotNil(s.T(), combination)

	s.T().Run("ok", func(t *testing.T) {
		// when
		_, shownCombi := test.ShowWorkItemLinkTypeCombinationOK(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, id, nil, nil)
		// then
		require.NotNil(t, shownCombi)
		goldenFile := filepath.Join(s.testDir, "show", "ok.golden")
		compareWithGolden(t, goldenFile, shownCombi)
	})

	s.T().Run("not found", func(t *testing.T) {
		// given
		notExistingID := uuid.FromStringOrNil("1c195b42-4d2d-428c-9a11-0cead28c35b9")
		// when
		_, jerr := test.ShowWorkItemLinkTypeCombinationNotFound(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, notExistingID, nil, nil)
		// then
		require.NotNil(t, jerr)
		goldenFile := filepath.Join(s.testDir, "show", "not_found.golden")
		compareWithGolden(t, goldenFile, jerr)
	})

	s.T().Run("ok using expired IfModifiedSince header", func(t *testing.T) {
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
		respWriter, result := test.ShowWorkItemLinkTypeCombinationOK(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, id, &ifModifiedSinceHeader, nil)
		// then
		require.NotNil(t, result)
		goldenFile := filepath.Join(s.testDir, "show", "ok_using_expired_ifmodifiedsince_header.golden")
		compareWithGolden(t, goldenFile, result)
		assertResponseHeaders(t, respWriter)
	})

	s.T().Run("ok using IfNoneMatch header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		respWriter, result := test.ShowWorkItemLinkTypeCombinationOK(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, id, nil, &ifNoneMatch)
		// then
		require.NotNil(t, result)
		goldenFile := filepath.Join(s.testDir, "show", "ok_using_ifnonematch_header.golden")
		compareWithGolden(t, goldenFile, result)
		assertResponseHeaders(t, respWriter)
	})

	s.T().Run("not modified using expired IfModifiedSince header", func(t *testing.T) {
		// when
		ifModifiedSinceHeader := app.ToHTTPTime(updatedAt)
		respWriter := test.ShowWorkItemLinkTypeCombinationNotModified(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, id, &ifModifiedSinceHeader, nil)
		// then
		assertResponseHeaders(t, respWriter)
	})

	s.T().Run("not modified using IfNoneMatch header", func(t *testing.T) {
		// when
		model, err := ConvertWorkItemLinkTypeCominationToModel(*combination.Data)
		require.Nil(t, err)
		ifNoneMatch := app.GenerateEntityTag(model)
		respWriter := test.ShowWorkItemLinkTypeCombinationNotModified(t, s.svc.Context, s.svc, s.linkTypeCombinationCtrl, s.spaceID, id, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(t, respWriter)
	})
}

func (s *workItemLinkTypeCombinationSuite) getWorkItemLinkTypeCombinationTestDataFunc() func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {

		privatekey, err := jwt.ParseRSAPrivateKeyFromPEM(s.Configuration.GetTokenPrivateKey())
		if err != nil {
			t.Fatal("Could not parse Key ", err)
		}
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
		if err != nil {
			t.Fatal("Could not parse different private key ", err)
		}

		createWorkItemLinkTypeCombinationPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"type": "workitemlinktypecombinations",
				"id": "4c986308-0f19-41f9-b8b3-b904291bda28",
				"attributes": {
					"version": 0
				},
				"relationships": {
					"link_type": { "data": {"id": "53ca887a-025b-4be7-9a79-79e0c8e28fa3", "type": "workitemlinktypes"}},
					"source_type": { "data": {"id": "f3b1d121-04ad-496d-a9c1-4cbea99185a3", "type": "workitemtypes"}},
					"space": { "data": { "id": "38f6a5e5-c241-4477-894b-530461636056", "type": "spaces"}},
					"target_type": { "data": {"id": "143befee-a646-4ce6-a192-b38134db4075", "type": "workitemtypes"}}
				}
			}
		}
		`))
		return []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypeCombinations, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypeCombinationPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypeCombinations, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypeCombinationPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypeCombinations, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypeCombinationPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemLinkTypeCombinations, "6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWorkItemLinkTypeCombinationPayloadString,
				jwtToken:           "",
			},
			// Try fetching a random work item link type combination
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemLinkTypeCombinations, "6ba7b810-9dad-11d1-80b4-00c04fd430c8") + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkTypeCombinationSuite) TestUnauthorizeWorkItemLinkTypeCombination() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemLinkTypeCombinationTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLinkTypeCombination-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkTypeCombinationController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkItemLinkTypeCombinationController(service, controller)
		return nil
	})
}
