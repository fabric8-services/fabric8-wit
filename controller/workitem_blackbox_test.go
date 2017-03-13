package controller_test

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/codebase"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/rendering"
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
)

var wibConfiguration *config.ConfigurationData

func init() {
	var err error
	wibConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestSuiteWorkItem1(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemSuite))
}

type WorkItemSuite struct {
	suite.Suite
	db             *gorm.DB
	clean          func()
	controller     app.WorkitemController
	pubKey         *rsa.PublicKey
	priKey         *rsa.PrivateKey
	svc            *goa.Service
	wi             *app.WorkItem2
	minimumPayload *app.UpdateWorkitemPayload
	testIdentity   account.Identity
	ctx            context.Context
}

func (s *WorkItemSuite) SetupSuite() {
	var err error
	s.db, err = gorm.Open("postgres", wibConfiguration.GetPostgresConfigString())
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	s.priKey, _ = almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if wibConfiguration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.db, func(tx *gorm.DB) error {
			s.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(s.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	s.clean = cleaner.DeleteCreatedEntities(s.db)

	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.db, "test user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = testIdentity
}

func (s *WorkItemSuite) TearDownSuite() {
	s.clean()
	if s.db != nil {
		s.db.Close()
	}
}

func (s *WorkItemSuite) SetupTest() {
	s.svc = testsupport.ServiceAsUser("TestUpdateWI-Service", almtoken.NewManagerWithPrivateKey(s.priKey), s.testIdentity)
	s.controller = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.db))
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	s.wi = wi.Data
	s.minimumPayload = getMinimumRequiredUpdatePayload(s.wi)
}

func (s *WorkItemSuite) TestGetWorkItemWithLegacyDescription() {
	// given
	_, wi := test.ShowWorkitemOK(s.T(), nil, nil, s.controller, *s.wi.ID)
	require.NotNil(s.T(), wi)
	assert.Equal(s.T(), s.wi.ID, wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Attributes[workitem.SystemCreatedAt])
	assert.Equal(s.T(), s.testIdentity.ID.String(), *wi.Data.Relationships.Creator.Data.ID)
	// when
	wi.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	updatedDescription := "= Updated Test WI description"
	wi.Data.Attributes[workitem.SystemDescription] = updatedDescription
	payload2 := minimumRequiredUpdatePayload()
	payload2.Data.ID = wi.Data.ID
	payload2.Data.Attributes = wi.Data.Attributes
	_, updated := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, *wi.Data.ID, &payload2)
	// then
	assert.NotNil(s.T(), updated.Data.Attributes[workitem.SystemCreatedAt])
	assert.Equal(s.T(), (s.wi.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(s.T(), *s.wi.ID, *updated.Data.ID)
	assert.Equal(s.T(), wi.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), updatedDescription, updated.Data.Attributes[workitem.SystemDescription])
}

func (s *WorkItemSuite) TestCreateWI() {
	// given
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when
	_, created := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// then
	require.NotNil(s.T(), created.Data.ID)
	assert.NotEmpty(s.T(), *created.Data.ID)
	assert.NotNil(s.T(), created.Data.Attributes[workitem.SystemCreatedAt])
	assert.NotNil(s.T(), created.Data.Relationships.Creator.Data)
	assert.Equal(s.T(), *created.Data.Relationships.Creator.Data.ID, s.testIdentity.ID.String())
}

// TestReorderAbove is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result3 and places it **above** result2
func (s *WorkItemSuite) TestReorderWorkitemAboveOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	// This workitem is created but not used to clearly test that the reorder workitem is moved between **two** workitems i.e. result1 and result2 and not to the **top** of the list
	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)

	_, result2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result3 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem2 // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result3.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionAbove)
	_, reordered1 := test.ReorderWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result3.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result3.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderBelow is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result1 and places it **below** result1
func (s *WorkItemSuite) TestReorderWorkitemBelowOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	_, result1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)

	// This workitem is created but not used to clearly demonstrate that the reorder workitem is moved between **two** workitems i.e. result2 and result3 and not to the **bottom** of the list
	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)

	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem2 // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionBelow)

	_, reordered1 := test.ReorderWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result1.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result1.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderTop is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result2 and places it to the top of the list
