package controller_test

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-common/auth"
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
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerQueryREST struct {
	gormtestsupport.DBTestSuite
	RwiScheduler  *remoteworkitem.Scheduler
	db            *gormapplication.GormDB
	authService   auth.AuthService
	workitemCtrl  app.WorkitemController
	workitemsCtrl app.WorkitemsController
}

func TestRunTrackerQueryREST(t *testing.T) {
	suite.Run(t, &TestTrackerQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestTrackerQueryREST) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.RwiScheduler = remoteworkitem.NewScheduler(s.DB)
	s.db = gormapplication.NewGormDB(s.DB)
	s.workitemCtrl = NewWorkitemController(s.svc, s.GormDB, s.Configuration)
	s.workitemsCtrl = NewWorkitemsController(s.svc, s.GormDB, s.Configuration)
}

type testAuthService struct{}

func (s *testAuthService) RequireScope(ctx context.Context, resourceID, requiredScope string) error {
	return nil
}

func (s *TestTrackerQueryREST) UnSecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := goa.New("TrackerQuery-Service")
	return svc, NewTrackerController(svc, s.db, s.RwiScheduler, s.Configuration), NewTrackerqueryController(svc, s.db, s.RwiScheduler, s.Configuration, &testAuthService{})
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
		controller := NewTrackerqueryController(service, s.GormDB, s.RwiScheduler, s.Configuration, &testAuthService{})
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func (s *TestTrackerQueryREST) TestCreateTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1), tf.TrackerQueries(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0], fxt.TrackerQueries[0])

	s.T().Run("nil WIT in trackerquery payload", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB,
			tf.Spaces(1),
			tf.Trackers(1),
		)
		svc, _, trackerQueryCtrl := s.SecuredController()

		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, uuid.Nil)
		_, err := test.CreateTrackerqueryBadRequest(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
		require.NotNil(t, err)
		require.IsType(t, strconv.Itoa(http.StatusBadRequest), *err.Errors[0].Status)
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
		_, err := test.CreateTrackerqueryBadRequest(t, svc.Context, svc, trackerQueryCtrl, &tqpayload)
		require.NotNil(t, err)
		require.IsType(t, strconv.Itoa(http.StatusBadRequest), *err.Errors[0].Status)
	})
}

