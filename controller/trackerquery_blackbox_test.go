package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	almtoken "github.com/almighty/almighty-core/token"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/require"
)

var trackerQueryBlackBoxTestConfiguration *config.ConfigurationData

func init() {
	var err error
	trackerQueryBlackBoxTestConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func getTrackerQueryTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(almtoken.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.Nil(t, err)

	createTrackerQueryPayload := bytes.NewBuffer([]byte(`{"type": "github", "url": "https://api.github.com/"}`))

	return []testSecureAPI{
		// Create tracker query API with different parameters
		{
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Update tracker query API with different parameters
		{
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Delete tracker query API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker query
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackerqueries/088481764871",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeTrackerQueryCUD(t *testing.T) {
	UnauthorizeCreateUpdateDeleteTest(t, getTrackerQueryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTrackerQuery-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerqueryController(service, gormapplication.NewGormDB(DB), RwiScheduler, trackerQueryBlackBoxTestConfiguration)
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func TestCreateTrackerQueryREST(t *testing.T) {
	resource.Require(t, resource.Database)

	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(almtoken.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}

	service := goa.New("API")

	controller := NewTrackerController(service, gormapplication.NewGormDB(DB), RwiScheduler, trackerQueryBlackBoxTestConfiguration)
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, tracker := test.CreateTrackerCreated(t, nil, nil, controller, &payload)

	jwtMiddleware := goajwt.New(&privatekey.PublicKey, nil, app.NewJWTSecurity())
	app.UseJWTMiddleware(service, jwtMiddleware)

	controller2 := NewTrackerqueryController(service, gormapplication.NewGormDB(DB), RwiScheduler, trackerQueryBlackBoxTestConfiguration)
	app.MountTrackerqueryController(service, controller2)

	server := httptest.NewServer(service.Mux)
	tqPayload := fmt.Sprintf(`{
		"query": "abcdefgh",
		"schedule": "1 1 * * * *",
		"trackerID": "%s",
		"relationships":{
			"space":{
				"data":{"id": "%s", "type": "spaces"},
				"links": {"self": "http://localhost:8080/api/spaces/%s"}
			}
		}
	}`, tracker.ID, space.SystemSpace, space.SystemSpace)
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
