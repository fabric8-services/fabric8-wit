package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBacklogREST struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
}

func TestRunPlannerBacklogREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBacklogREST))
}

func (rest *TestPlannerBacklogREST) SetupTest() {
	rest.DBTestSuite.SetupTest()
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = *testIdentity
}

func (rest *TestPlannerBacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *TestPlannerBacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	rest.T().Run("setupPlannerBacklogWorkItems", func(t *testing.T) {
		fxt := tf.NewTestFixture(t, rest.DB,
			tf.CreateWorkItemEnvironment(),
			tf.Spaces(1, tf.SetSpaceNames("PlannerBacklogWorkItems-"+uuid.NewV4().String())),
			tf.Iterations(2,
				tf.SetIterationNames("Parent Iteration", "Child Iteration"),
				tf.SetIterationStates(iteration.StateNew, iteration.StateStart),
			),
			tf.WorkItems(2,
				tf.SetWorkItemField(workitem.SystemState, workitem.SystemStateNew, workitem.SystemStateClosed),
				tf.SetWorkItemField(workitem.SystemTitle, "parentIteration Test", "childIteration Test"),
				tf.SetWorkItemIterationsByName("Parent Iteration", "Child Iteration"),
			),
		)
		testSpace = fxt.Spaces[0]
		parentIteration = fxt.Iterations[0]
		createdWI = fxt.WorkItems[1]
	})
	return
}

func assertPlannerBacklogWorkItems(t *testing.T, workitems *app.WorkItemList, testSpace *space.Space, parentIteration *iteration.Iteration) {
	// Two iteration have to be found
	require.NotNil(t, workitems)
	assert.Len(t, workitems.Data, 1)
	for _, workItem := range workitems.Data {
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, testSpace.ID.String(), workItem.Relationships.Space.Data.ID.String())
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, "new", workItem.Attributes[workitem.SystemState])
		assert.Equal(t, parentIteration.ID.String(), *workItem.Relationships.Iteration.Data.ID)
	}
}

func (rest *TestPlannerBacklogREST) TestCountPlannerBacklogWorkItemsOK() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	svc, _ := rest.UnSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, gormapplication.NewGormDB(rest.DB), testSpace.ID)
	// we expect the count to be equal to 1
	require.Nil(rest.T(), err)
	assert.Equal(rest.T(), 1, count)
}

func (rest *TestPlannerBacklogREST) TestCountZeroPlannerBacklogWorkItemsOK() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Spaces(1))
	spaceCount := fxt.Spaces[0]
	svc, _ := rest.UnSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, gormapplication.NewGormDB(rest.DB), spaceCount.ID)
	// we expect the count to be equal to 0
	assert.Nil(rest.T(), err)
	assert.Equal(rest.T(), 0, count)
}
