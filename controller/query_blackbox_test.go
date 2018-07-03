package controller_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-auth/auth"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/query"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestQueryREST struct {
	gormtestsupport.DBTestSuite
	testDir string
	policy  *auth.KeycloakPolicy
}

func TestRunQueryREST(t *testing.T) {
	suite.Run(t, &TestQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (rest *TestQueryREST) SecuredController() (*goa.Service, *QueryController) {
	svc := testsupport.ServiceAsUser("Query-Service", testsupport.TestIdentity)
	return svc, NewQueryController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestQueryREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *QueryController) {
	svc := testsupport.ServiceAsUser("Query-Service", *idn)
	return svc, NewQueryController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestQueryREST) UnSecuredController() (*goa.Service, *QueryController) {
	svc := goa.New("Query-Service")
	return svc, NewQueryController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestQueryREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
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

func (rest *TestQueryREST) TestCreate() {

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
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.res.query.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.headers.golden.json"), resp.Header())
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.req.payload.golden.json"), cq)
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
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.res.query.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.headers.golden.json"), resp.Header())
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.req.payload.golden.json"), cq)
		})
		t.Run("same object after delete", func(t *testing.T) {
			queryTitle := "query 1"
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload(queryTitle, nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			resp, created := test.CreateQueryCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
			require.NotNil(t, created)

			// detele the query
			test.DeleteQueryNoContent(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, *created.Data.ID)

			// try to create exact same query again
			resp, created = test.CreateQueryCreated(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
			// then
			require.NotNil(t, created)
			require.Equal(t, fxt.Identities[0].ID.String(), *created.Data.Relationships.Creator.Data.ID)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.res.query.golden.json"), created)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.headers.golden.json"), resp.Header())
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "create", "ok_create.req.payload.golden.json"), cq)
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
		t.Run("invalid query - empty title", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload(" ", nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.CreateQueryBadRequest(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
		})
		t.Run("conflict same title, spaceID, creator", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(1, tf.SetQueryTitles("q1")))
			cq := getQueryCreatePayload(fxt.Queries[0].Title, nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.CreateQueryConflict(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, cq)
		})
		t.Run("unknown space ID", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			cq := getQueryCreatePayload("new query", nil)
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.CreateQueryNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), cq)
		})
	})
}

func (rest *TestQueryREST) TestList() {

	rest.T().Run("success", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(3, tf.SetQueryTitles("q1", "q2", "q3")))
			fxt2 := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(3, tf.SetQueryTitles("q4", "q5", "q6")))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			_, qList := test.ListQueryOK(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, nil, nil)
			// then
			require.NotNil(t, qList)
			mustHave := map[string]struct{}{
				"q1": {},
				"q2": {},
				"q3": {},
			}
			for _, q := range qList.Data {
				delete(mustHave, q.Attributes.Title)
			}
			assert.Empty(t, mustHave)
			assert.Equal(t, 3, qList.Meta.TotalCount)
			// list by different user
			// when
			svc, ctrl = rest.SecuredControllerWithIdentity(fxt2.Identities[0])
			_, qList = test.ListQueryOK(t, svc.Context, svc, ctrl, fxt2.Spaces[0].ID, nil, nil)
			// then
			require.NotNil(t, qList)
			mustHave = map[string]struct{}{
				"q4": {},
				"q5": {},
				"q6": {},
			}
			for _, q := range qList.Data {
				delete(mustHave, q.Attributes.Title)
			}
			assert.Empty(t, mustHave)
			assert.Equal(t, 3, qList.Meta.TotalCount)
		})
	})

	rest.T().Run("fail", func(t *testing.T) {
		t.Run("unauthorized", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(2))
			svc, ctrl := rest.UnSecuredController()
			// when
			test.ListQueryUnauthorized(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, nil, nil)
		})
		t.Run("unknown space ID", func(t *testing.T) {
			// given
			fxt := tf.NewTestFixture(t, rest.DB, tf.Identities(1))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.ListQueryNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
		})
	})
}

