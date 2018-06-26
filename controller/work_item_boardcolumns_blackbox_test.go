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
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
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
	db           *gormapplication.GormDB
	ctx          context.Context
	clean        func()
	testIdentity account.Identity
}

func TestRunWorkItemBoardcolumnREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestWorkItemBoardcolumnREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (l *TestWorkItemBoardcolumnREST) SetupTest() {
	l.db = gormapplication.NewGormDB(l.DB)
	l.clean = cleaner.DeleteCreatedEntities(l.DB)
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	l.ctx = goa.NewContext(context.Background(), nil, req, params)
	fixture := tf.NewTestFixture(l.T(), l.DB, tf.Identities(1))
	require.NotNil(l.T(), fixture)
	require.Len(l.T(), fixture.Identities, 1)
	l.testIdentity = *fixture.Identities[0]
}

func (l *TestWorkItemBoardcolumnREST) TearDownTest() {
	l.clean()
}

func (l *TestWorkItemBoardcolumnREST) SecuredController() (*goa.Service, app.WorkitemController) {
	svc := testsupport.ServiceAsUser("WorkItemBoardcolumn-Service", l.testIdentity)
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemBoardcolumnREST) UnSecuredController() (*goa.Service, app.WorkitemController) {
	svc := goa.New("WorkItemBoardcolumn-Service")
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemBoardcolumnREST) TestAddWItoBoardcolumn() {
	wiCnt := 2
	boardCnt := 2
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController()

	// Fetch WI and verify boardcolumns Relationship
	_, fetchedWI := test.ShowWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, nil, nil)
	require.NotNil(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns)
	assert.Empty(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns.Data)

	u := app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			ID:   &fixtures.WorkItems[0].ID,
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				"version": fixtures.WorkItems[0].Version,
			},
			Relationships: &app.WorkItemRelationships{},
		},
	}
	// add a column reference
	u.Data.Relationships.SystemBoardcolumns = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   ptr.String(fixtures.WorkItemBoards[0].Columns[0].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
			{
				ID:   ptr.String(fixtures.WorkItemBoards[1].Columns[1].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Len(l.T(), updatedWI.Data.Relationships.SystemBoardcolumns.Data, 2)
	mustHave := map[string]struct{}{
		*u.Data.Relationships.SystemBoardcolumns.Data[0].ID: {},
		*u.Data.Relationships.SystemBoardcolumns.Data[1].ID: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.SystemBoardcolumns.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)
}

func (l *TestWorkItemBoardcolumnREST) TestAddWItoDistinctBoardcolumn() {
	wiCnt := 2
	boardCnt := 2
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController()

	// Fetch WI and verify boardcolumns Relationship
	_, fetchedWI := test.ShowWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, nil, nil)
	require.NotNil(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns)
	assert.Empty(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns.Data)

	u := app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			ID:   &fixtures.WorkItems[0].ID,
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				"version": fixtures.WorkItems[0].Version,
			},
			Relationships: &app.WorkItemRelationships{},
		},
	}
	// add a column reference duplicate
	u.Data.Relationships.SystemBoardcolumns = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   ptr.String(fixtures.WorkItemBoards[0].Columns[0].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
			{
				ID:   ptr.String(fixtures.WorkItemBoards[0].Columns[0].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
			{
				ID:   ptr.String(fixtures.WorkItemBoards[1].Columns[1].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
			{
				ID:   ptr.String(fixtures.WorkItemBoards[1].Columns[1].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Len(l.T(), updatedWI.Data.Relationships.SystemBoardcolumns.Data, 2)
	mustHave := map[string]struct{}{
		*u.Data.Relationships.SystemBoardcolumns.Data[0].ID: {},
		*u.Data.Relationships.SystemBoardcolumns.Data[2].ID: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.SystemBoardcolumns.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)
}

func (l *TestWorkItemBoardcolumnREST) TestRemoveAllBoardcolumns() {
	wiCnt := 2
	boardCnt := 2
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(wiCnt), tf.WorkItemBoards(boardCnt))
	svc, ctrl := l.SecuredController()

	// Fetch WI and verify boardcolumns Relationship
	_, fetchedWI := test.ShowWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, nil, nil)
	require.NotNil(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns)
	assert.Empty(l.T(), fetchedWI.Data.Relationships.SystemBoardcolumns.Data)

	u := app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			ID:   &fixtures.WorkItems[0].ID,
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				"version": fixtures.WorkItems[0].Version,
			},
			Relationships: &app.WorkItemRelationships{},
		},
	}
	// add a column reference
	u.Data.Relationships.SystemBoardcolumns = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   ptr.String(fixtures.WorkItemBoards[0].Columns[0].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
			{
				ID:   ptr.String(fixtures.WorkItemBoards[1].Columns[1].ID.String()),
				Type: ptr.String("boardcolumns"),
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Len(l.T(), updatedWI.Data.Relationships.SystemBoardcolumns.Data, 2)
	mustHave := map[string]struct{}{
		*u.Data.Relationships.SystemBoardcolumns.Data[0].ID: {},
		*u.Data.Relationships.SystemBoardcolumns.Data[1].ID: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.SystemBoardcolumns.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)

	// now remove all columns
	u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
	u.Data.Relationships.SystemBoardcolumns = &app.RelationGenericList{
		Data: []*app.GenericData{},
	}
	_, updatedWI = test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Empty(l.T(), updatedWI.Data.Relationships.SystemBoardcolumns.Data)

}

/* FIXME(michaelkleinhenz): Add tests as soon as isValid is added to workitem.go
func (l *TestWorkItemBoardcolumnREST) TestFailInvalidLabel() {
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(1))
	svc, ctrl := l.SecuredController()
	apiLabelType := label.APIStringTypeLabels
	invalidLabelID := uuid.NewV4().String()
	u := app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			ID:   &fixtures.WorkItems[0].ID,
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				"version": fixtures.WorkItems[0].Version,
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
	test.UpdateWorkitemBadRequest(l.T(), svc.Context, svc, ctrl, fixtures.Spaces[0].ID, &u)
}
*/
