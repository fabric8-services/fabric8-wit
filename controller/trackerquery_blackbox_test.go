package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	uuid "github.com/satori/go.uuid"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/goadesign/goa"
)

type TestTrackerQueryREST struct {
	gormtestsupport.DBTestSuite
	RwiScheduler *remoteworkitem.Scheduler
	db           *gormapplication.GormDB
}

func TestRunTrackerQueryREST(t *testing.T) {
	suite.Run(t, &TestTrackerQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestTrackerQueryREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.RwiScheduler = remoteworkitem.NewScheduler(rest.DB)
	rest.db = gormapplication.NewGormDB(rest.DB)
}

func (rest *TestTrackerQueryREST) SecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := testsupport.ServiceAsUser("Tracker-Service", testsupport.TestIdentity)
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func (rest *TestTrackerQueryREST) UnSecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := goa.New("Tracker-Service")
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func getTrackerQueryTestData(t *testing.T) []testSecureAPI {
	privatekey := testtoken.PrivateKey()
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
func (rest *TestTrackerQueryREST) TestUnauthorizeTrackerQueryCUD() {
	UnauthorizeCreateUpdateDeleteTest(rest.T(), getTrackerQueryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTrackerQuery-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerqueryController(service, rest.db, rest.RwiScheduler, rest.Configuration)
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func (rest *TestTrackerQueryREST) TestCreateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://api.github.com",
				Type: remoteworkitem.ProviderJira,
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)
	t.Log(result.Data.ID)

	tqpayload := newCreateTrackerQueryPayload(*result.Data.ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	assert.NotNil(rest.T(), tqresult)
}

func (rest *TestTrackerQueryREST) TestGetTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://api.github.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)

	tqpayload := newCreateTrackerQueryPayload(*result.Data.ID)

	fmt.Printf("tq payload %#v", tqpayload)
	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, *tqresult.Data.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, *tqresult.Data.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.Data.ID)
	}
	if tqr.Data.ID != tqresult.Data.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.Data.ID, tqr.Data.ID)
	}
}

func (rest *TestTrackerQueryREST) TestUpdateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://api.github.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)

	tqpayload := newCreateTrackerQueryPayload(*result.Data.ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, *tqresult.Data.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, *tqresult.Data.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.Data.ID)
	}
	if tqr.Data.ID != tqresult.Data.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.Data.ID, tqr.Data.ID)
	}

	spaceID := space.SystemSpace.String()
	trackerID := result.Data.ID.String()
	payload2 := app.UpdateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			Attributes: &app.TrackerQueryAttributes{
				Query:    tqr.Data.Attributes.Query,
				Schedule: tqr.Data.Attributes.Schedule,
			},
			Relationships: &app.TrackerQueryRelations{
				Space: &app.RelationGeneric{
					Data: &app.GenericData{
						ID: &spaceID,
					},
				},
				Tracker: &app.RelationGeneric{
					Data: &app.GenericData{
						ID: &trackerID,
					},
				},
			},
		},
	}

	_, updated := test.UpdateTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqr.Data.ID.String(), &payload2)

	if updated.Data.ID != tqresult.Data.ID {
		t.Errorf("Id has changed from %s to %s", tqresult.Data.ID, updated.Data.ID)
	}
	if updated.Data.Attributes.Query != tqresult.Data.Attributes.Query {
		t.Errorf("Query has changed from %s to %s", tqresult.Data.Attributes.Query, updated.Data.Attributes.Query)
	}
	if updated.Data.Attributes.Schedule != tqresult.Data.Attributes.Schedule {
		t.Errorf("Type has changed has from %s to %s", tqresult.Data.Attributes.Schedule, updated.Data.Attributes.Schedule)
	}
}

// This test ensures that List does not return NIL items.
func (rest *TestTrackerQueryREST) TestTrackerQueryListItemsNotNil() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://api.github.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)
	t.Log(result.Data.ID)

	tqpayload := newCreateTrackerQueryPayload(*result.Data.ID)

	test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)

	_, list := test.ListTrackerqueryOK(t, nil, nil, trackerQueryCtrl, nil, nil)
	for _, tq := range list.Data {
		if tq == nil {
			t.Error("Returned Tracker Query found nil")
		}
	}
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (rest *TestTrackerQueryREST) TestCreateTrackerQueryValidId() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerPayload{
		Data: &app.Tracker{
			Attributes: &app.TrackerAttributes{
				URL:  "http://api.github.com",
				Type: "jira",
			},
			Type: remoteworkitem.APIStringTypeTrackers,
		},
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)

	tqpayload := newCreateTrackerQueryPayload(*result.Data.ID)

	_, trackerquery := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	_, created := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, *trackerquery.Data.ID)
	if created != nil && created.Data.ID != trackerquery.Data.ID {
		t.Error("Failed because fetched Tracker query not same as requested. Found: ", trackerquery.Data.ID, " Expected, ", created.Data.ID)
	}
}

func newCreateTrackerQueryPayload(trackerID uuid.UUID) app.CreateTrackerqueryPayload {
	space := space.SystemSpace.String()
	tracker := trackerID.String()
	return app.CreateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			Attributes: &app.TrackerQueryAttributes{
				Query:    "is:open is:issue user:arquillian author:aslakknutsen",
				Schedule: "15 * * * * *",
			},
			Relationships: &app.TrackerQueryRelations{
				Space: &app.RelationGeneric{
					Data: &app.GenericData{
						ID: &space,
					},
				},
				Tracker: &app.RelationGeneric{
					Data: &app.GenericData{
						ID: &tracker,
					},
				},
			},
			Type: remoteworkitem.APIStringTypeTrackerQuery,
		},
	}
}