func (s *WorkItemSuite) TestReorderWorkitemTopOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	// There are two workitems in the list -> result1 and result2
	// In this case, we reorder result2 to the top of the list i.e. above result1
	_, result1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem2 // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	payload2.Position.Direction = string(workitem.DirectionTop)
	_, reordered1 := test.ReorderWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result1.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result1.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderBottom is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result1 and places it to the bottom of the list
func (s *WorkItemSuite) TestReorderWorkitemBottomOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	// There are two workitems in the list -> result1 and result2
	// In this case, we reorder result1 to the bottom of the list i.e. below result2
	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem2 // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result2.Data)
	payload2.Data = dataArray
	payload2.Position.Direction = string(workitem.DirectionBottom)

	_, reordered1 := test.ReorderWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result2.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result2.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderMultipleWorkitem is positive test which tests successful reorder by providing valid input
// This case reorders two workitems -> result3 and result4 and places them above result2
func (s *WorkItemSuite) TestReorderMultipleWorkitems() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result3 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	_, result4 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem2 // dataArray contains the workitems that have to be reordered
	dataArray = append(dataArray, result3.Data, result4.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionAbove)

	_, reordered1 := test.ReorderWorkitemOK(s.T(), s.svc.Context, s.svc, s.controller, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 2) // checks the correct number of workitems reordered

	assert.Equal(s.T(), result3.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), result4.Data.Attributes["version"].(int)+1, reordered1.Data[1].Attributes["version"])

	assert.Equal(s.T(), *result3.Data.ID, *reordered1.Data[0].ID)
	assert.Equal(s.T(), *result4.Data.ID, *reordered1.Data[1].ID)
}

// TestReorderWorkitemBadRequest is negative test which tests unsuccessful reorder by providing invalid input
func (s *WorkItemSuite) TestReorderWorkitemBadRequestOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	_, result1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	// This case gives empty dataArray as input
	// Response is Bad Parameter
	// Reorder is unsuccessful

	var dataArray []*app.WorkItem2
	payload2.Data = dataArray
	payload2.Position.ID = result1.Data.ID
	payload2.Position.Direction = string(workitem.DirectionAbove)
	test.ReorderWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.controller, &payload2)
}

// TestReorderWorkitemNotFound is negative test which tests unsuccessful reorder by providing invalid input
func (s *WorkItemSuite) TestReorderWorkitemNotFoundOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	_, result1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	payload2 := minimumRequiredReorderPayload()

	// This case gives id of workitem in position.ID which is not present in db as input
	// Response is Not Found
	// Reorder is unsuccessful

	var dataArray []*app.WorkItem2
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	randomID := "78"
	payload2.Position.ID = &randomID
	payload2.Position.Direction = string(workitem.DirectionAbove)
	test.ReorderWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.controller, &payload2)
}

func (s *WorkItemSuite) TestCreateWorkItemWithoutContext() {
	// given
	s.svc = goa.New("TestCreateWorkItemWithoutContext-Service")
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when/then
	test.CreateWorkitemUnauthorized(s.T(), s.svc.Context, s.svc, s.controller, &payload)
}

func (s *WorkItemSuite) TestListByFields() {
	// given
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "run integration test"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// when
	filter := "{\"system.title\":\"run integration test\"}"
	offset := "0"
	limit := 1
	_, result := test.ListWorkitemOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, nil, nil, &limit, &offset)
	// then
	require.NotNil(s.T(), result)
	require.Equal(s.T(), 1, len(result.Data))
	// when
	filter = fmt.Sprintf("{\"system.creator\":\"%s\"}", s.testIdentity.ID.String())
	// then
	_, result = test.ListWorkitemOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, nil, nil, &limit, &offset)
	require.NotNil(s.T(), result)
	require.Equal(s.T(), 1, len(result.Data))
}

func getWorkItemTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((wibConfiguration.GetTokenPrivateKey()))
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
func (s *WorkItemSuite) TestUnauthorizeWorkItemCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWI-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemController(service, gormapplication.NewGormDB(s.db))
		app.MountWorkitemController(service, controller)
		return nil
	})
}