func (s *TestTrackerQueryREST) TestShowTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1), tf.TrackerQueries(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0], fxt.TrackerQueries[0])

	_, tqr := test.ShowTrackerqueryOK(s.T(), svc.Context, svc, trackerQueryCtrl, fxt.TrackerQueries[0].ID)
	assert.NotNil(s.T(), tqr)
	assert.Equal(s.T(), fxt.TrackerQueries[0].ID, *tqr.Data.ID)
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (s *TestTrackerQueryREST) TestCreateTrackerQueryID() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1), tf.TrackerQueries(1))

	s.T().Run("valid - success", func(t *testing.T) {
		_, result := test.ShowTrackerqueryOK(t, svc.Context, svc, trackerQueryCtrl, fxt.TrackerQueries[0].ID)
		require.NotNil(t, result)
		assert.Equal(t, fxt.TrackerQueries[0].ID, *result.Data.ID)
	})
	s.T().Run("invalid - fail", func(t *testing.T) {
		tqpayload := newCreateTrackerQueryPayload(fxt.Spaces[0].ID, fxt.Trackers[0].ID, fxt.WorkItemTypes[0].ID)
		invalidID := uuid.Nil
		tqpayload.Data.ID = &invalidID
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

func (s *TestTrackerQueryREST) TestDeleteTrackerQuery() {
	resource.Require(s.T(), resource.Database)

	svc, _, trackerQueryCtrl := s.SecuredController()
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1), tf.Trackers(1), tf.WorkItemTypes(1), tf.TrackerQueries(1))
	assert.NotNil(s.T(), fxt.Spaces[0], fxt.Trackers[0], fxt.TrackerQueries[0])

	s.T().Run("delete trackerquery - success", func(t *testing.T) {
		test.DeleteTrackerqueryNoContent(t, svc.Context, svc, trackerQueryCtrl, fxt.TrackerQueries[0].ID)
	})

	s.T().Run("delete trackerquery - not found", func(t *testing.T) {
		test.DeleteTrackerqueryNotFound(t, svc.Context, svc, trackerQueryCtrl, uuid.NewV4())
	})

	s.T().Run("delete trackerquery - unauthorized", func(t *testing.T) {
		svc2, _, trackerQueryUnsecuredCtrl := s.UnSecuredController()
		_, err := test.DeleteTrackerqueryUnauthorized(t, svc2.Context, svc2, trackerQueryUnsecuredCtrl, fxt.TrackerQueries[0].ID)
		require.NotNil(t, err)
		require.IsType(t, strconv.Itoa(http.StatusUnauthorized), *err.Errors[0].Status)
	})

	t.Run("delete remoteworkitems - true", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(1),
			tf.WorkItemTypes(1),
			tf.Trackers(1),
			tf.TrackerQueries(2),
			tf.WorkItems(3, func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0, 1:
					fxt.WorkItems[idx].Fields[workitem.SystemRemoteTrackerID] = fxt.TrackerQueries[0].ID
				default:
					fxt.WorkItems[idx].Fields[workitem.SystemRemoteTrackerID] = fxt.TrackerQueries[1].ID
				}
				return nil
			}),
		)
		assert.NotNil(s.T(), fxt.Spaces, fxt.Trackers, fxt.WorkItemTypes, fxt.TrackerQueries, fxt.WorkItems)
		s.svc = testsupport.ServiceAsUser("TestDeleteTrackerQuery-Service", *fxt.Identities[0])

		_, result := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Len(t, result.Data, 3)

		err := test.DeleteTrackerqueryOK(t, s.svc.Context, s.svc, s.trackerqueryCtrl, fxt.TrackerQueries[0].ID, true)
		require.NotNil(t, err)

		_, result = test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Len(t, result.Data, 1)

		_, jerr := test.ShowWorkitemNotFound(t, s.svc.Context, s.svc, s.workitemCtrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotNil(t, jerr)

		_, jerr = test.ShowWorkitemNotFound(t, s.svc.Context, s.svc, s.workitemCtrl, fxt.WorkItems[1].ID, nil, nil)
		require.NotNil(t, jerr)
	})

	t.Run("delete remoteworkitems - false", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB,
			tf.Spaces(1),
			tf.WorkItemTypes(1),
			tf.Trackers(1),
			tf.TrackerQueries(2),
			tf.WorkItems(3, func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0, 1:
					fxt.WorkItems[idx].Fields[workitem.SystemRemoteTrackerID] = fxt.TrackerQueries[0].ID
				default:
					fxt.WorkItems[idx].Fields[workitem.SystemRemoteTrackerID] = fxt.TrackerQueries[1].ID
				}
				return nil
			}),
		)
		assert.NotNil(s.T(), fxt.Spaces, fxt.Trackers, fxt.WorkItemTypes, fxt.TrackerQueries, fxt.WorkItems)
		s.svc = testsupport.ServiceAsUser("TestDeleteTrackerQuery-Service", *fxt.Identities[0])

		_, result := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Len(t, result.Data, 3)

		err := test.DeleteTrackerqueryOK(t, s.svc.Context, s.svc, s.trackerqueryCtrl, fxt.TrackerQueries[0].ID, false)
		require.NotNil(t, err)

		_, result = test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, fxt.Spaces[0].ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Len(t, result.Data, 3)

		_, jerr := test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workitemCtrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotNil(t, jerr)

		_, jerr = test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workitemCtrl, fxt.WorkItems[1].ID, nil, nil)
		require.NotNil(t, jerr)
	})

}
