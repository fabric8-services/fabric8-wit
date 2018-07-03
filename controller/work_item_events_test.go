package controller_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/label"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestEvent struct {
	gormtestsupport.DBTestSuite
	testDir string
}

func TestRunEvent(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestEvent{DBTestSuite: gormtestsupport.NewDBTestSuite()})
}

func (s *TestEvent) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.testDir = filepath.Join("test-files", "event")
}

func (s *TestEvent) TestListEvent() {

	s.T().Run("event list ok - state", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemState:   "resolved",
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-state.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-state.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - title", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemTitle:   "New Title",
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-title.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-title.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - Field Type Float", func(t *testing.T) {
		fieldName := "myFloatType"
		fxt := tf.NewTestFixture(t, s.DB,
			tf.CreateWorkItemEnvironment(),
			tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
					fieldName: {
						Type: &workitem.SimpleType{Kind: workitem.KindFloat},
					},
				}
				return nil
			}),
			tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItems[idx].Type = fxt.WorkItemTypes[idx].ID
				fxt.WorkItems[idx].Fields[fieldName] = 1.99
				fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "My workitem"
				return nil
			}),
		)
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemTitle:   fxt.WorkItems[0].Fields[workitem.SystemTitle],
					fieldName:              2.99,
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindFloat.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindFloat.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - Field Type Int", func(t *testing.T) {
		fieldName := "myIntType"
		fxt := tf.NewTestFixture(t, s.DB,
			tf.CreateWorkItemEnvironment(),
			tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItemTypes[idx].Fields = map[string]workitem.FieldDefinition{
					fieldName: {
						Type: &workitem.SimpleType{Kind: workitem.KindInteger},
					},
				}
				return nil
			}),
			tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
				fxt.WorkItems[idx].Type = fxt.WorkItemTypes[idx].ID
				fxt.WorkItems[idx].Fields[fieldName] = 200
				fxt.WorkItems[idx].Fields[workitem.SystemTitle] = "My workitem"
				return nil
			}),
		)
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemTitle:   fxt.WorkItems[0].Fields[workitem.SystemTitle],
					fieldName:              4235,
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindInt.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindInt.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - description", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemDescription: "New Description",
					workitem.SystemVersion:     fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-description.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-description.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - assigned", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		assignee := []string{fxt.Identities[0].ID.String()}
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemAssignees: assignee,
					workitem.SystemVersion:   fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-assignees.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-assignees.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - label", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1), tf.Labels(2))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		wiCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)

		u := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				ID:   &fxt.WorkItems[0].ID,
				Type: APIStringTypeWorkItem,
				Attributes: map[string]interface{}{
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{},
			},
		}
		// attach 2 labels
		apiLabelType := label.APIStringTypeLabels
		lbl0 := fxt.Labels[0].ID.String()
		lbl1 := fxt.Labels[1].ID.String()
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
		_, updatedWI := test.UpdateWorkitemOK(t, svc.Context, svc, wiCtrl, fxt.WorkItems[0].ID, &u)
		assert.NotNil(t, updatedWI)
		require.NotNil(t, updatedWI.Data.Relationships.Labels.Links)
		assert.Len(t, updatedWI.Data.Relationships.Labels.Data, 2)

		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-labels.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-labels.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - iteration", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemIteration: fxt.Iterations[0].ID.String(),
					workitem.SystemVersion:   fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-iteration.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-iteration.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - area", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemArea:    fxt.Areas[0].ID.String(),
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-area.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-area.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list - empty", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-no-event.res.errors.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-no-event.res.errors.headers.golden.json"), res.Header())
	})

	s.T().Run("many events", func(t *testing.T) {
		fxt := tf.NewTestFixture(s.T(), s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1), tf.Iterations(2))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))
		payload1 := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemIteration: fxt.Iterations[0].ID.String(),
					workitem.SystemVersion:   fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload1) // update iteration
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)

		assignee := []string{fxt.Identities[0].ID.String()}
		payload2 := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemAssignees: assignee,
					workitem.SystemVersion:   fxt.WorkItems[0].Version + 1,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload2) // update assignee
		res, eventList = test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 2)

		payload3 := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemIteration: fxt.Iterations[1].ID.String(),
					workitem.SystemVersion:   fxt.WorkItems[0].Version + 2,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload3) // update iteration
		res, eventList = test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 3)
	})
}
