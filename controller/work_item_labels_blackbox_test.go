package controller_test

import (
	"context"
	"fmt"
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
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestWorkItemLabelREST struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	ctx          context.Context
	clean        func()
	testIdentity account.Identity
}

func TestRunWorkItemLabelREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestWorkItemLabelREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (l *TestWorkItemLabelREST) SetupTest() {
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

func (l *TestWorkItemLabelREST) TearDownTest() {
	l.clean()
}

func (l *TestWorkItemLabelREST) SecuredController() (*goa.Service, app.WorkitemController) {
	svc := testsupport.ServiceAsUser("WorkItemLabel-Service", l.testIdentity)
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemLabelREST) UnSecuredController() (*goa.Service, app.WorkitemController) {
	svc := goa.New("WorkItemLabel-Service")
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemLabelREST) TestAttachLabelToWI() {
	wiCnt := 2
	lblCnt := 3
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(wiCnt), tf.Labels(lblCnt))
	svc, ctrl := l.SecuredController()
	relatedLink := fmt.Sprintf("/%s/labels", fixtures.WorkItems[0].ID)

	// Fetch WI and verify Labels Relationship
	_, fetchedWI := test.ShowWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, nil, nil)
	require.NotNil(l.T(), fetchedWI.Data.Relationships.Labels)
	require.NotNil(l.T(), fetchedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *fetchedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Empty(l.T(), fetchedWI.Data.Relationships.Labels.Data)

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
	// attach 2 labels
	apiLabelType := label.APIStringTypeLabels
	lbl0 := fixtures.Labels[0].ID.String()
	lbl1 := fixtures.Labels[1].ID.String()
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &lbl0,
				Type: &apiLabelType,
			},
			{
				ID:   &lbl1,
				Type: &apiLabelType,
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 2)
	mustHave := map[string]struct{}{
		lbl0: {},
		lbl1: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)

	// attach 1 label
	lbl2 := fixtures.Labels[2].ID.String()
	u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &lbl2,
				Type: &apiLabelType,
			},
		},
	}
	_, updatedWI = test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	require.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 1)
	mustHave = map[string]struct{}{
		lbl2: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)
}

func (l *TestWorkItemLabelREST) TestAttachDistinctLabels() {
	wiCnt := 2
	lblCnt := 3
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(wiCnt), tf.Labels(lblCnt))
	svc, ctrl := l.SecuredController()
	relatedLink := fmt.Sprintf("/%s/labels", fixtures.WorkItems[0].ID)
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
	// attach multiple labels with duplicates
	apiLabelType := label.APIStringTypeLabels
	lbl0 := fixtures.Labels[0].ID.String()
	lbl1 := fixtures.Labels[1].ID.String()
	lbl2 := fixtures.Labels[2].ID.String()
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &lbl0,
				Type: &apiLabelType,
			}, {
				ID:   &lbl0,
				Type: &apiLabelType,
			}, {
				ID:   &lbl0,
				Type: &apiLabelType,
			}, {
				ID:   &lbl1,
				Type: &apiLabelType,
			}, {
				ID:   &lbl1,
				Type: &apiLabelType,
			}, {
				ID:   &lbl2,
				Type: &apiLabelType,
			}, {
				ID:   &lbl2,
				Type: &apiLabelType,
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	require.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 3)
	mustHave := map[string]struct{}{
		lbl0: {},
		lbl1: {},
		lbl2: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)
}

func (l *TestWorkItemLabelREST) TestAttachLabelUnauthorized() {
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.WorkItems(1))
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
	// verify Unauthorized access
	svc, ctrl := l.UnSecuredController()
	test.UpdateWorkitemUnauthorized(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
}

func (l *TestWorkItemLabelREST) TestDetachAllLabels() {
	wiCnt := 2
	lblCnt := 3
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(wiCnt), tf.Labels(lblCnt))
	svc, ctrl := l.SecuredController()
	relatedLink := fmt.Sprintf("/%s/labels", fixtures.WorkItems[0].ID)
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
	// First attach some labels
	lbl := fixtures.Labels[2].ID.String()
	apiLabelType := label.APIStringTypeLabels
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &lbl,
				Type: &apiLabelType,
			},
		},
	}
	_, updatedWI := test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	require.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 1)
	mustHave := map[string]struct{}{
		lbl: {},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)

	// now detach all labels
	u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{},
	}
	_, updatedWI = test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Empty(l.T(), updatedWI.Data.Relationships.Labels.Data)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
}

func (l *TestWorkItemLabelREST) TestAttachLabelBadRequest() {
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.WorkItems(1))
	svc, ctrl := l.SecuredController()
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
	// Bad Request using nil
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: nil,
	}
	test.UpdateWorkitemBadRequest(l.T(), svc.Context, svc, ctrl, fixtures.Spaces[0].ID, &u)
}

func (l *TestWorkItemLabelREST) TestFailInvalidLabel() {
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
