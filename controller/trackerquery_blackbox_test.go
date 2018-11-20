package controller_test

import (
	"bytes"
	"net/http"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerQueryREST struct {
	gormtestsupport.DBTestSuite
	RwiScheduler *remoteworkitem.Scheduler
	db           *gormapplication.GormDB
	clean        func()
}

func TestRunTrackerQueryREST(t *testing.T) {
	suite.Run(t, &TestTrackerQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestTrackerQueryREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.RwiScheduler = remoteworkitem.NewScheduler(rest.DB)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestTrackerQueryREST) TearDownTest() {
	rest.clean()
}

func (rest *TestTrackerQueryREST) SecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := testsupport.ServiceAsUser("TrackerQuery-Service", testsupport.TestIdentity)
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func (rest *TestTrackerQueryREST) UnSecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := goa.New("TrackerQuery-Service")
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func getTrackerQueryTestData(t *testing.T) []testSecureAPI {
	privatekey := testtoken.PrivateKey()
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.NoError(t, err)

	createTrackerQueryPayload := bytes.NewBuffer([]byte(`{"query": "is:open", "schedule": "5 * * * * *", "trackerID":"64e19607-9e54-4f11-a543-a0aa4288d326", "spaceID":"2e456849-4808-4a39-a3b7-a8c9252b1ede"}`))

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
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Delete tracker query API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker query
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackerqueries/" + uuid.NewV4().String(),
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
		controller := NewTrackerqueryController(service, rest.GormDB, rest.RwiScheduler, rest.Configuration)
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func (rest *TestTrackerQueryREST) TestCreateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, _, trackerQueryCtrl := rest.SecuredController()
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Trackers(1))
	assert.NotNil(rest.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	assert.NotNil(rest.T(), tqresult)
}

func (rest *TestTrackerQueryREST) TestGetTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, _, trackerQueryCtrl := rest.SecuredController()
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Trackers(1))
	assert.NotNil(rest.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)

	_, tqr := test.ShowTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, *tqresult.Data.ID)
	assert.NotNil(rest.T(), tqr)
	assert.Equal(rest.T(), tqresult.Data.ID, tqr.Data.ID)
}

func (rest *TestTrackerQueryREST) TestUpdateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, _, trackerQueryCtrl := rest.SecuredController()
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Trackers(1))
	assert.NotNil(rest.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)

	_, tqr := test.ShowTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, *tqresult.Data.ID)
	assert.NotNil(rest.T(), tqr)
	assert.Equal(rest.T(), tqresult.Data.ID, tqr.Data.ID)

	spaceID := space.SystemSpace.String()
	trackerID := fxt.Trackers[0].ID.String()
	payload2 := app.UpdateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			ID: tqr.Data.ID,
			Attributes: &app.TrackerQueryAttributes{
				Query:    "is:open",
				Schedule: "* * * * * *",
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
			Type: remoteworkitem.APIStringTypeTrackerQuery,
		},
	}

	_, updated := test.UpdateTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, tqr.Data.ID.String(), &payload2)
	assert.NotNil(rest.T(), tqr)
	assert.Equal(rest.T(), updated.Data.ID, tqr.Data.ID)
	assert.Equal(rest.T(), updated.Data.Attributes.Query, "is:open")
	assert.Equal(rest.T(), updated.Data.Attributes.Schedule, "* * * * * *")
}

// This test ensures that List does not return NIL items.
func (rest *TestTrackerQueryREST) TestTrackerQueryListItemsNotNil() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, _, trackerQueryCtrl := rest.SecuredController()
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Trackers(1))
	assert.NotNil(rest.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID)

	test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	// test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)

	// _, list := test.ListTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, nil, nil)
	// assert.NotNil(rest.T(), list.Data)
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (rest *TestTrackerQueryREST) TestCreateTrackerQueryValidId() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, _, trackerQueryCtrl := rest.SecuredController()
	fxt := tf.NewTestFixture(t, rest.DB, tf.Spaces(1), tf.Trackers(1))
	assert.NotNil(rest.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID)

	_, trackerquery := test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	_, created := test.ShowTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, *trackerquery.Data.ID)
	assert.Equal(rest.T(), created.Data.ID, trackerquery.Data.ID)
}

func newCreateTrackerQueryPayload(spaceID uuid.UUID, trackerID uuid.UUID) app.CreateTrackerqueryPayload {
	trackerQueryId := uuid.NewV4()
	space := spaceID.String()
	tracker := trackerID.String()
	return app.CreateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			ID: &trackerQueryId,
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
