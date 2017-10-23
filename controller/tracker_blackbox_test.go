package controller_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
)

type TestTrackerREST struct {
	gormtestsupport.DBTestSuite
	RwiScheduler *remoteworkitem.Scheduler
	db           *gormapplication.GormDB
	clean        func()
}

func TestRunTrackerREST(t *testing.T) {
	suite.Run(t, &TestTrackerREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestTrackerREST) SetupTest() {
	rest.RwiScheduler = remoteworkitem.NewScheduler(rest.DB)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestTrackerREST) TearDownTest() {
	rest.clean()
}

func (rest *TestTrackerREST) SecuredController() (*goa.Service, *TrackerController) {
	svc := testsupport.ServiceAsUser("Tracker-Service", testsupport.TestIdentity)
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func (rest *TestTrackerREST) UnSecuredController() (*goa.Service, *TrackerController) {
	svc := goa.New("Tracker-Service")
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

// This test case will check authorized access to Create/Update/Delete APIs
func (rest *TestTrackerREST) TestUnauthorizeTrackerCUD() {
	UnauthorizeCreateUpdateDeleteTest(rest.T(), getTrackerTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTracker-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerController(service, rest.db, rest.RwiScheduler, rest.Configuration)
		app.MountTrackerController(service, controller)
		return nil
	})
}

func getTrackerTestData(t *testing.T) []testSecureAPI {
	privatekey := testtoken.PrivateKey()
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
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Delete tracker API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackers/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

func (rest *TestTrackerREST) TestCreateTracker() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://issues.jboss.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}

	_, created := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)
	assert.NotNil(rest.T(), created)
}

func (rest *TestTrackerREST) TestUpdateTracker() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://issues.jboss.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}

	_, result := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)
	_, tr := test.ShowTrackerOK(t, svc.Context, svc, ctrl, *result.Data.ID)
	assert.NotNil(rest.T(), tr)
	assert.Equal(rest.T(), result.Data.ID, tr.Data.ID)

	payload2 := app.UpdateTrackerPayload{
		Data: &app.Tracker{
			ID: result.Data.ID,
			Attributes: &app.TrackerAttributes{
				URL:  "http://issues.jboss.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, updated := test.UpdateTrackerOK(t, svc.Context, svc, ctrl, tr.Data.ID.String(), &payload2)
	assert.NotNil(rest.T(), updated)
	assert.Equal(rest.T(), result.Data.ID, updated.Data.ID)
	assert.Equal(rest.T(), result.Data.Attributes.URL, updated.Data.Attributes.URL)
	assert.Equal(rest.T(), result.Data.Attributes.Type, updated.Data.Attributes.Type)
}

// This test ensures that List does not return NIL items.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/191
func (rest *TestTrackerREST) TestTrackerListItemsNotNil() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://issues.jboss.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)

	test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)

	_, list := test.ListTrackerOK(t, svc.Context, svc, ctrl, nil, nil)

	for _, tracker := range list.Data {
		if tracker == nil {
			t.Error("Returned Tracker found nil")
		}
	}
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (rest *TestTrackerREST) TestCreateTrackerValidId() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://issues.jboss.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, tracker := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)
	_, created := test.ShowTrackerOK(t, svc.Context, svc, ctrl, *tracker.Data.ID)
	require.NotNil(t, created.Data)
	require.Equal(t, *tracker.Data.ID, *created.Data.ID)
}
