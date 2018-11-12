package controller_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/ptr"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestWorkItemBoardcolumnREST struct {
	gormtestsupport.DBTestSuite
	ctx context.Context
}

func TestRunWorkItemBoardcolumnREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestWorkItemBoardcolumnREST{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (l *TestWorkItemBoardcolumnREST) SetupTest() {
	l.DBTestSuite.SetupTest()
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	l.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (l *TestWorkItemBoardcolumnREST) SecuredController(identity account.Identity) (*goa.Service, app.WorkitemController) {
	svc := testsupport.ServiceAsUser("WorkItemBoardcolumn-Service", identity)
	return svc, NewWorkitemController(svc, l.GormDB, l.Configuration)
}

func (l *TestWorkItemBoardcolumnREST) UnSecuredController() (*goa.Service, app.WorkitemController) {
	svc := goa.New("WorkItemBoardcolumn-Service")
	return svc, NewWorkitemController(svc, l.GormDB, l.Configuration)
}

func (l *TestWorkItemBoardcolumnREST) TestAddWItoBoardcolumn() {
	wiCnt := 2
	boardCnt := 2
	fxt := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController(*fxt.Identities[0])

	l.T().Run("Fetch WI and verify boardcolumns Relationship", func(t *testing.T) {
		_, fetchedWI := test.ShowWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotNil(l.T(), fetchedWI.Data.Relationships.Boardcolumns)
		assert.Empty(l.T(), fetchedWI.Data.Relationships.Boardcolumns.Data)
	})

	l.T().Run("add a column reference", func(t *testing.T) {
		// given
		u := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				ID:   &fxt.WorkItems[0].ID,
				Type: APIStringTypeWorkItem,
				Attributes: map[string]interface{}{
					"version": fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Boardcolumns: &app.RelationGenericList{
						Data: []*app.GenericData{
							{
								ID:   ptr.String(fxt.WorkItemBoards[0].Columns[0].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
							{
								ID:   ptr.String(fxt.WorkItemBoards[1].Columns[1].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
						},
					},
				},
			},
		}
		// when
		_, updatedWI := test.UpdateWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, &u)
		// then
		assert.NotNil(t, updatedWI)
		assert.Len(t, updatedWI.Data.Relationships.Boardcolumns.Data, 2)
		mustHave := map[string]struct{}{
			*u.Data.Relationships.Boardcolumns.Data[0].ID: {},
			*u.Data.Relationships.Boardcolumns.Data[1].ID: {},
		}
		for _, lblData := range updatedWI.Data.Relationships.Boardcolumns.Data {
			delete(mustHave, *lblData.ID)
		}
		require.Empty(t, mustHave)
	})
}

func (l *TestWorkItemBoardcolumnREST) TestAddWItoDistinctBoardcolumn() {
	wiCnt := 2
	boardCnt := 2
	fxt := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController(*fxt.Identities[0])

	l.T().Run("Fetch WI and verify boardcolumns Relationship", func(t *testing.T) {
		_, fetchedWI := test.ShowWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotNil(t, fetchedWI.Data.Relationships.Boardcolumns)
		assert.Empty(t, fetchedWI.Data.Relationships.Boardcolumns.Data)
	})

	l.T().Run("add a column reference duplicate", func(t *testing.T) {
		// given
		u := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				ID:   &fxt.WorkItems[0].ID,
				Type: APIStringTypeWorkItem,
				Attributes: map[string]interface{}{
					"version": fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Boardcolumns: &app.RelationGenericList{
						Data: []*app.GenericData{
							{
								ID:   ptr.String(fxt.WorkItemBoards[0].Columns[0].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
							{
								ID:   ptr.String(fxt.WorkItemBoards[0].Columns[0].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
							{
								ID:   ptr.String(fxt.WorkItemBoards[1].Columns[1].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
							{
								ID:   ptr.String(fxt.WorkItemBoards[1].Columns[1].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
						},
					},
				},
			},
		}
		// when
		_, updatedWI := test.UpdateWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, &u)
		// then
		assert.NotNil(t, updatedWI)
		assert.Len(t, updatedWI.Data.Relationships.Boardcolumns.Data, 2)
		mustHave := map[string]struct{}{
			*u.Data.Relationships.Boardcolumns.Data[0].ID: {},
			*u.Data.Relationships.Boardcolumns.Data[2].ID: {},
		}
		for _, lblData := range updatedWI.Data.Relationships.Boardcolumns.Data {
			delete(mustHave, *lblData.ID)
		}
		require.Empty(t, mustHave)
	})
}

func (l *TestWorkItemBoardcolumnREST) TestRemoveAllBoardcolumns() {
	wiCnt := 2
	boardCnt := 2
	fxt := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController(*fxt.Identities[0])

	l.T().Run("Fetch WI and verify boardcolumns Relationship", func(t *testing.T) {
		_, fetchedWI := test.ShowWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotNil(t, fetchedWI.Data.Relationships.Boardcolumns)
		assert.Empty(t, fetchedWI.Data.Relationships.Boardcolumns.Data)
	})

	l.T().Run("add a column reference", func(t *testing.T) {
		// given
		u := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				ID:   &fxt.WorkItems[0].ID,
				Type: APIStringTypeWorkItem,
				Attributes: map[string]interface{}{
					"version": fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Boardcolumns: &app.RelationGenericList{
						Data: []*app.GenericData{
							{
								ID:   ptr.String(fxt.WorkItemBoards[0].Columns[0].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
							{
								ID:   ptr.String(fxt.WorkItemBoards[1].Columns[1].ID.String()),
								Type: ptr.String("boardcolumns"),
							},
						},
					},
				},
			},
		}
		// when
		_, updatedWI := test.UpdateWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, &u)
		// then
		assert.NotNil(t, updatedWI)
		assert.Len(t, updatedWI.Data.Relationships.Boardcolumns.Data, 2)
		mustHave := map[string]struct{}{
			*u.Data.Relationships.Boardcolumns.Data[0].ID: {},
			*u.Data.Relationships.Boardcolumns.Data[1].ID: {},
		}
		for _, lblData := range updatedWI.Data.Relationships.Boardcolumns.Data {
			delete(mustHave, *lblData.ID)
		}
		require.Empty(t, mustHave)

		t.Run("now remove all columns", func(t *testing.T) {
			// given
			u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
			u.Data.Relationships.Boardcolumns = &app.RelationGenericList{
				Data: []*app.GenericData{},
			}
			// when
			_, updatedWI = test.UpdateWorkitemOK(t, svc.Context, svc, ctrl, fxt.WorkItems[0].ID, &u)
			// then
			assert.NotNil(t, updatedWI)
			assert.Empty(t, updatedWI.Data.Relationships.Boardcolumns.Data)
		})
	})
}

/* FIXME(michaelkleinhenz): Add tests as soon as isValid is added to workitem.go
func (l *TestWorkItemBoardcolumnREST) TestFailInvalidLabel() {
	fxt := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(1))
	svc, ctrl := l.SecuredController()
	apiLabelType := label.APIStringTypeLabels
	invalidLabelID := uuid.NewV4().String()
	u := app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			ID:   &fxt.WorkItems[0].ID,
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				"version": fxt.WorkItems[0].Version,
			},
			Relationships: &app.WorkItemRelationships{
				Labels: &app.RelationGenericList{
					Data: []*app.GenericData{
						{
							ID:   &invalidLabelID,
							Type: &apiLabelType,
						},
					},
				},
			},
		},
	}
	test.UpdateWorkitemBadRequest(l.T(), svc.Context, svc, ctrl, fxt.Spaces[0].ID, &u)
}
*/
