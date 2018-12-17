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
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
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
}

func TestRunTrackerQueryREST(t *testing.T) {
	suite.Run(t, &TestTrackerQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestTrackerQueryREST) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.RwiScheduler = remoteworkitem.NewScheduler(s.DB)
	s.db = gormapplication.NewGormDB(s.DB)
}

func (s *TestTrackerQueryREST) SecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := testsupport.ServiceAsUser("TrackerQuery-Service", testsupport.TestIdentity)
	return svc, NewTrackerController(svc, s.db, s.RwiScheduler, s.Configuration), NewTrackerqueryController(svc, s.db, s.RwiScheduler, s.Configuration)
}

func (s *TestTrackerQueryREST) UnSecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := goa.New("TrackerQuery-Service")
	return svc, NewTrackerController(svc, s.db, s.RwiScheduler, s.Configuration), NewTrackerqueryController(svc, s.db, s.RwiScheduler, s.Configuration)
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
func (s *TestTrackerQueryREST) TestUnauthorizeTrackerQueryCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getTrackerQueryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTrackerQuery-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerqueryController(service, s.GormDB, s.RwiScheduler, s.Configuration)
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func (s *TestTrackerQueryREST) TestCreateTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tqresult := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload)
	assert.NotNil(s.T(), tqresult)
}

func (s *TestTrackerQueryREST) TestShowTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)

	_, tqresult := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload)
	_, tqr := test.ShowTrackerqueryOK(s.T(), svc.Context, svc, trackerQueryCtrl, *tqresult.Data.ID)
	assert.NotNil(s.T(), tqr)
	assert.Equal(s.T(), tqresult.Data.ID, tqr.Data.ID)
}

func (s *TestTrackerQueryREST) TestUpdateTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)

	_, tqresult := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload)

	_, tqr := test.ShowTrackerqueryOK(s.T(), svc.Context, svc, trackerQueryCtrl, *tqresult.Data.ID)
	assert.NotNil(s.T(), tqr)
	assert.Equal(s.T(), tqresult.Data.ID, tqr.Data.ID)

	payload2 := app.UpdateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			ID: tqr.Data.ID,
			Attributes: &app.TrackerQueryAttributes{
				Query:    "is:open",
				Schedule: "* * * * * *",
			},
			Relationships: &app.TrackerQueryRelations{
				Space: app.NewSpaceRelation(fxt.Spaces[0].ID, ""),
				Tracker: &app.RelationKindUUID{
					Data: &app.DataKindUUID{
						ID:   fxt.Trackers[0].ID,
						Type: remoteworkitem.APIStringTypeTrackers,
					},
				},
				WorkItemType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						ID:   fxt.WorkItemTypes[0].ID,
						Type: APIStringTypeWorkItemType,
					},
				},
			},
			Type: remoteworkitem.APIStringTypeTrackerQuery,
		},
	}

	_, updated := test.UpdateTrackerqueryOK(s.T(), svc.Context, svc, trackerQueryCtrl, tqr.Data.ID.String(), &payload2)
	require.NotNil(s.T(), tqr)
	require.Equal(s.T(), tqr.Data.ID, updated.Data.ID)
	require.Equal(s.T(), "is:open", updated.Data.Attributes.Query)
	require.Equal(s.T(), "* * * * * *", updated.Data.Attributes.Schedule)
}

// This test ensures that List does not return NIL items.
func (s *TestTrackerQueryREST) TestTrackerQueryListItemsNotNil() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0])

	tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tq1 := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload)
	assert.NotNil(s.T(), tq1)

	tqpayload2 := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
	_, tq2 := test.CreateTrackerqueryCreated(s.T(), svc.Context, svc, trackerQueryCtrl, &tqpayload2)
	assert.NotNil(s.T(), tq2)

	_, list := test.ListTrackerqueryOK(s.T(), svc.Context, svc, trackerQueryCtrl, nil, nil)
	assert.NotNil(s.T(), list.Data)
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (s *TestTrackerQueryREST) TestCreateTrackerQueryID() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1))

	s.T().Run("valid - success", func(t *testing.T) {
		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
		_, trackerquery := test.CreateTrackerqueryCreated(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
		require.NotNil(t, trackerquery)

		_, result := test.ShowTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, *trackerquery.Data.ID)
		require.NotNil(t, result)
		assert.Equal(t, trackerquery.Data.ID, result.Data.ID)
	})
	s.T().Run("invalid - fail", func(t *testing.T) {
		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
		invalidID := uuid.Nil
		tqpayload.Data.ID = &invalidID
		test.CreateTrackerqueryBadRequest(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	})
}

func (s *TestTrackerQueryREST) TestInvalidWITinTrackerQuery() {
	resource.Require(s.T(), resource.Database)
	s.T().Run("nil WIT in trackerquery payload", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1),
			tf.Trackers(1),
		)
		svc, _, trackerQueryCtrl := s.SecuredController()

		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, uuid.Nil)
		test.CreateTrackerqueryBadRequest(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	})

	s.T().Run("disallow creation if WIT belongs to different spacetemplate", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.SpaceTemplates(2),
			tf.Spaces(1),
			tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemTypes[idx].SpaceTemplateID = fxt.SpaceTemplates[1].ID
				return nil
			}),
			tf.Trackers(1),
		)
		svc, _, trackerQueryCtrl := s.SecuredController()

		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
		test.CreateTrackerqueryBadRequest(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
	})
}

func newCreateTrackerQueryPayload(spaceID uuid.UUID, trackerID uuid.UUID, witID uuid.UUID) app.CreateTrackerqueryPayload {
	trackerQueryID := uuid.NewV4()
	return app.CreateTrackerqueryPayload{
		Data: &app.TrackerQuery{
			ID: &trackerQueryID,
			Attributes: &app.TrackerQueryAttributes{
				Query:    "is:open is:issue user:arquillian author:aslakknutsen",
				Schedule: "15 * * * * *",
			},
			Relationships: &app.TrackerQueryRelations{
				Space: app.NewSpaceRelation(spaceID, ""),
				Tracker: &app.RelationKindUUID{
					Data: &app.DataKindUUID{
						ID:   trackerID,
						Type: remoteworkitem.APIStringTypeTrackers,
					},
				},
				WorkItemType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						ID:   witID,
						Type: APIStringTypeWorkItemType,
					},
				},
			},
			Type: remoteworkitem.APIStringTypeTrackerQuery,
		},
	}
}
