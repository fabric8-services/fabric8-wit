package main_test

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"testing"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
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
)

func TestGetWorkItemWithLegacyDescription(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestGetWorkItem-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	payload.Data.Attributes[workitem.SystemDescription] = "Test WI description"

	_, result := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)

	assert.NotNil(t, result.Data.Attributes[workitem.SystemCreatedAt])
	assert.NotNil(t, result.Data.Attributes[workitem.SystemDescription])
	_, wi := test.ShowWorkitemOK(t, nil, nil, controller, *result.Data.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", *result.Data.ID)
	}

	if *wi.Data.ID != *result.Data.ID {
		t.Errorf("Id should be %s, but is %s", *result.Data.ID, *wi.Data.ID)
	}
	assert.NotNil(t, wi.Data.Attributes[workitem.SystemCreatedAt])

	if *wi.Data.Relationships.Creator.Data.ID != account.TestIdentity.ID.String() {
		t.Errorf("Creator should be %s, but it is %s", account.TestIdentity.ID.String(), *wi.Data.Relationships.Creator.Data.ID)
	}
	wi.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	updatedDescription := "= Updated Test WI description"
	wi.Data.Attributes[workitem.SystemDescription] = updatedDescription

	payload2 := minimumRequiredUpdatePayload()
	payload2.Data.ID = wi.Data.ID
	payload2.Data.Attributes = wi.Data.Attributes

	_, updated := test.UpdateWorkitemOK(t, nil, nil, controller, *wi.Data.ID, &payload2)
	assert.NotNil(t, updated.Data.Attributes[workitem.SystemCreatedAt])

	assert.Equal(t, (result.Data.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(t, *result.Data.ID, *updated.Data.ID)
	assert.Equal(t, wi.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])
	assert.Equal(t, updatedDescription, updated.Data.Attributes[workitem.SystemDescription])

	test.DeleteWorkitemOK(t, nil, nil, controller, *result.Data.ID)
}

func TestCreateWI(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestCreateWI-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	_, created := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)
	if created.Data.ID == nil || *created.Data.ID == "" {
		t.Error("no id")
	}
	assert.NotNil(t, created.Data.Attributes[workitem.SystemCreatedAt])
	assert.NotNil(t, created.Data.Relationships.Creator.Data)
	assert.Equal(t, *created.Data.Relationships.Creator.Data.ID, account.TestIdentity.ID.String())
}

func TestCreateWorkItemWithoutContext(t *testing.T) {
	resource.Require(t, resource.Database)
	svc := goa.New("TestCreateWorkItemWithoutContext-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	test.CreateWorkitemUnauthorized(t, svc.Context, svc, controller, &payload)
}

func TestListByFields(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestListByFields-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "run integration test"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	_, wi := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)

	filter := "{\"system.title\":\"run integration test\"}"
	offset := "0"
	limit := 1
	_, result := test.ListWorkitemOK(t, nil, nil, controller, &filter, nil, nil, &limit, &offset)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result.Data) != 1 {
		t.Errorf("unexpected length, should be %d but is %d", 1, len(result.Data))
	}

	filter = fmt.Sprintf("{\"system.creator\":\"%s\"}", account.TestIdentity.ID.String())
	_, result = test.ListWorkitemOK(t, nil, nil, controller, &filter, nil, nil, &limit, &offset)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result.Data) != 1 {
		t.Errorf("unexpected length, should be %d but is %d ", 1, len(result.Data))
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, *wi.Data.ID)
}

func getWorkItemTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))

	if err != nil {
		t.Fatal("Could not parse different private key ", err)
	}

	createWIPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"type": "workitems"
				"attributes": {
					"system.state": "new",
					"system.title": "My special story",
					"system.description": "description"
				}
			}
		}`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItems,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItems,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItems,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItems,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPatch,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work Item
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItems + "/088481764871",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeWorkItemCUD(t *testing.T) {
	UnauthorizeCreateUpdateDeleteTest(t, getWorkItemTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWI-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemController(service, gormapplication.NewGormDB(DB))
		app.MountWorkitemController(service, controller)
		return nil
	})
}

func createPagingTest(t *testing.T, controller *WorkitemController, repo *testsupport.WorkItemRepository, totalCount int) func(start int, limit int, first string, last string, prev string, next string) {
	return func(start int, limit int, first string, last string, prev string, next string) {
		count := computeCount(totalCount, int(start), int(limit))
		repo.ListReturns(makeWorkItems(count), uint64(totalCount), nil)
		offset := strconv.Itoa(start)
		_, response := test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
		assertLink(t, "first", first, response.Links.First)
		assertLink(t, "last", last, response.Links.Last)
		assertLink(t, "prev", prev, response.Links.Prev)
		assertLink(t, "next", next, response.Links.Next)
		assert.Equal(t, totalCount, response.Meta.TotalCount)
	}
}

func assertLink(t *testing.T, l string, expected string, actual *string) {
	if expected == "" {
		if actual != nil {
			assert.Fail(t, fmt.Sprintf("link %s should be nil but is %s", l, *actual))
		}
	} else {
		if actual == nil {
			assert.Fail(t, "link %s should be %s, but is nil", l, expected)
		} else {
			assert.True(t, strings.HasSuffix(*actual, expected), "link %s should be %s, but is %s", l, expected, *actual)
		}
	}
}

func computeCount(totalCount int, start int, limit int) int {
	if start < 0 || start >= totalCount {
		return 0
	}
	if start+limit > totalCount {
		return totalCount - start
	}
	return limit
}

func makeWorkItems(count int) []*app.WorkItem {
	res := make([]*app.WorkItem, count)
	for index := range res {
		res[index] = &app.WorkItem{
			ID:     fmt.Sprintf("id%d", index),
			Type:   "foobar",
			Fields: map[string]interface{}{},
		}
	}
	return res
}

func TestPagingLinks(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginLinks-Service")
	assert.NotNil(t, svc)
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	pagingTest := createPagingTest(t, controller, repo, 13)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=12&page[limit]=5", "page[offset]=0&page[limit]=2", "page[offset]=7&page[limit]=5")
	pagingTest(10, 3, "page[offset]=0&page[limit]=1", "page[offset]=10&page[limit]=3", "page[offset]=7&page[limit]=3", "")
	pagingTest(0, 4, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=4", "", "page[offset]=4&page[limit]=4")
	pagingTest(4, 8, "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8", "page[offset]=0&page[limit]=4", "page[offset]=12&page[limit]=8")

	pagingTest(16, 14, "page[offset]=0&page[limit]=2", "page[offset]=2&page[limit]=14", "page[offset]=2&page[limit]=14", "")
	pagingTest(16, 18, "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "page[offset]=0&page[limit]=16", "")

	pagingTest(3, 50, "page[offset]=0&page[limit]=3", "page[offset]=3&page[limit]=50", "page[offset]=0&page[limit]=3", "")
	pagingTest(0, 50, "page[offset]=0&page[limit]=50", "page[offset]=0&page[limit]=50", "", "")

	pagingTest = createPagingTest(t, controller, repo, 0)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=2", "", "")
}

func TestPagingErrors(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginErrors-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(100), uint64(100), nil)

	var offset string = "-1"
	var limit int = 2
	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "0"
	limit = 0
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is 0", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "0"
	limit = -1
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "-3"
	limit = -1
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "ALPHA"
	limit = 40
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=40") {
		assert.Fail(t, "Limit is within range", "Expected limit to be size %d, but was %s", 40, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}
}

func TestPagingLinksHasAbsoluteURL(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginAbsoluteURL-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	offset := "10"
	limit := 10

	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.HasPrefix(*result.Links.First, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "First", *result.Links.First)
	}
	if !strings.HasPrefix(*result.Links.Last, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Last", *result.Links.Last)
	}
	if !strings.HasPrefix(*result.Links.Prev, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Prev", *result.Links.Prev)
	}
	if !strings.HasPrefix(*result.Links.Next, "http://") {
		assert.Fail(t, "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Next", *result.Links.Next)
	}
}

func TestPagingDefaultAndMaxSize(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	svc := goa.New("TestPaginSize-Service")
	db := testsupport.NewMockDB()
	controller := NewWorkitemController(svc, db)

	offset := "0"
	var limit int
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, nil, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is nil", "Expected limit to be default size %d, got %v", 20, *result.Links.First)
	}
	limit = 1000
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=100") {
		assert.Fail(t, "Limit is more than max", "Expected limit to be %d, got %v", 100, *result.Links.First)
	}

	limit = 50
	_, result = test.ListWorkitemOK(t, context.Background(), nil, controller, nil, nil, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=50") {
		assert.Fail(t, "Limit is within range", "Expected limit to be %d, got %v", 50, *result.Links.First)
	}
}

// ========== helper functions for tests inside WorkItem2Suite ==========
func getMinimumRequiredUpdatePayload(wi *app.WorkItem2) *app.UpdateWorkitemPayload {
	return &app.UpdateWorkitemPayload{
		Data: &app.WorkItem2{
			Type: APIStringTypeWorkItem,
			ID:   wi.ID,
			Attributes: map[string]interface{}{
				"version": wi.Attributes["version"],
			},
		},
	}
}

func minimumRequiredUpdatePayload() app.UpdateWorkitemPayload {
	return app.UpdateWorkitemPayload{
		Data: &app.WorkItem2{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
		},
	}
}

func minimumRequiredCreateWithType(wit string) app.CreateWorkitemPayload {
	c := minimumRequiredCreatePayload()
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: APIStringTypeWorkItemType,
			ID:   wit,
		},
	}
	return c
}

func minimumRequiredCreatePayload() app.CreateWorkitemPayload {
	return app.CreateWorkitemPayload{
		Data: &app.WorkItem2{
			Type:          APIStringTypeWorkItem,
			Attributes:    map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{},
		},
	}
}

func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB) *account.Identity {
	newUserUUID := uuid.NewV4()
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		FullName: "Test User Integration Random",
		ImageURL: "http://images.com/42",
		ID:       newUserUUID,
	}
	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		fmt.Println("should not happen off.")
		return nil
	}
	return &identity
}

func createOneRandomIteration(ctx context.Context, db *gorm.DB) *iteration.Iteration {
	iterationRepo := iteration.NewIterationRepository(db)
	itr := iteration.Iteration{
		Name: "Sprint 101",
	}
	err := iterationRepo.Create(ctx, &itr)
	if err != nil {
		fmt.Println("Failed to create iteration.")
		return nil
	}
	return &itr
}

// ========== WorkItem2Suite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type WorkItem2Suite struct {
	suite.Suite
	db             *gorm.DB
	clean          func()
	wiCtrl         app.WorkitemController
	wi2Ctrl        app.WorkitemController
	pubKey         *rsa.PublicKey
	priKey         *rsa.PrivateKey
	svc            *goa.Service
	wi             *app.WorkItem2
	minimumPayload *app.UpdateWorkitemPayload
}

func (s *WorkItem2Suite) SetupSuite() {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	s.db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())

	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	s.pubKey, _ = almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	s.priKey, _ = almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("TestUpdateWI2-Service", almtoken.NewManager(s.pubKey, s.priKey), account.TestIdentity)
	require.NotNil(s.T(), s.svc)

	s.wiCtrl = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.wiCtrl)

	s.wi2Ctrl = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.wi2Ctrl)

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if configuration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	s.clean = cleaner.DeleteCreatedEntities(s.db)
}

func (s *WorkItem2Suite) TearDownSuite() {
	s.clean()
	if s.db != nil {
		s.db.Close()
	}
}

func (s *WorkItem2Suite) SetupTest() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wiCtrl, &payload)
	s.wi = wi.Data
	s.minimumPayload = getMinimumRequiredUpdatePayload(s.wi)
}

// ========== Actual Test functions ==========
func (s *WorkItem2Suite) TestWI2UpdateOnlyState() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	s.minimumPayload.Data.Attributes["system.state"] = "invalid_value"
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	newStateValue := "closed"
	s.minimumPayload.Data.Attributes[workitem.SystemState] = newStateValue
	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), updatedWI.Data.Attributes[workitem.SystemState], newStateValue)
}

func (s *WorkItem2Suite) TestWI2UpdateVersionConflict() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	s.minimumPayload.Data.Attributes["version"] = 2398475203
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
}

func (s *WorkItem2Suite) TestWI2UpdateWithNonExistentID() {
	id := "2398475203"
	s.minimumPayload.Data.ID = &id
	test.UpdateWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, id, s.minimumPayload)
}

func (s *WorkItem2Suite) TestWI2UpdateWithInvalidID() {
	id := "some non-int ID"
	s.minimumPayload.Data.ID = &id
	// pass*s.wi.ID below, because that creates a route to the controller
	// if do not pass*s.wi.ID then we will be testing goa's code and not ours
	test.UpdateWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
}

func (s *WorkItem2Suite) TestWI2UpdateSetBaseType() {
	c := minimumRequiredCreateWithType(workitem.SystemBug)
	c.Data.Attributes[workitem.SystemTitle] = "Test title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	_, created := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.Equal(s.T(), created.Data.Relationships.BaseType.Data.ID, workitem.SystemBug)

	u := minimumRequiredUpdatePayload()
	u.Data.Attributes[workitem.SystemTitle] = "Test title"
	u.Data.Attributes["version"] = created.Data.Attributes["version"]
	u.Data.ID = created.Data.ID
	u.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				ID:   workitem.SystemExperience,
				Type: APIStringTypeWorkItemType, // Not allowed to change the WIT of a WI
			},
		},
	}

	_, newWi := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, &u)

	// Ensure the type wasn't updated
	require.Equal(s.T(), workitem.SystemBug, newWi.Data.Relationships.BaseType.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyLegacyDescription() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := "Only Description is modified"
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "Only Description is modified"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyMarkupDescriptionWithoutMarkup() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := rendering.NewMarkupContentFromLegacy("Only Description is modified")
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "Only Description is modified"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription
	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyMarkupDescriptionWithMarkup() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := rendering.NewMarkupContent("Only Description is modified", rendering.SystemMarkupMarkdown)
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "<p>Only Description is modified</p>\n"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateMultipleScenarios() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	// update title attribute
	modifiedTitle := "Is the model updated?"
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = modifiedTitle

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), updatedWI.Data.Attributes[workitem.SystemTitle], modifiedTitle)

	// verify self link value
	if !strings.HasPrefix(updatedWI.Links.Self, "http://") {
		assert.Fail(s.T(), fmt.Sprintf("%s is not absolute URL", updatedWI.Links.Self))
	}
	if !strings.HasSuffix(updatedWI.Links.Self, fmt.Sprintf("/%s", *updatedWI.Data.ID)) {
		assert.Fail(s.T(), fmt.Sprintf("%s is not FETCH URL of the resource", updatedWI.Links.Self))
	}
	// clean up and keep version updated in order to keep object future usage
	delete(s.minimumPayload.Data.Attributes, workitem.SystemTitle)
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	s.minimumPayload.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]

	// update assignee relationship and verify
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	require.NotNil(s.T(), newUser)

	newUserUUID := newUser.ID.String()
	s.minimumPayload.Data.Relationships = &app.WorkItemRelationships{}

	userType := APIStringTypeUser
	// update with invalid assignee string (non-UUID)
	maliciousUUID := "non UUID string"
	s.minimumPayload.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			&app.GenericData{
				ID:   &maliciousUUID,
				Type: &userType,
			}},
	}
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)

	s.minimumPayload.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			&app.GenericData{
				ID:   &newUserUUID,
				Type: &userType,
			}},
	}

	_, updatedWI = test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), *updatedWI.Data.Relationships.Assignees.Data[0].ID, newUser.ID.String())

	// update to wrong version
	correctVersion := updatedWI.Data.Attributes["version"]
	s.minimumPayload.Data.Attributes["version"] = 12453972348
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	s.minimumPayload.Data.Attributes["version"] = correctVersion

	// Add test to remove assignee for WI
	s.minimumPayload.Data.Relationships.Assignees.Data = nil
	_, updatedWI = test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	require.Len(s.T(), updatedWI.Data.Relationships.Assignees.Data, 0)
	// need to do in order to keep object future usage
	s.minimumPayload.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItem() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), "Title", wi.Data.Attributes[workitem.SystemTitle])
	assert.NotNil(s.T(), wi.Data.Relationships.BaseType.Data.ID)
	assert.NotNil(s.T(), wi.Data.Relationships.Comments.Links.Self)
	assert.NotNil(s.T(), wi.Data.Relationships.Creator.Data.ID)
	assert.NotNil(s.T(), wi.Data.Links)
	assert.NotNil(s.T(), wi.Data.Links.Self)
}

// TestWI2SuccessCreateWorkItemWithoutDescription verifies that the `workitem.SystemDescription` attribute is not set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is not provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithoutDescription() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescription])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescriptionMarkup])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescriptionRendered])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithLegacyDescription() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = "Description"
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithDescriptionAndMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown)
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "<p>Description</p>\n", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set as a MarkupContent element
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithDescriptionAndNoMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy("Description")
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2FailCreateWorkItemWithDescriptionAndUnsupportedMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContent("Description", "foo")
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when/then
	test.CreateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateMissingBaseType() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when/then
	test.CreateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithAssigneeAsField() {
	// given
	s.T().Skip("Not working.. require WIT understanding on server side")
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemAssignees] = []string{"34343"}
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Nil(s.T(), wi.Data.Relationships.Assignees.Data)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithMissingTitle() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when/then
	test.CreateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithEmptyTitle() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = ""
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	// when/then
	test.CreateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWithAssigneeRelation() {
	// given
	userType := "identities"
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	newUserId := newUser.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				&app.GenericData{
					Type: &userType,
					ID:   &newUserId,
				}},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.NotNil(s.T(), wi.Data.Relationships.Assignees.Data)
	assert.NotNil(s.T(), wi.Data.Relationships.Assignees.Data[0].ID)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWithAssigneesRelation() {
	// given
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	newUser2 := createOneRandomUserIdentity(s.svc.Context, s.db)
	newUser3 := createOneRandomUserIdentity(s.svc.Context, s.db)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser.ID),
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *wi.Data.Relationships.Assignees.Data[0].ID)
	update := minimumRequiredUpdatePayload()
	update.Data.ID = wi.Data.ID
	update.Data.Type = wi.Data.Type
	update.Data.Attributes[workitem.SystemTitle] = "Title"
	update.Data.Attributes["version"] = wi.Data.Attributes["version"]
	update.Data.Relationships = &app.WorkItemRelationships{
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser2.ID),
				ident(newUser3.ID),
			},
		},
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID, &update)
	assert.Len(s.T(), wiu.Data.Relationships.Assignees.Data, 2)
	assert.Equal(s.T(), newUser2.ID.String(), *wiu.Data.Relationships.Assignees.Data[0].ID)
	assert.Equal(s.T(), newUser3.ID.String(), *wiu.Data.Relationships.Assignees.Data[1].ID)
}

func (s *WorkItem2Suite) TestWI2ListByAssigneeFilter() {
	// given
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser.ID),
			},
		},
	}
	// when
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *wi.Data.Relationships.Assignees.Data[0].ID)
	newUserID := newUser.ID.String()
	_, list := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, &newUserID, nil, nil, nil)
	assert.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *list.Data[0].Relationships.Assignees.Data[0].ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[assignee]"))
}

func (s *WorkItem2Suite) TestWI2ListByIterationFilter() {
	tempIteration := createOneRandomIteration(s.svc.Context, s.db)
	require.NotNil(s.T(), tempIteration)
	iterationID := tempIteration.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: APIStringTypeWorkItemType,
				ID:   workitem.SystemBug,
			},
		},
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				ID: &iterationID,
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	assert.Equal(s.T(), iterationID, *wi.Data.Relationships.Iteration.Data.ID)

	_, list := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, nil, &iterationID, nil, nil)
	require.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), iterationID, *list.Data[0].Relationships.Iteration.Data.ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[iteration]"))
}

func (s *WorkItem2Suite) TestWI2FailCreateInvalidAssignees() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(uuid.NewV4()),
			},
		},
	}
	// when/then
	test.CreateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2FailUpdateInvalidAssignees() {
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser.ID),
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)

	update := minimumRequiredUpdatePayload()
	update.Data.ID = wi.Data.ID
	update.Data.Type = wi.Data.Type
	update.Data.Attributes["version"] = wi.Data.Attributes["version"]
	update.Data.Relationships = &app.WorkItemRelationships{
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(uuid.NewV4()),
			},
		},
	}
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID, &update)
}

func (s *WorkItem2Suite) TestWI2SuccessUpdateWithAssigneesRelation() {
	newUser := createOneRandomUserIdentity(s.svc.Context, s.db)
	newUser2 := createOneRandomUserIdentity(s.svc.Context, s.db)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser.ID),
				ident(newUser2.ID),
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 2)
}

func (s *WorkItem2Suite) TestWI2SuccessShow() {
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	assert.NotNil(s.T(), fetchedWi.Data)
	assert.NotNil(s.T(), fetchedWi.Data.ID)
	assert.Equal(s.T(), *createdWi.Data.ID, *fetchedWi.Data.ID)
	assert.NotNil(s.T(), fetchedWi.Data.Type)
	assert.NotNil(s.T(), fetchedWi.Data.Attributes)
	assert.NotNil(s.T(), fetchedWi.Data.Links.Self)
	assert.NotNil(s.T(), fetchedWi.Data.Relationships.Creator.Data.ID)
	assert.NotNil(s.T(), fetchedWi.Data.Relationships.BaseType.Data.ID)

}

func (s *WorkItem2Suite) TestWI2FailShowMissing() {
	test.ShowWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, "00000000")
}

func (s *WorkItem2Suite) TestWI2SuccessDelete() {
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	test.DeleteWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	test.ShowWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
}

func (s *WorkItem2Suite) TestWI2FailMissingDelete() {
	test.DeleteWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, "00000000")
}

func (s *WorkItem2Suite) TestWI2CreateWithIteration() {
	t := s.T()

	iterationInstance := createSpaceAndIteration(t, gormapplication.NewGormDB(s.db))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &itType,
				ID:   &iterationID,
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Iteration)
	assert.Equal(t, iterationID, *wi.Data.Relationships.Iteration.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateWithIteration() {
	t := s.T()

	iterationInstance := createSpaceAndIteration(t, gormapplication.NewGormDB(s.db))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Iteration)
	assert.Nil(t, wi.Data.Relationships.Iteration.Data)

	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &itType,
				ID:   &iterationID,
			},
		},
	}

	_, wiu := test.UpdateWorkitemOK(t, s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID, &u)
	require.NotNil(t, wiu.Data.Relationships.Iteration)
	require.NotNil(t, wiu.Data.Relationships.Iteration.Data)
	assert.Equal(t, iterationID, *wiu.Data.Relationships.Iteration.Data.ID)
	assert.Equal(t, itType, *wiu.Data.Relationships.Iteration.Data.Type)
}

func (s *WorkItem2Suite) TestWI2UpdateRemoveIteration() {
	t := s.T()

	t.Skip("iteration.data can't be sent as nil from client libs since it's optionall and is removed during json encoding")

	iterationInstance := createSpaceAndIteration(t, gormapplication.NewGormDB(s.db))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &itType,
				ID:   &iterationID,
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Iteration)
	assert.NotNil(t, wi.Data.Relationships.Iteration.Data)

	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Iteration: &app.RelationGeneric{
			Data: nil,
		},
	}

	_, wiu := test.UpdateWorkitemOK(t, s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID, &u)
	assert.NotNil(t, wiu.Data.Relationships.Iteration)
	assert.Nil(t, wiu.Data.Relationships.Iteration.Data)
}

func (s *WorkItem2Suite) TestWI2CreateUnknownIteration() {
	t := s.T()

	itType := iteration.APIStringTypeIteration
	iterationID := uuid.NewV4().String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &itType,
				ID:   &iterationID,
			},
		},
	}
	test.CreateWorkitemBadRequest(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithLegacyDescription() {
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := "<img src=x onerror=alert('description') />"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	require.NotNil(s.T(), fetchedWi.Data)
	require.NotNil(s.T(), fetchedWi.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWi.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), html.EscapeString(description), fetchedWi.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithPlainTextDescription() {
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := rendering.NewMarkupContent("<img src=x onerror=alert('description') />", rendering.SystemMarkupPlainText)
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	require.NotNil(s.T(), fetchedWi.Data)
	require.NotNil(s.T(), fetchedWi.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWi.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), html.EscapeString(description.Content), fetchedWi.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithMarkdownDescription() {
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := rendering.NewMarkupContent("<img src=x onerror=alert('description') />", rendering.SystemMarkupMarkdown)
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	require.NotNil(s.T(), fetchedWi.Data)
	require.NotNil(s.T(), fetchedWi.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWi.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), "<p>"+html.EscapeString(description.Content)+"</p>\n", fetchedWi.Data.Attributes[workitem.SystemDescriptionRendered])
}

// a normal test function that will kick off WorkItem2Suite
func TestSuiteWorkItem2(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItem2Suite))
}

func ident(id uuid.UUID) *app.GenericData {
	ut := APIStringTypeUser
	i := id.String()
	return &app.GenericData{
		Type: &ut,
		ID:   &i,
	}
}