func createPagingTest(t *testing.T, ctx context.Context, controller *WorkitemController, repo *testsupport.WorkItemRepository, totalCount int) func(start int, limit int, first string, last string, prev string, next string) {
	return func(start int, limit int, first string, last string, prev string, next string) {
		count := computeCount(totalCount, int(start), int(limit))
		repo.ListReturns(makeWorkItems(count), uint64(totalCount), nil)
		offset := strconv.Itoa(start)

		_, response := test.ListWorkitemOK(t, ctx, nil, controller, nil, nil, nil, nil, nil, nil, &limit, &offset)
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

	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	for index := range res {
		res[index] = &app.WorkItem{
			ID:   fmt.Sprintf("id%d", index),
			Type: uuid.NewV4(), // used to be "foobar"
			Fields: map[string]interface{}{
				workitem.SystemUpdatedAt: time.Now(),
			},
			Relationships: &app.WorkItemRelationships{
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		}
	}
	return res
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
			Relationships: wi.Relationships,
		},
	}
}

func minimumRequiredUpdatePayload() app.UpdateWorkitemPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	return app.UpdateWorkitemPayload{
		Data: &app.WorkItem2{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
}

func minimumRequiredReorderPayload() app.ReorderWorkitemPayload {
	return app.ReorderWorkitemPayload{
		Data: []*app.WorkItem2{},
		Position: &app.WorkItemReorderPosition{
			ID: nil,
		},
	}
}

func minimumRequiredCreateWithType(wit uuid.UUID) app.CreateWorkitemPayload {
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
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))

	return app.CreateWorkitemPayload{
		Data: &app.WorkItem2{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
}

func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB) *account.Identity {
	newUserUUID := uuid.NewV4()
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		Username: "Test User Integration Random",
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
	spaceRepo := space.NewRepository(db)

	newSpace := space.Space{
		Name: "Space iteration",
	}
	space, err := spaceRepo.Create(ctx, &newSpace)
	if err != nil {
		fmt.Println("Failed to create space for iteration.")
		return nil
	}

	itr := iteration.Iteration{
		Name:    "Sprint 101",
		SpaceID: space.ID,
	}
	err = iterationRepo.Create(ctx, &itr)
	if err != nil {
		fmt.Println("Failed to create iteration.")
		return nil
	}
	return &itr
}

func createOneRandomArea(ctx context.Context, db *gorm.DB) *area.Area {
	areaRepo := area.NewAreaRepository(db)
	spaceRepo := space.NewRepository(db)

	newSpace := space.Space{
		Name: "Space area",
	}
	space, err := spaceRepo.Create(ctx, &newSpace)
	if err != nil {
		fmt.Println("Failed to create space for area.")
		return nil
	}
	ar := area.Area{
		Name:    "Area 51",
		SpaceID: space.ID,
	}
	err = areaRepo.Create(ctx, &ar)
	if err != nil {
		fmt.Println("Failed to create area.")
		return nil
	}
	return &ar
}

// ========== WorkItem2Suite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
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

type WorkItem2Suite struct {
	suite.Suite
	db             *gorm.DB
	clean          func()
	wiCtrl         app.WorkitemController
	wi2Ctrl        app.WorkitemController
	linkCtrl       app.WorkItemLinkController
	linkCatCtrl    app.WorkItemLinkCategoryController
	linkTypeCtrl   app.WorkItemLinkTypeController
	spaceCtrl      app.SpaceController
	pubKey         *rsa.PublicKey
	priKey         *rsa.PrivateKey
	svc            *goa.Service
	wi             *app.WorkItem2
	minimumPayload *app.UpdateWorkitemPayload
	ctx            context.Context
}

func (s *WorkItem2Suite) SetupSuite() {
	var err error
	s.db, err = gorm.Open("postgres", wibConfiguration.GetPostgresConfigString())
	if err != nil {
		panic("Failed to connect database: " + err.Error())
	}
	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if wibConfiguration.GetPopulateCommonTypes() {
		if err := models.Transactional(s.db, func(tx *gorm.DB) error {
			s.ctx = migration.NewMigrationContext(context.Background())
			return migration.PopulateCommonTypes(s.ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	s.clean = cleaner.DeleteCreatedEntities(s.db)

	// create identity
	testIdentity, err := testsupport.CreateTestIdentity(s.db, "test user", "test provider")
	require.Nil(s.T(), err)
	s.priKey, _ = almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("TestUpdateWI2-Service", almtoken.NewManagerWithPrivateKey(s.priKey), testIdentity)
}

func (s *WorkItem2Suite) TearDownSuite() {
	s.clean()
	if s.db != nil {
		s.db.Close()
	}
}

func (s *WorkItem2Suite) SetupTest() {
	s.wiCtrl = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.db))
	s.wi2Ctrl = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.db))
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.db))
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.db))
	s.linkCtrl = NewWorkItemLinkController(s.svc, gormapplication.NewGormDB(s.db))
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.db))

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wiCtrl, &payload)
	s.wi = wi.Data
	s.minimumPayload = getMinimumRequiredUpdatePayload(s.wi)
	//s.minimumReorderPayload = getMinimumRequiredReorderPayload(s.wi)
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
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription.ToMap()
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
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription.ToMap()

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
			{
				ID:   &maliciousUUID,
				Type: &userType,
			}},
	}
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *s.wi.ID, s.minimumPayload)

	s.minimumPayload.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				Type: &userType,
				ID:   &newUserId,
			}},
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
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
	spaceRelation := update.Data.Relationships.Space
	update.Data.Relationships = &app.WorkItemRelationships{
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser2.ID),
				ident(newUser3.ID),
			},
		},
		Space: spaceRelation,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
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
	_, list := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, nil, &newUserID, nil, nil, nil, nil, nil)
	assert.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *list.Data[0].Relationships.Assignees.Data[0].ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[assignee]"))
}

