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
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
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
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-state.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-state.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - title", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindInt.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-kindInt.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - description", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))

		modifiedDescription := "# Description is modified1"
		modifiedMarkup := rendering.SystemMarkupMarkdown

		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemDescription:       modifiedDescription,
					workitem.SystemDescriptionMarkup: modifiedMarkup,
					workitem.SystemVersion:           fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-description.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-description.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - assigned", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
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

		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-labels.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-labels.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - iteration", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-iteration.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-iteration.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list ok - area", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
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
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-area.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-area.res.headers.golden.json"), res.Header())
	})

	s.T().Run("event list - empty", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-no-event.res.errors.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-no-event.res.errors.headers.golden.json"), res.Header())
	})

	s.T().Run("many events", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1), tf.Iterations(2))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))

		t.Run("1st update", func(t *testing.T) {
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
			test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload) // update iteration
			_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
			require.NotEmpty(t, eventList)
			require.Len(t, eventList.Data, 1)
		})

		var secondUpdateRevID uuid.UUID

		t.Run("2nd update", func(t *testing.T) {
			newAssignee := []string{fxt.Identities[0].ID.String()}
			newTitle := "The Bare Necessities"
			payload := app.UpdateWorkitemPayload{
				Data: &app.WorkItem{
					Type: APIStringTypeWorkItem,
					ID:   &fxt.WorkItems[0].ID,
					Attributes: map[string]interface{}{
						workitem.SystemAssignees: newAssignee,
						workitem.SystemTitle:     newTitle,
						workitem.SystemVersion:   fxt.WorkItems[0].Version + 1,
					},
					Relationships: &app.WorkItemRelationships{
						Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
					},
				},
			}
			test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload) // update assignee
			_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
			require.NotEmpty(t, eventList)
			require.Len(t, eventList.Data, 3)
			secondUpdateRevID = eventList.Data[2].Attributes.RevisionID
		})

		t.Run("3rd update", func(t *testing.T) {
			payload := app.UpdateWorkitemPayload{
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
			test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload) // update iteration
			_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
			require.NotEmpty(t, eventList)
			require.Len(t, eventList.Data, 4)
		})

		t.Run("ensure we can list events produced just by the 2nd revision", func(t *testing.T) {
			_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, &secondUpdateRevID, nil, nil)
			require.NotEmpty(t, eventList)
			require.Len(t, eventList.Data, 2)
			toBeFound := map[string]struct{}{
				workitem.SystemAssignees: {},
				workitem.SystemTitle:     {},
			}
			for _, e := range eventList.Data {
				require.Equal(t, secondUpdateRevID, e.Attributes.RevisionID, "wrong revision ID")
				_, ok := toBeFound[e.Attributes.Name]
				require.True(t, ok, "found unexpected event name: %s", e.Attributes.Name)
				delete(toBeFound, e.Attributes.Name)
			}
			require.Empty(t, toBeFound, "failed to find event for these fields: %+v", toBeFound)
		})
	})

	s.T().Run("one revision results in two events", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1), tf.Iterations(2))

		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		eventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))

		// given two fields that we want to update
		newAssignees := []string{fxt.Identities[0].ID.String()}
		newIteration := fxt.Iterations[0].ID.String()

		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemIteration: newIteration,
					workitem.SystemAssignees: newAssignees,
					workitem.SystemVersion:   fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		_, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, eventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 2)

		assert.Equal(t, eventList.Data[0].Attributes.RevisionID, eventList.Data[1].Attributes.RevisionID, "revision IDs must be the same across the two events")
		assert.NotEqual(t, eventList.Data[0].ID, eventList.Data[1].ID, "event IDs must be unique")
	})

	s.T().Run("non-relational field kinds", func(t *testing.T) {
		testData := workitem.GetFieldTypeTestData(t)
		for _, kind := range testData.GetKinds() {
			if !kind.IsSimpleType() || kind.IsRelational() {
				continue
			}

			// TODO(kwk): Once the new type system enhancements are in, also
			// test instant fields
			if kind == workitem.KindInstant {
				continue
			}

			fieldNameSingle := kind.String() + "_single"
			fieldNameList := kind.String() + "_list"
			// NOTE(kwk): Leave this commented out until we have proper test data
			// fieldNameEnum := kind.String() + "_enum"

			fxt := tf.NewTestFixture(t, s.DB,
				tf.CreateWorkItemEnvironment(),
				tf.WorkItemTypes(2, func(fxt *tf.TestFixture, idx int) error {
					switch idx {
					case 0:
						fxt.WorkItemTypes[idx].Fields[fieldNameSingle] = workitem.FieldDefinition{
							Label:       fieldNameSingle,
							Description: "A single value of a " + kind.String() + " object",
							Type:        workitem.SimpleType{Kind: kind},
						}
					case 1:
						fxt.WorkItemTypes[idx].Fields[fieldNameList] = workitem.FieldDefinition{
							Label:       fieldNameList,
							Description: "An array of " + kind.String() + " objects",
							Type: workitem.ListType{
								SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
								ComponentType: workitem.SimpleType{Kind: kind},
							},
						}
						// NOTE(kwk): Leave this commented out until we have proper test data
						// case 3:
						// fxt.WorkItemTypes[idx].Fields[fieldNameEnum] = workitem.FieldDefinition{
						// 	Label:       fieldNameEnum,
						// 	Description: "An enum value of a " + kind.String() + " object",
						// 	Type: workitem.EnumType{
						// 		SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
						// 		BaseType:   workitem.SimpleType{Kind: kind},
						// 		Values: []interface{}{
						// 			testData[kind].Valid[0],
						// 			testData[kind].Valid[1],
						// 		},
						// 	},
						// }
					}
					return nil
				}),
				tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
					fxt.WorkItems[idx].Type = fxt.WorkItemTypes[idx].ID
					return nil
				}),
			)
			svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
			EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
			workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
			spaceSelfURL := rest.AbsoluteURL(&http.Request{Host: "api.service.domain.org"}, app.SpaceHref(fxt.Spaces[0].ID.String()))

			t.Run(fieldNameSingle, func(t *testing.T) {
				// NOTE(kwk): Leave this commented out until we have proper test data
				// fieldDef := fxt.WorkItemTypes[0].Fields[fieldNameSingle]
				// val, err := fieldDef.ConvertFromModel(fieldNameSingle, testData[kind].Valid[0])
				// require.NoError(t, err)
				newValue := testData[kind].Valid[0]
				payload := app.UpdateWorkitemPayload{
					Data: &app.WorkItem{
						Type: APIStringTypeWorkItem,
						ID:   &fxt.WorkItems[0].ID,
						Attributes: map[string]interface{}{
							fieldNameSingle:        newValue,
							workitem.SystemVersion: fxt.WorkItems[0].Version,
						},
						Relationships: &app.WorkItemRelationships{
							Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
						},
					},
				}
				// update work item once
				test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
				// update it twice
				payload.Data.Attributes[workitem.SystemVersion] = fxt.WorkItems[0].Version + 1
				payload.Data.Attributes[fieldNameSingle] = testData[kind].Valid[1]
				test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)

				res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
				safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
				require.NotEmpty(t, eventList)
				require.Len(t, eventList.Data, 2)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok."+fieldNameSingle+".res.payload.golden.json"), eventList)
			})
			t.Run(fieldNameList, func(t *testing.T) {
				// NOTE(kwk): Leave this commented out until we have proper test data
				// listDef := fxt.WorkItemTypes[0].Fields[fieldNameList]
				// fieldDef, ok := listDef.Type.(workitem.ListType)
				// require.True(t, ok, "failed to cast %+v (%[1]T) to workitem.ListType", listDef)
				// vals, err := fieldDef.ConvertFromModel([]interface{}{testData[kind].Valid[0], testData[kind].Valid[1]})
				// require.NoError(t, err)
				newValue := []interface{}{testData[kind].Valid[0], testData[kind].Valid[1]}
				payload := app.UpdateWorkitemPayload{
					Data: &app.WorkItem{
						Type: APIStringTypeWorkItem,
						ID:   &fxt.WorkItems[1].ID,
						Attributes: map[string]interface{}{
							fieldNameList:          newValue,
							workitem.SystemVersion: fxt.WorkItems[1].Version,
						},
						Relationships: &app.WorkItemRelationships{
							Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
						},
					},
				}
				// update work item once
				test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[1].ID, &payload)
				// update it twice
				payload.Data.Attributes[workitem.SystemVersion] = fxt.WorkItems[1].Version + 1
				payload.Data.Attributes[fieldNameList] = []interface{}{testData[kind].Valid[1], testData[kind].Valid[0]}
				test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[1].ID, &payload)
				res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[1].ID, nil, nil, nil)
				safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
				require.NotEmpty(t, eventList)
				require.Len(t, eventList.Data, 2)
				compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok."+fieldNameList+".res.payload.golden.json"), eventList)
			})
			// NOTE(kwk): Leave this commented out until we have proper test data
			// TODO(kwk): Once the new type system enhancements are in, also
			// test for enum fields here.

			// t.Run(fieldNameEnum, func(t *testing.T) {
			// 	// NOTE(kwk): Leave this commented out until we have proper test data
			// 	// listDef := fxt.WorkItemTypes[0].Fields[fieldNameEnum]
			// 	// fieldDef, ok := listDef.Type.(workitem.EnumType)
			// 	// require.True(t, ok, "failed to cast %+v (%[1]T) to workitem.EnumType", listDef)
			// 	// val, err := fieldDef.ConvertFromModel(testData[kind].Valid[0])
			// 	// require.NoError(t, err)

			// 	// we have to use the second value because we default to the
			// 	// first one upon creation of the work item.
			// 	newValue := testData[kind].Valid[1]
			// 	payload := app.UpdateWorkitemPayload{
			// 		Data: &app.WorkItem{
			// 			Type: APIStringTypeWorkItem,
			// 			ID:   &fxt.WorkItems[2].ID,
			// 			Attributes: map[string]interface{}{
			// 				fieldNameEnum:          newValue,
			// 				workitem.SystemVersion: fxt.WorkItems[2].Version,
			// 			},
			// 			Relationships: &app.WorkItemRelationships{
			// 				Space: app.NewSpaceRelation(fxt.Spaces[0].ID, spaceSelfURL),
			// 			},
			// 		},
			// 	}
			// // update work item once
			// test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[2].ID, &payload)
			// // update it twice
			// payload.Data.Attributes[workitem.SystemVersion] = fxt.WorkItems[2].Version + 1
			// payload.Data.Attributes[fieldNameEnum] = testData[kind].Valid[1]
			// test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[2].ID, &payload)
			// 	res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[2].ID, nil, nil)
			// 	safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
			// 	require.NotEmpty(t, eventList)
			// 	require.Len(t, eventList.Data, 2)
			// 	compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok."+fieldNameEnum+".res.payload.golden.json"), eventList)
			// 	// compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok."+fieldNameEnum+".res.headers.golden.json"), res.Header())
			// })
		}
	})

	s.T().Run("event list ok - workitem type change", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, s.DB, tf.CreateWorkItemEnvironment(), tf.WorkItems(1), tf.WorkItemTypes(2))
		svc := testsupport.ServiceAsSpaceUser("Event-Service", *fxt.Identities[0], &TestSpaceAuthzService{*fxt.Identities[0], ""})
		EventCtrl := NewEventsController(svc, s.GormDB, s.Configuration)
		workitemCtrl := NewWorkitemController(svc, s.GormDB, s.Configuration)
		payload := app.UpdateWorkitemPayload{
			Data: &app.WorkItem{
				Type: APIStringTypeWorkItem,
				ID:   &fxt.WorkItems[0].ID,
				Attributes: map[string]interface{}{
					workitem.SystemVersion: fxt.WorkItems[0].Version,
				},
				Relationships: &app.WorkItemRelationships{
					BaseType: &app.RelationBaseType{
						Data: &app.BaseTypeData{
							ID:   fxt.WorkItemTypes[1].ID,
							Type: APIStringTypeWorkItemType,
						},
					},
				},
			},
		}
		test.UpdateWorkitemOK(t, svc.Context, svc, workitemCtrl, fxt.WorkItems[0].ID, &payload)
		res, eventList := test.ListWorkItemEventsOK(t, svc.Context, svc, EventCtrl, fxt.WorkItems[0].ID, nil, nil, nil)
		safeOverriteHeader(t, res, app.ETag, "1GmclFDDPcLR1ZWPZnykWw==")
		require.NotEmpty(t, eventList)
		require.Len(t, eventList.Data, 1)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-witype-change.res.payload.golden.json"), eventList)
		compareWithGoldenAgnostic(t, filepath.Join(s.testDir, "list", "ok-witype-change.res.headers.golden.json"), res.Header())
	})
}
