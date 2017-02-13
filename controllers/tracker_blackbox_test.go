package controllers_test

import (
	"bytes"
	"net/http"
	"testing"

	. "github.com/almighty/almighty-core"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/jsonapi"
	almtoken "github.com/almighty/almighty-core/token"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
)

func getTrackerTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(almtoken.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.Nil(t, err)

	createTrackerPayload := bytes.NewBuffer([]byte(`{"type": "github", "url": "https://api.github.com/"}`))

	return []testSecureAPI{
		// Create tracker API with different parameters
		{
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Update tracker API with different parameters
		{
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Delete tracker API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackers/088481764871",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func TestUnauthorizeTrackerCUD(t *testing.T) {
	UnauthorizeCreateUpdateDeleteTest(t, getTrackerTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTracker-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerController(service, gormapplication.NewGormDB(DB), RwiScheduler)
		app.MountTrackerController(service, controller)
		return nil
	})
}