func (s *WorkItem2Suite) TestWI2ListByWorkitemtypeFilter() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	// when
	_, expected := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	// then
	assert.NotNil(s.T(), expected.Data)
	require.NotNil(s.T(), expected.Data.ID)
	require.NotNil(s.T(), expected.Data.Type)
	_, actual := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, nil, nil, nil, nil, &workitem.SystemBug, nil, nil)
	require.NotNil(s.T(), actual)
	require.True(s.T(), len(actual.Data) > 1)
	assert.Contains(s.T(), *actual.Links.First, fmt.Sprintf("filter[workitemtype]=%s", workitem.SystemBug))
	for _, actualWI := range actual.Data {
		assert.Equal(s.T(), expected.Data.Type, actualWI.Type)
		require.NotNil(s.T(), actualWI.ID)
	}
}

func (s *WorkItem2Suite) TestWI2ListByWorkitemstateFilter() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	l := minimumRequiredCreatePayload()
	l.Data.Attributes[workitem.SystemTitle] = "Title"
	l.Data.Attributes[workitem.SystemState] = workitem.SystemStateInProgress
	l.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	// when
	_, expected := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, notExpected := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &l)
	// then
	assert.NotNil(s.T(), expected.Data)
	require.NotNil(s.T(), expected.Data.ID)
	require.NotNil(s.T(), expected.Data.Type)
	require.NotNil(s.T(), expected.Data.Attributes)
	var dataArray []*app.WorkItem2Single
	dataArray = append(dataArray, notExpected)
	dataArray = append(dataArray, expected)
	wiNew := workitem.SystemStateNew
	// var foundExpected bool
	_, actual := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, nil, nil, nil, &wiNew, nil, nil, nil)

	require.NotNil(s.T(), actual)
	require.True(s.T(), len(actual.Data) > 1)
	assert.Contains(s.T(), *actual.Links.First, fmt.Sprintf("filter[workitemstate]=%s", workitem.SystemStateNew))
	for _, actualWI := range actual.Data {
		assert.Equal(s.T(), expected.Data.Attributes[workitem.SystemState], actualWI.Attributes[workitem.SystemState])
		require.NotNil(s.T(), actualWI.Attributes[workitem.SystemState])
		// if *expected.Data.ID == *actualWI.ID {
		// 	foundExpected = true
		// }
	}
	// assert.True(s.T(), foundExpected, "did not find expected work item in filtered list response")
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilter() {
	tempArea := createOneRandomArea(s.svc.Context, s.db)
	require.NotNil(s.T(), tempArea)
	areaID := tempArea.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: APIStringTypeWorkItemType,
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Area = &app.RelationGeneric{
		Data: &app.GenericData{
			ID: &areaID,
		},
	}
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	require.NotNil(s.T(), wi.Data.Relationships.Area)
	assert.Equal(s.T(), areaID, *wi.Data.Relationships.Area.Data.ID)

	_, list := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, &areaID, nil, nil, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), areaID, *list.Data[0].Relationships.Area.Data.ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[area]"))
}

