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
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
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
	pub, _ := wittoken.RSAPublicKey()
	svc := testsupport.ServiceAsUser("WorkItemLabel-Service", wittoken.NewManager(pub), l.testIdentity)
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemLabelREST) UnSecuredController() (*goa.Service, app.WorkitemController) {
	svc := goa.New("WorkItemLabel-Service")
	return svc, NewWorkitemController(svc, l.db, l.Configuration)
}

func (l *TestWorkItemLabelREST) TestAttachDetachLabelToWI() {
	wiCnt := 2
	lblCnt := 3
	fixtures := tf.NewTestFixture(l.T(), l.DB, tf.Spaces(1), tf.Iterations(1), tf.Areas(1), tf.WorkItems(wiCnt), tf.Labels(lblCnt))
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
	relatedLink := fmt.Sprintf("/%s/labels", fixtures.WorkItems[0].ID)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 2)
	mustHave := map[string]struct{}{
		lbl0: struct{}{},
		lbl1: struct{}{},
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
	assert.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 1)
	mustHave = map[string]struct{}{
		lbl2: struct{}{},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)

	// detach all labels
	u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: nil,
	}
	_, updatedWI = test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	assert.Empty(l.T(), updatedWI.Data.Relationships.Labels.Data)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)

	// verify distinct labels are attached
	lbl1 = fixtures.Labels[1].ID.String()
	lbl2 = fixtures.Labels[2].ID.String()
	u.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
	u.Data.Relationships.Labels = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
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
	_, updatedWI = test.UpdateWorkitemOK(l.T(), svc.Context, svc, ctrl, fixtures.WorkItems[0].ID, &u)
	assert.NotNil(l.T(), updatedWI)
	require.NotNil(l.T(), updatedWI.Data.Relationships.Labels.Links)
	assert.Contains(l.T(), *updatedWI.Data.Relationships.Labels.Links.Related, relatedLink)
	assert.Len(l.T(), updatedWI.Data.Relationships.Labels.Data, 2)
	mustHave = map[string]struct{}{
		lbl1: struct{}{},
		lbl2: struct{}{},
	}
	for _, lblData := range updatedWI.Data.Relationships.Labels.Data {
		delete(mustHave, *lblData.ID)
	}
	require.Empty(l.T(), mustHave)

	// verify Unauthorized access
	svc2, ctrl2 := l.UnSecuredController()
	test.UpdateWorkitemUnauthorized(l.T(), svc2.Context, svc2, ctrl2, fixtures.WorkItems[0].ID, &u)
}
