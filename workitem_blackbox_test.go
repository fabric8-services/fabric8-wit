package main_test

import (
	"bytes"
	"fmt"
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
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestGetWorkItem-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)
	payload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle:   "Test WI",
			models.SystemCreator: "aslak",
			models.SystemState:   "closed"},
	}

	_, result := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, wi.ID)
	}

	if wi.Fields[models.SystemCreator] != account.TestIdentity.ID.String() {
		t.Errorf("Creator should be %s, but it is %s", account.TestIdentity.ID.String(), wi.Fields[models.SystemCreator])
	}
	wi.Fields[models.SystemTitle] = "Updated Test WI"
	payload2 := app.UpdateWorkItemPayload{
		Type:    wi.Type,
		Version: wi.Version,
		Fields:  wi.Fields,
	}
	_, updated := test.UpdateWorkitemOK(t, nil, nil, controller, wi.ID, &payload2)
	if updated.Version != result.Version+1 {
		t.Errorf("expected version %d, but got %d", result.Version+1, updated.Version)
	}
	if updated.ID != result.ID {
		t.Errorf("id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.Fields[models.SystemTitle] != "Updated Test WI" {
		t.Errorf("expected title %s, but got %s", "Updated Test WI", updated.Fields[models.SystemTitle])
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestCreateWI-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)
	payload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle:   "Test WI",
			models.SystemCreator: "tmaeder",
			models.SystemState:   models.SystemStateNew,
		},
	}

	_, created := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
	assert.NotNil(t, created.Fields[models.SystemCreator])
	assert.Equal(t, created.Fields[models.SystemCreator], account.TestIdentity.ID.String())
}

func TestCreateWorkItemWithoutContext(t *testing.T) {
	resource.Require(t, resource.Database)
	svc := goa.New("TestCreateWorkItemWithoutContext-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)
	payload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle:   "Test WI",
			models.SystemCreator: "tmaeder",
			models.SystemState:   models.SystemStateNew,
		},
	}
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
	payload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle:   "run integration test",
			models.SystemCreator: "aslak",
			models.SystemState:   models.SystemStateClosed,
		},
	}

	_, wi := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)

	filter := "{\"system.title\":\"run integration test\"}"
	page := "0,1"
	_, result := test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d", 1, len(result))
	}

	filter = fmt.Sprintf("{\"system.creator\":\"%s\"}", account.TestIdentity.ID.String())
	_, result = test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d ", 1, len(result))
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, wi.ID)
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
			"type": "system.userstory",
			"fields": {
				"system.creator": "tmaeder",
				"system.state": "new",
				"system.title": "My special story",
				"system.description": "description"
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
			method:             http.MethodPut,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                endpointWorkItems + "/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
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
		// adding security tests for workitem.2 endpoint
		{
			method:             "PATCH",
			url:                "/api/workitems.2/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString, // doesnt matter actually because we expect it to fail
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "PATCH",
			url:                "/api/workitems.2/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString, // doesnt matter actually because we expect it to fail
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "PATCH",
			url:                "/api/workitems.2/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString, // doesnt matter actually because we expect it to fail
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "PATCH",
			url:                "/api/workitems.2/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error", // doesnt matter actually because we expect it to fail
			payload:            createWIPayloadString,
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
		controller2 := NewWorkitem2Controller(service, gormapplication.NewGormDB(DB))
		app.MountWorkitem2Controller(service, controller2)
		return nil
	})
}