func (rest *TestQueryREST) TestShow() {

	rest.T().Run("success", func(t *testing.T) {
		t.Run("ok with identity", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(1))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			q := fxt.Queries[0]
			// when
			resp, queryObj := test.ShowQueryOK(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, q.ID, nil, nil)
			// then
			require.NotNil(t, queryObj)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok_show.res.query.golden.json"), queryObj)
			compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "show", "ok_show.headers.golden.json"), resp.Header())
		})
	})

	rest.T().Run("fail", func(t *testing.T) {
		t.Run("unauthorized", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(1))
			svc, ctrl := rest.UnSecuredController()
			q := fxt.Queries[0]
			// when
			test.ShowQueryUnauthorized(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, q.ID, nil, nil)
		})
		t.Run("random UUID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			randomUUID := uuid.NewV4()
			test.ShowQueryNotFound(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, randomUUID, nil, nil)
		})
		t.Run("different space ID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
			fxt2 := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.ShowQueryNotFound(t, svc.Context, svc, ctrl, fxt2.Spaces[0].ID, fxt.Queries[0].ID, nil, nil)
		})
		t.Run("unknown space ID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.ShowQueryNotFound(t, svc.Context, svc, ctrl, uuid.NewV4(), fxt.Queries[0].ID, nil, nil)
		})
		t.Run("forbidden", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Identities(2),
				tf.Queries(1))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[1])
			// when
			test.ShowQueryForbidden(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, fxt.Queries[0].ID, nil, nil)
		})
	})
}

func (rest *TestQueryREST) TestUpdate() {

	rest.T().Run("update query", func(t *testing.T) {
		// 1 for each test, without conflict of changes during each test execution, isolated or not :)
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt.Identities[0])
		newTitle := "Query New 1001"
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`
		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   newTitle,
					Version: &testFxt.Queries[0].Version,
					Fields:  fields,
				},
				ID:   &testFxt.Queries[0].ID,
				Type: query.APIStringTypeQuery,
			},
		}
		resp, updated := test.UpdateQueryOK(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		assert.Equal(t, newTitle, updated.Data.Attributes.Title)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "update_query.res.payload.golden.json"), updated)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "update_query.res.headers.golden.json"), resp.Header())

		_, queries2 := test.ShowQueryOK(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil)
		assertQueryLinking(t, queries2.Data)
		require.NotEmpty(t, queries2.Data, "queries found")
		assert.Equal(t, newTitle, queries2.Data.Attributes.Title)
	})

	rest.T().Run("update query with version conflict", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt.Identities[0])
		newVersion := testFxt.Queries[0].Version + 2
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`
		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   testFxt.Queries[0].Title,
					Version: &newVersion,
					Fields:  fields,
				},
				ID:   &testFxt.Queries[0].ID,
				Type: query.APIStringTypeQuery,
			},
		}
		resp, jerrs := test.UpdateQueryConflict(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "version conflict")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "update_conflict.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "update_conflict.res.headers.golden.json"), resp.Header())
	})

	rest.T().Run("update query with bad parameter", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt.Identities[0])
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`
		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:  testFxt.Queries[0].Title,
					Fields: fields,
				},
				Type: query.APIStringTypeQuery,
			},
		}

		resp, jerrs := test.UpdateQueryBadRequest(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Bad value for parameter 'data.attributes.version'")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_version.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_version.res.headers.golden.json"), resp.Header())
	})

	rest.T().Run("update query with bad parameter - title", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt.Identities[0])
		newTitle := " 	   " // tab & spaces
		newVersion := testFxt.Queries[0].Version
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`
		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   newTitle,
					Version: &newVersion,
					Fields:  fields,
				},
				ID:   &testFxt.Queries[0].ID,
				Type: query.APIStringTypeQuery,
			},
		}

		resp, jerrs := test.UpdateQueryBadRequest(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Bad value for parameter 'query title cannot be empty string'")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_name.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "badparam_name.res.headers.golden.json"), resp.Header())
	})

	rest.T().Run("update query with unauthorized", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc := goa.New("Query-Service")
		ctrl := NewQueryController(svc, rest.GormDB, rest.Configuration)
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`

		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   testFxt.Queries[0].Title,
					Version: &testFxt.Queries[0].Version,
					Fields:  fields,
				},
				Type: query.APIStringTypeQuery,
			},
		}

		resp, jerrs := test.UpdateQueryUnauthorized(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "Missing token manager")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "unauthorized.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "unauthorized.res.headers.golden.json"), resp.Header())
	})

	rest.T().Run("different user updating", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		testFxt2 := tf.NewTestFixture(t, rest.DB,
			tf.Identities(1))

		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt2.Identities[0])
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`

		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   testFxt.Queries[0].Title,
					Version: &testFxt.Queries[0].Version,
					Fields:  fields,
				},
				Type: query.APIStringTypeQuery,
			},
		}

		resp, jerrs := test.UpdateQueryForbidden(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, testFxt.Queries[0].ID, nil, nil, &payload)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.Contains(t, jerrs.Errors[0].Detail, "user is not the query creator")
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "forbiden.res.payload.golden.json"), jerrs)
		compareWithGoldenAgnostic(t, filepath.Join(rest.testDir, "update", "forbiden.res.headers.golden.json"), resp.Header())
	})

	rest.T().Run("update query not found", func(t *testing.T) {
		testFxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Queries(1))
		svc, ctrl := rest.SecuredControllerWithIdentity(testFxt.Identities[0])
		newTitle := "Query New 1002"
		newVersion := testFxt.Queries[0].Version + 1
		fields := `{"$AND": [{"space": "1b0efd64-ba69-42a6-b7da-762264744223"}]}`
		id := uuid.NewV4()
		payload := app.UpdateQueryPayload{
			Data: &app.Query{
				Attributes: &app.QueryAttributes{
					Title:   newTitle,
					Version: &newVersion,
					Fields:  fields,
				},
				ID:   &id,
				Type: query.APIStringTypeQuery,
			},
		}
		test.UpdateQueryNotFound(t, svc.Context, svc, ctrl, testFxt.Spaces[0].ID, id, nil, nil, &payload)
	})
}

