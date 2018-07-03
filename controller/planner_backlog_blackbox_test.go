package controller_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBacklogBlackboxREST struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
}

func TestRunPlannerBacklogBlackboxREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBacklogBlackboxREST))
}

func (rest *TestPlannerBacklogBlackboxREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogBlackboxREST user", "test provider")
	require.NoError(rest.T(), err)
	rest.testIdentity = *testIdentity
}

func (rest *TestPlannerBacklogBlackboxREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestPlannerBacklogBlackboxREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.Spaces(1),
		tf.Iterations(2, func(fxt *tf.TestFixture, idx int) error {
			i := fxt.Iterations[idx]
			switch idx {
			case 0:
				i.Name = "parent"
				i.State = iteration.StateNew
			case 1:
				i.Name = "child"
				i.Path = append(fxt.Iterations[0].Path, fxt.Iterations[0].ID)
				i.State = iteration.StateStart
			}
			return nil
		}),
		tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			switch idx {
			case 0:
				wi.Fields[workitem.SystemTitle] = "parentIteration Test"
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("parent").ID.String()
				// wi.Type = workitem.SystemPlannerItem
			case 1:
				wi.Fields[workitem.SystemTitle] = "childIteration Test"
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("child").ID.String()
				// wi.Type = workitem.SystemPlannerItem
			}
			return nil
		}),
	)
	testSpace = fxt.Spaces[0]
	parentIteration = fxt.IterationByName("parent")
	createdWI = fxt.WorkItemByTitle("childIteration Test")
	require.NotNil(rest.T(), testSpace)
	require.NotNil(rest.T(), parentIteration)
	require.NotNil(rest.T(), createdWI)
	return testSpace, parentIteration, createdWI
}

func assertPlannerBacklogWorkItems(t *testing.T, workitems *app.WorkItemList, testSpace *space.Space, parentIteration *iteration.Iteration) {
	// Two iteration have to be found
	require.NotNil(t, workitems)
	assert.Len(t, workitems.Data, 1)
	for _, workItem := range workitems.Data {
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, testSpace.ID.String(), workItem.Relationships.Space.Data.ID.String())
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, workitem.SystemStateNew, workItem.Attributes[workitem.SystemState])
		assert.Equal(t, parentIteration.ID.String(), *workItem.Relationships.Iteration.Data.ID)
	}
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOK() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfModifiedSinceHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	rest.T().Log("Test Space: " + testSpace.ID.String())
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifModifiedSince := app.ToHTTPTime(parentIteration.UpdatedAt.Add(-1 * time.Hour))
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfNoneMatchHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifNoneMatch := "foo"
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfModifiedSinceHeader() {
	// given
	testSpace, _, lastWorkItem := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	updatedAt, ok := lastWorkItem.Fields[workitem.SystemUpdatedAt]
	require.True(rest.T(), ok)
	updatedAtTime, ok := updatedAt.(time.Time)
	require.True(rest.T(), ok)
	ifModifiedSince := app.ToHTTPTime(updatedAtTime)
	res := test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfNoneMatchHeader() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	filter := ""
	limit := -1
	res, _ := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	res = test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestSuccessEmptyListPlannerBacklogWorkItems() {
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Iterations(2),
		tf.WorkItems(1, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			wi.Fields[workitem.SystemTitle] = "parentIteration Test"
			wi.Fields[workitem.SystemState] = workitem.SystemStateNew
			wi.Fields[workitem.SystemIteration] = fxt.Iterations[1].ID.String()
			return nil
		}),
	)

	svc, ctrl := rest.UnSecuredController()

	offset := "0"
	filter := ""
	limit := -1
	_, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, fxt.Spaces[0].ID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// The list has to be empty
	assert.Len(rest.T(), workitems.Data, 0)
}