func createPagingTest(t *testing.T, controller *Workitem2Controller, repo *testsupport.WorkItemRepository, totalCount int) func(start int, limit int, first string, last string, prev string, next string) {
	return func(start int, limit int, first string, last string, prev string, next string) {
		count := computeCount(totalCount, int(start), int(limit))
		repo.ListReturns(makeWorkItems(count), uint64(totalCount), nil)
		offset := strconv.Itoa(start)
		_, response := test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
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
	controller := NewWorkitem2Controller(svc, db)

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
	controller := NewWorkitem2Controller(svc, db)
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(100), uint64(100), nil)

	var offset string = "-1"
	var limit int = 2
	_, result := test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "0"
	limit = 0
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is 0", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "0"
	limit = -1
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "-3"
	limit = -1
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(t, "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "ALPHA"
	limit = 40
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
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
	controller := NewWorkitem2Controller(svc, db)

	offset := "10"
	limit := 10

	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
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
	controller := NewWorkitem2Controller(svc, db)

	offset := "0"
	var limit int
	repo := db.WorkItems().(*testsupport.WorkItemRepository)
	repo.ListReturns(makeWorkItems(10), uint64(100), nil)

	_, result := test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, nil, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(t, "Limit is nil", "Expected limit to be default size %d, got %v", 20, *result.Links.First)
	}
	limit = 1000
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=100") {
		assert.Fail(t, "Limit is more than max", "Expected limit to be %d, got %v", 100, *result.Links.First)
	}

	limit = 50
	_, result = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &limit, &offset)
	if !strings.Contains(*result.Links.First, "page[limit]=50") {
		assert.Fail(t, "Limit is within range", "Expected limit to be %d, got %v", 50, *result.Links.First)
	}
}

func generatePayloadBase(wi *app.WorkItem) *app.UpdateWorkItemJSONAPIPayload {
	return &app.UpdateWorkItemJSONAPIPayload{
		Data: &app.WorkItemDataForUpdate{
			Type: "workitems",
			ID:   wi.ID,
			Attributes: map[string]string{
				"version": strconv.Itoa(wi.Version),
			},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationshipBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   wi.Type,
					},
				},
			},
		},
	}
}

func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB) *account.Identity {
	newUUID := uuid.NewV4()
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		FullName: "Test User Integration Random",
		ImageURL: "http://images.com/42",
		ID:       newUUID,
	}
	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		fmt.Println("should not happen off.")
		return nil
	}
	return &identity
}

func TestUpdateWI2(t *testing.T) {
	resource.Require(t, resource.Database)
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("TestUpdateWI2-Service", almtoken.NewManager(pub, priv), account.TestIdentity)
	assert.NotNil(t, svc)
	controller := NewWorkitemController(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller)
	payload := app.CreateWorkItemPayload{
		Type: models.SystemBug,
		Fields: map[string]interface{}{
			models.SystemTitle: "Test WI",
			models.SystemState: "closed"},
	}

	_, wi := test.CreateWorkitemCreated(t, svc.Context, svc, controller, &payload)

	defer test.DeleteWorkitemOK(t, svc.Context, svc, controller, wi.ID)

	controller2 := NewWorkitem2Controller(svc, gormapplication.NewGormDB(DB))
	assert.NotNil(t, controller2)

	patchPayload := generatePayloadBase(wi)

	// update title attribute
	modifiedTitle := "Is the model updated?"
	patchPayload.Data.Attributes[models.SystemTitle] = modifiedTitle

	_, updatedWI := test.UpdateWorkitem2OK(t, svc.Context, svc, controller2, wi.ID, patchPayload)
	assert.Equal(t, updatedWI.Fields[models.SystemTitle], modifiedTitle)
	patchPayload.Data.Attributes["version"] = strconv.Itoa(updatedWI.Version)
	// update assignee relationship and verify
	newUser := createOneRandomUserIdentity(svc.Context, DB)
	assert.NotNil(t, newUser)

	patchPayload.Data.Relationships.Assignee = &app.RelationAssignee{
		Data: &app.AssigneeData{
			ID:   newUser.ID.String(),
			Type: "some_invalid_type_identities",
		},
	}

	test.UpdateWorkitem2BadRequest(t, svc.Context, svc, controller2, wi.ID, patchPayload)

	patchPayload.Data.Relationships.Assignee = &app.RelationAssignee{
		Data: &app.AssigneeData{
			ID:   newUser.ID.String(),
			Type: "identities",
		},
	}
	_, updatedWI = test.UpdateWorkitem2OK(t, svc.Context, svc, controller2, wi.ID, patchPayload)
	assert.Equal(t, updatedWI.Fields[models.SystemAssignee], newUser.ID.String())
	patchPayload.Data.Attributes["version"] = strconv.Itoa(updatedWI.Version)

	// update to wrong version
	patchPayload.Data.Attributes["version"] = "12453972348"
	test.UpdateWorkitem2BadRequest(t, svc.Context, svc, controller2, wi.ID, patchPayload)
}