func (s *WorkItem2Suite) TestWI2ListByIterationFilter() {
	tempIteration := createOneRandomIteration(s.svc.Context, s.db)
	require.NotNil(s.T(), tempIteration)
	iterationID := tempIteration.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: APIStringTypeWorkItemType,
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			ID: &iterationID,
		},
	}
	_, wi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	assert.Equal(s.T(), iterationID, *wi.Data.Relationships.Iteration.Data.ID)

	_, list := test.ListWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, nil, nil, nil, &iterationID, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), iterationID, *list.Data[0].Relationships.Iteration.Data.ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[iteration]"))
}

func (s *WorkItem2Suite) TestWI2FailCreateInvalidAssignees() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(uuid.NewV4()),
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
			ident(newUser2.ID),
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	test.DeleteWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	test.ShowWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
}

// TestWI2DeleteLinksOnWIDeletionOK creates two work items (WI1 and WI2) and
// creates a link between them. When one of the work items is deleted, the
// link shall be gone as well.
func (s *WorkItem2Suite) TestWI2DeleteLinksOnWIDeletionOK() {
	// Create two work items (wi1 and wi2)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "WI1"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, wi1 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(s.T(), wi1)
	c.Data.Attributes[workitem.SystemTitle] = "WI2"
	_, wi2 := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(s.T(), wi2)

	// Create link category
	linkCatPayload := CreateWorkItemLinkCategory("test-user")
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)

	// Create link space
	spacePayload := CreateSpacePayload("test-space", "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, spacePayload)

	// Create work item link type payload
	linkTypePayload := CreateWorkItemLinkType("MyLinkType", workitem.SystemBug, workitem.SystemBug, *linkCat.Data.ID, *space.Data.ID)
	_, linkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Create link between wi1 and wi2
	id1, err := strconv.ParseUint(*wi1.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	id2, err := strconv.ParseUint(*wi2.Data.ID, 10, 64)
	require.Nil(s.T(), err)
	linkPayload := CreateWorkItemLink(id1, id2, *linkType.Data.ID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.linkCtrl, linkPayload)
	require.NotNil(s.T(), workItemLink)

	// Delete work item wi1
	test.DeleteWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *wi1.Data.ID)

	// Check that the link was deleted by deleting wi1
	test.ShowWorkItemLinkNotFound(s.T(), s.svc.Context, s.svc, s.linkCtrl, *workItemLink.Data.ID)

	// Check that we can query for wi2 without problems
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *wi2.Data.ID)
}

func (s *WorkItem2Suite) TestWI2FailMissingDelete() {
	test.DeleteWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, "00000000")
}

func (s *WorkItem2Suite) TestWI2CreateWithArea() {
	t := s.T()

	areaInstance := createSpaceAndArea(t, gormapplication.NewGormDB(s.db))
	areaID := areaInstance.ID.String()
	arType := area.APIStringTypeAreas

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		Area: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &arType,
				ID:   &areaID,
			},
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Area)
	assert.Equal(t, areaID, *wi.Data.Relationships.Area.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateWithArea() {
	t := s.T()

	areaInstance := createSpaceAndArea(t, gormapplication.NewGormDB(s.db))
	areaID := areaInstance.ID.String()
	arType := area.APIStringTypeAreas
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Area)
	assert.Nil(t, wi.Data.Relationships.Area.Data)

	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Area: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &arType,
				ID:   &areaID,
			},
		},
	}

	_, wiu := test.UpdateWorkitemOK(t, s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID, &u)
	require.NotNil(t, wiu.Data.Relationships.Area)
	require.NotNil(t, wiu.Data.Relationships.Area.Data)
	assert.Equal(t, areaID, *wiu.Data.Relationships.Area.Data.ID)
	assert.Equal(t, arType, *wiu.Data.Relationships.Area.Data.Type)
}

