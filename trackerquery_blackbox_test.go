package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/resource"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/assert"
)

func getTrackerQueryTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))

	createTrackerQueryPayload := bytes.NewBuffer([]byte(`{"type": "github", "url": "https://api.github.com/"}`))

	return []testSecureAPI{
		// Create tracker query API with different parameters
		{
			method:             "POST",
			url:                "/api/trackerqueries",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/trackerqueries",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "POST",
			url:                "/api/trackerqueries",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "POST",
			url:                "/api/trackerqueries",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Update tracker query API with different parameters
		{
			method:             "PUT",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "PUT",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "PUT",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "PUT",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Delete tracker query API with different parameters
		{
			method:             "DELETE",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             "DELETE",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             "DELETE",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             "DELETE",
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: 401,
			expectedErrorCode:  "jwt_security_error",
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker query
		// We do not have security on GET hence this should return 404 not found
		{
			method:             "GET",
			url:                "/api/trackerqueries/088481764871",
			expectedStatusCode: 404,
			expectedErrorCode:  "not_found",
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeTrackerQueryCUD(t *testing.T) {
	resource.Require(t, resource.Database)

	publickey, err := jwt.ParseRSAPublicKeyFromPEM((configuration.GetTokenPublicKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	tokenTests := getTrackerQueryTestData(t)

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
		service := goa.New("TestUnauthorizedTrackerQuery-Service")
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

		controller := NewTrackerqueryController(service, gormapplication.NewGormDB(DB), RwiScheduler)
		app.MountTrackerqueryController(service, controller)

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

func TestCreateTrackerQueryREST(t *testing.T) {
	resource.Require(t, resource.Database)

	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((configuration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}

	service := goa.New("API")

	controller := NewTrackerController(service, gormapplication.NewGormDB(DB), RwiScheduler)
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, tracker := test.CreateTrackerCreated(t, nil, nil, controller, &payload)

	publickey, err := jwt.ParseRSAPublicKeyFromPEM((configuration.GetTokenPublicKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	jwtMiddleware := goajwt.New(publickey, nil, app.NewJWTSecurity())
	app.UseJWTMiddleware(service, jwtMiddleware)

	controller2 := NewTrackerqueryController(service, gormapplication.NewGormDB(DB), RwiScheduler)
	app.MountTrackerqueryController(service, controller2)

	server := httptest.NewServer(service.Mux)
	tqPayload := fmt.Sprintf(`{"query": "abcdefgh", "schedule": "1 1 * * * *", "trackerID": "%s"}`, tracker.ID)
	trackerQueryCreateURL := "/api/trackerqueries"
	req, _ := http.NewRequest("POST", server.URL+trackerQueryCreateURL, strings.NewReader(tqPayload))

	jwtToken := getValidAuthHeader(t, privatekey)
	req.Header.Set("Authorization", jwtToken)
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Server error %s", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected a 201 Created response, got %d", res.StatusCode)
	}

	server.Close()
}