func (rest *TestQueryREST) TestDelete() {

	rest.T().Run("success", func(t *testing.T) {
		t.Run("ok with identity", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB,
				tf.CreateWorkItemEnvironment(),
				tf.Queries(1))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			q := fxt.Queries[0]
			// when
			test.DeleteQueryNoContent(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, q.ID)
		})
	})

	rest.T().Run("fail", func(t *testing.T) {
		t.Run("random UUID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			randomUUID := uuid.NewV4()
			test.DeleteQueryNotFound(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, randomUUID)
		})
		t.Run("different space ID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
			fxt2 := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.DeleteQueryNotFound(t, svc.Context, svc, ctrl, fxt2.Spaces[0].ID, fxt.Queries[0].ID)
		})
		t.Run("different user", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1), tf.Identities(2))
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[1])
			// when
			test.DeleteQueryForbidden(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, fxt.Queries[0].ID)
		})
		t.Run("unauthorized", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment(), tf.Queries(1))
			svc, ctrl := rest.UnSecuredController()
			// when
			test.DeleteQueryUnauthorized(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, fxt.Queries[0].ID)
		})
		t.Run("nil UUID", func(t *testing.T) {
			fxt := tf.NewTestFixture(t, rest.DB, tf.CreateWorkItemEnvironment())
			svc, ctrl := rest.SecuredControllerWithIdentity(fxt.Identities[0])
			// when
			test.DeleteQueryNotFound(t, svc.Context, svc, ctrl, fxt.Spaces[0].ID, uuid.Nil)
		})
	})
}

func assertQueryLinking(t *testing.T, target *app.Query) {
	assert.NotNil(t, target.ID)
	assert.Equal(t, query.APIStringTypeQuery, target.Type)
	assert.NotNil(t, target.Links.Self)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Space)
	require.NotNil(t, target.Relationships.Space.Links)
	require.NotNil(t, target.Relationships.Space.Links.Self)
	assert.True(t, strings.Contains(*target.Relationships.Space.Links.Self, "/api/spaces/"))
}