func (s *WorkItem2Suite) TestWI2CreateUnknownArea() {
	t := s.T()

	arType := area.APIStringTypeAreas
	areaID := uuid.NewV4().String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Area = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &arType,
			ID:   &areaID,
		},
	}
	test.CreateWorkitemBadRequest(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestWI2CreateWithIteration() {
	t := s.T()

	iterationInstance := createSpaceAndIteration(t, gormapplication.NewGormDB(s.db))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))

	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: &app.RelationBaseType{
			Data: &app.BaseTypeData{
				Type: "workitemtypes",
				ID:   workitem.SystemBug,
			},
		},
		Space: space.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Iteration)
	assert.Nil(t, wi.Data.Relationships.Iteration.Data)

	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
		},
	}
	_, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	assert.NotNil(t, wi.Data.Relationships.Iteration)
	assert.NotNil(t, wi.Data.Relationships.Iteration.Data)

	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: nil,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
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
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	require.NotNil(s.T(), fetchedWi.Data)
	require.NotNil(s.T(), fetchedWi.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWi.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), "<p>"+html.EscapeString(description.Content)+"</p>\n", fetchedWi.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestCreateWIWithCodebase() {
	t := s.T()
	c := minimumRequiredCreatePayload()
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemPlannerItem,
		},
	}
	branch := "earth-recycle-101"
	repo := "golang-project"
	file := "main.go"
	line := 200
	cbase := codebase.CodebaseContent{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	_, createdWi := test.CreateWorkitemCreated(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(t, createdWi)
	_, fetchedWi := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.wi2Ctrl, *createdWi.Data.ID)
	require.NotNil(t, fetchedWi.Data)
	require.NotNil(t, fetchedWi.Data.Attributes)
	assert.Equal(t, title, fetchedWi.Data.Attributes[workitem.SystemTitle])
	cb := fetchedWi.Data.Attributes[workitem.SystemCodebase].(codebase.CodebaseContent)
	assert.Equal(t, repo, cb.Repository)
	assert.Equal(t, branch, cb.Branch)
	assert.Equal(t, file, cb.FileName)
	assert.Equal(t, line, cb.LineNumber)

	// TODO: Uncomment following block that tests DO-IT URL
	// require.NotNil(t, fetchedWi.Data.Links)
	// expectedURL := fmt.Sprintf("/codebase/generate?repo=%s&branch=%s&file=%s&line=%d", cb.Repository, cb.Branch, cb.FileName, cb.LineNumber)
	// expectedURL = url.QueryEscape(expectedURL)
	// assert.Contains(t, *fetchedWi.Data.Links.Doit, expectedURL)
}

func (s *WorkItem2Suite) TestFailToCreateWIWithCodebase() {
	t := s.T()
	c := minimumRequiredCreatePayload()
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemPlannerItem,
		},
	}
	branch := "earth-recycle-101"
	cbase := codebase.CodebaseContent{
		Branch: branch,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	test.CreateWorkitemBadRequest(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithDefaultSpace() {
	t := s.T()
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// remove Space relation and see if WI gets default space.
	c.Data.Relationships.Space = nil
	_, item := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(t, item)
	assert.Equal(t, title, item.Data.Attributes[workitem.SystemTitle])
	require.NotNil(t, item.Data.Relationships)
	require.NotNil(t, item.Data.Relationships.Space)
	assert.Equal(t, space.SystemSpace, *item.Data.Relationships.Space.Data.ID)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithCustomSpace() {
	t := s.T()
	spaceName := "My own Space " + uuid.NewV4().String()
	sp := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name: &spaceName,
			},
		},
	}
	_, customSpace := test.CreateSpaceCreated(t, s.svc.Context, s.svc, s.spaceCtrl, sp)
	require.NotNil(t, customSpace)
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// set custom space and see if WI gets custom space
	c.Data.Relationships.Space.Data.ID = customSpace.Data.ID
	_, item := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
	require.NotNil(t, item)
	assert.Equal(t, title, item.Data.Attributes[workitem.SystemTitle])
	require.NotNil(t, item.Data.Relationships)
	require.NotNil(t, item.Data.Relationships.Space)
	assert.Equal(t, *customSpace.Data.ID, *item.Data.Relationships.Space.Data.ID)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithInvalidSpace() {
	t := s.T()
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// set custom space and see if WI gets custom space
	fakeSpaceID := uuid.NewV4()
	c.Data.Relationships.Space.Data.ID = &fakeSpaceID
	test.CreateWorkitemBadRequest(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)
}

//Ignore, middlewares not respected by the generated test framework. No way to modify Request?
// Require full HTTP request access.
func (s *WorkItem2Suite) xTestWI2IfModifiedSince() {
	t := s.T()

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

	resp, wi := test.CreateWorkitemCreated(t, s.svc.Context, s.svc, s.wi2Ctrl, &c)

	lastMod := resp.Header().Get("Last-Modified")
	s.svc.Use(func(handler goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
			req.Header.Set("If-Modified-Since", lastMod)
			return nil
		}
	})
	test.ShowWorkitemNotModified(t, s.svc.Context, s.svc, s.wi2Ctrl, *wi.Data.ID)
}
