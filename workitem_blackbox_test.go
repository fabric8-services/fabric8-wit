package main_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"
	"time"

	"encoding/json"

	"golang.org/x/net/context"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/assert"
)

func TestGetWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	svc := goa.New("TestGetWorkItem-Service")
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

	_, result := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, wi.ID)
	}

	wi.Fields[models.SystemCreator] = "thomas"
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
	if updated.Fields[models.SystemCreator] != "thomas" {
		t.Errorf("expected creator %s, but got %s", "thomas", updated.Fields[models.SystemCreator])
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	resource.Require(t, resource.Database)
	svc := goa.New("TestCreateWI-Service")
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

	_, created := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func TestListByFields(t *testing.T) {
	resource.Require(t, resource.Database)
	svc := goa.New("TestListByFields-Service")
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

	_, wi := test.CreateWorkitemCreated(t, nil, nil, controller, &payload)

	filter := "{\"system.title\":\"run integration test\"}"
	page := "0,1"
	_, result := test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d", 1, len(result))
	}

	filter = "{\"system.creator\":\"aslak\"}"
	_, result = test.ListWorkitemOK(t, nil, nil, controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d ", 1, len(result))
	}

	test.DeleteWorkitemOK(t, nil, nil, controller, wi.ID)
}

func getExpiredAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{"exp": float64(time.Now().Unix() - 100)}
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

func getMalformedAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Malformed Bearer %s", tokenStr)
}

func getValidAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

// Expected strcture of 401 error response
type errorResponseStruct struct {
	Id     string
	Code   string
	Status int
	Detail string
}

// testSecureAPI defines how a Test object is.
type testSecureAPI struct {
	method             string
	url                string
	expectedStatusCode int    // this will be tested against responseRecorder.Code
	expectedErrorCode  string // this will be tested only if response body gets unmarshelled into errorResponseStruct
	payload            *bytes.Buffer
	jwtToken           string
}

func getWorkItemTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))

	createWIPayloadString := bytes.NewBuffer([]byte(`{"type": "system.userstory" ,"fields": { "system.creator": "tmaeder", "system.state": "new", "system.title": "My special story", "system.description": "description" }}`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             "POST",
			url:                "/api/workitems",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/workitems",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/workitems",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "POST",
			url:                "/api/workitems",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             "PUT",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "PUT",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "PUT",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "PUT",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             "DELETE",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "DELETE",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "DELETE",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "DELETE",
			url:                "/api/workitems/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createWIPayloadString,
			jwtToken:           "",
		},
		// Try fetching a random work Item
		// We do not have security on GET hence this should return 404 not found
		{
			method:             "GET",
			url:                "/api/workitems/088481764871",
			expectedStatusCode: 404,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeWorkItemCUD(t *testing.T) {
	resource.Require(t, resource.Database)

	// This will be modified after merge PR for "Viper Environment configurations"
	publickey, err := jwt.ParseRSAPublicKeyFromPEM((configuration.GetTokenPublicKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	tokenTests := getWorkItemTestData(t)

	for _, testObject := range tokenTests {
		// Build a request
		var req *http.Request
		var err error
		if testObject.payload == nil {
			req, err = http.NewRequest(testObject.method, testObject.url, nil)
		} else {
			req, err = http.NewRequest(testObject.method, testObject.url, testObject.payload)
		}
		// req, err := http.NewRequest(testObject.method, testObject.url, testObject.payload)
		if err != nil {
			t.Fatal("could not create a HTTP request")
		}
		// Add Authorization Header
		req.Header.Add("Authorization", testObject.jwtToken)

		rr := httptest.NewRecorder()

		// temperory service for testing the middleware
		service := goa.New("TestUnauthorizedCreateWI-Service")
		assert.NotNil(t, service)

		// if error is thrown during request processing, it will be caught by ErrorHandler middleware
		// this will put error code, status, details in recorder object.
		// e.g> {"id":"AL6spYb2","code":"jwt_security_error","status":401,"detail":"JWT validation failed: crypto/rsa: verification error"}
		service.Use(middleware.ErrorHandler(service, true))

		// append a middleware to service. Use appropriate RSA keys
		jwtMiddleware := goajwt.New(publickey, nil, app.NewJWTSecurity())
		// Adding middleware via "app" is important
		// Because it will check the design and accordingly apply the middleware if mentioned in design
		// But if I use `service.Use(jwtMiddleware)` then middleware is applied for all the requests (without checking design)
		app.UseJWTMiddleware(service, jwtMiddleware)

		controller := NewWorkitemController(service, gormapplication.NewGormDB(DB))
		app.MountWorkitemController(service, controller)

		// Hit the service with own request
		service.Mux.ServeHTTP(rr, req)

		assert.Equal(t, testObject.expectedStatusCode, rr.Code)

		// Below code tries to open Body response which is expected to be a JSON
		// If could not parse it correctly into errorResponseStruct
		// Then it gets logged and continue the test loop
		content := new(errorResponseStruct)
		err = json.Unmarshal(rr.Body.Bytes(), content)
		if err != nil {
			t.Log("Could not parse JSON response: ", rr.Body)
			// safe to continue because we alread checked rr.Code=required_value
			continue
		}
		// Additional checks for 'more' confirmation
		assert.Equal(t, testObject.expectedErrorCode, content.Code)
		assert.Equal(t, testObject.expectedStatusCode, content.Status)
	}
}

// this key will be used to sign the token but verification should
// fail as this is not the key used by server security layer
// ssh-keygen -f test-alm
var RSADifferentPrivateKeyTest = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsIT76Mr3p8VvtSrzCVcXEcyvalUp50mm4yvfqvZ1fZfbqAzJ
c4GNJEkpBGoXF+WgjLNkPnwS+k1YuqvPeGG4vFPtErF7nxNCHpzU+cXScOOl3WrM
S5fj928sBSJiDIDBwc98mQbIKaCrpLSsFe/kapV5mHmmWGAx6dqnObbqtIte4M7w
arE/c8xW1Fww2YZ4e4Xwknm+Rs2kQmg0SJPpgRih05y3snEQjXz1kR9bGTEBUPmX
HBTySgA93gmimQUlSAT0+hz9hcYrwCgjbnXUHlcBP2VbB4omJ7L1zJc/XMPwINR/
PtkGRhL/DXXA4v/0MLkYDXXmZGku/X1+du2ypQIDAQABAoIBAFi0m3si9E2FNFvQ
l42sDFXPjJ9c6M/n/UvP8niRnf1dYO8Ube/zvJ/tfAVR4wUJSiMqy0dzRn4ufFZi
nMIcKZ/KdSqdskgAf4uuuIBEXzqHzAR29O9QBymC3pY97xPlaHki8bRc6h2xNlBw
0sG7agf90btD9soWnT6tuLeSKmRLh5aHUQv3aGwzPyNfKHQ8J/KwKdPudP+tVBsi
eNd7DZDgSEw6pRaSCKS3ChrsQQ2XPjGo+OI6HjZ/LAFhFXMq2cRGELGF766a6phK
aCTB619AXiRHdKE98zEY3GMDtXSgeA0yzxcbvr224rEkHGDfkZ0BJwOCqCiaw4tZ
F/lFDMkCgYEA36Uqyj0cML5rMwC/W4b6ihuK/DujBBFYPQ8eVYt5yUSyLNJn5RLt
33eBUvgGB/FyAio5aCp49mcPtfFv5GKXpzTSYo/bWV1iy+oVwgPF7UA5gvtRw90w
NScLNsZ/7fOEpPJvlsKq/PQoMIoAjkegoj95cqM1yzC3aZpaAjx6188CgYEAyg58
5e5WK3zXICMpE8q+1AB+kJ/3UhQ71kpK4Xml0PtTJ7Bzqn0hiU4ThfpKj1n9PtpU
9Op3PqcfVjf11SA1tI5LRHQvgUSNppvf2hPgW8QrqRs5YFgNg0DkVXs3OxWIA4QA
Ko6oZu2ZpvK3TdYXRmcRUXXNyCDoSmJvH339N0sCgYB0g1kCmcm4/0tb+/S1m2Gl
V+oVtIAeG2csEFdOW+ar27Uzsr5b0nvI4zql3f9OXhR2WkckJJR2UoUV1d3kTxUR
EGzW2nl9WjChaafCNzMDgmUz/vi/INn/pwKpm8qETkz5njBSi8KHHDBf8VWOynQ+
cvEzryHUZOH5C2f/KEEbcwKBgQCGzVGgaPjOPJSdWTfPf4T+lXHa9Q4gkWU2WwxI
D0uD+BiLMxqH1MGqBA/cY5aYutXMuAbT+xUhFIhAkkcNMFcEJaarfcQvvteuHvIi
YP5e2qqyQHpv/27McV+kc/buEThT+B3QRqqtOLk4+1c1s66Fhr+0FB789I9lCPTQ
EtL7rwKBgQC5x7lGs+908uqf7yFXHzw7rPGFUe6cuxZ3jVOzovVoXRma+C7nroNx
/A4rWPqfpmiKcmrd7K4DQFlYhoq+MALEDmQm+/8G6j2inF53fRGgJVzaZhSvnO9X
CMnDipW5SU9AQE+xC8Zc+02rcyuZ7ha1WXKgIKwAa92jmJSCJjzdxA==
-----END RSA PRIVATE KEY-----`

func TestPaging(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := testsupport.NoTransactionSupport{}
	repo := &testsupport.WorkItemRepository{}
	svc := goa.New("TestListByFields-Service")
	assert.NotNil(t, svc)
	controller := NewWorkitem2Controller(svc, repo, ts)

	repo.ListReturns(makeWorkItems(5), 13, nil)
	page := "2,5"
	_, response := test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &page)
	assert.NotNil(t, response)
	assert.NotNil(t, response.Links.Prev)
	assert.NotNil(t, response.Links.Next)
	assert.NotNil(t, response.Links.Last)
	assert.True(t, strings.HasSuffix(*response.Links.Prev, "page=0,2"), *response.Links.Prev)
	assert.True(t, strings.HasSuffix(*response.Links.Next, "page=7,5"), *response.Links.Next)
	assert.True(t, strings.HasSuffix(*response.Links.Last, "page=12,1"), *response.Links.Last)

	repo.ListReturns(makeWorkItems(3), 13, nil)
	page = "10,3"
	_, response = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &page)
	assert.NotNil(t, response.Links.Prev)
	assert.Nil(t, response.Links.Next)
	assert.NotNil(t, response.Links.Last)

	assert.True(t, strings.HasSuffix(*response.Links.Prev, "page=7,3"), *response.Links.Prev)
	assert.True(t, strings.HasSuffix(*response.Links.Last, "page=10,3"), *response.Links.Last)

	repo.ListReturns(makeWorkItems(4), 13, nil)
	page = "0,4"
	_, response = test.ListWorkitem2OK(t, context.Background(), nil, controller, nil, &page)

	assert.Nil(t, response.Links.Prev)
	assert.NotNil(t, response.Links.Next)
	assert.NotNil(t, response.Links.Last)
	assert.True(t, strings.HasSuffix(*response.Links.Next, "page=4,4"), *response.Links.Next)
	assert.True(t, strings.HasSuffix(*response.Links.Last, "page=12,1"), *response.Links.Last)
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
