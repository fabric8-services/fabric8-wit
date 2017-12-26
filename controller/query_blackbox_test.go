package controller_test

import (
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-auth/auth"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/query"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestQueryREST struct {
	gormtestsupport.DBTestSuite
	db      *gormapplication.GormDB
	testDir string
	policy  *auth.KeycloakPolicy
}

func TestRunQueryREST(t *testing.T) {
	suite.Run(t, &TestQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestQueryREST) SecuredController() (*goa.Service, *QueryController) {
	svc := testsupport.ServiceAsUser("Query-Service", testsupport.TestIdentity)
	return svc, NewQueryController(svc, rest.db, rest.Configuration)
}

func (rest *TestQueryREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *QueryController) {
	svc := testsupport.ServiceAsUser("Query-Service", *idn)
	return svc, NewQueryController(svc, rest.db, rest.Configuration)
}

func (rest *TestQueryREST) UnSecuredController() (*goa.Service, *QueryController) {
	svc := goa.New("Query-Service")
	return svc, NewQueryController(svc, rest.db, rest.Configuration)
}

func (rest *TestQueryREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.testDir = filepath.Join("test-files", "query")
	rest.policy = &auth.KeycloakPolicy{
		Name:             "TestCollaborators-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
}

func getQueryCreatePayload(title string, qs *string) *app.CreateQueryPayload {
	defaultFields := `{"$AND": [{"space": "2a0efd64-ba69-42a6-b7da-750264744223"}]}`
	if qs == nil {
		qs = &defaultFields
	}
	qType := query.APIStringTypeQuery
	return &app.CreateQueryPayload{
		Data: &app.Query{
			Type: qType,
			Attributes: &app.QueryAttributes{
				Title:  title,
				Fields: *qs,
			},
		},
	}
}

func (rest *TestQueryREST) TestCreateQueryREST() {
	resetFn := rest.DisableGormCallbacks()
	defer resetFn()

	rest.T().Run("success", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload("query 1", nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			resp, created := test.CreateQueryCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
			// then
			require.NotNil(t, created)
			require.Equal(t, fxt.Identities[0].ID.String(), *created.Data.Relationships.Creator.Data.ID)
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.res.query.golden.json"), created)
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.headers.golden.json"), resp.Header())
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.req.payload.golden.json"), cq)
		})
		t.Run("same title with different spaceID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.Spaces(2),
				tf.Queries(1, tf.SetQueryTitles("query 1")))
			cq := getQueryCreatePayload(fxt.Queries[0].Title, nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			resp, created := test.CreateQueryCreated(t, svc.Context, svc, ctrl, fxt.Spaces[1].ID, cq)
			// then
			require.NotNil(t, created)
			require.Equal(t, fxt.Identities[0].ID.String(), *created.Data.Relationships.Creator.Data.ID)
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.res.query.golden.json"), created)
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.headers.golden.json"), resp.Header())
			compareWithGoldenUUIDAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.req.payload.golden.json"), cq)
		})
	})

	rest.T().Run("fail", func(t *testing.T) {
		t.Run("Unauthorized", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload("query 1", nil)
			svc, ctrl := rest.UnSecuredController()
			// when
			test.CreateQueryUnauthorized(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
		})
		t.Run("invalid query", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload("query 1", nil)
			cq.Data.Attributes.Fields = `{"invalid: json"}`
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.CreateQueryBadRequest(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
		})
		t.Run("conflict same title, spaceID, creator", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(1, tf.SetQueryTitles("q1")))
			cq := getQueryCreatePayload(fxt.Queries[0].Title, nil)
			cq.Data.Attributes.Fields = `{"invalid: json"}`
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.CreateQueryBadRequest(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
		})
	})
}
