package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
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
	require.NoError(rest.T(), err)
	rest.testIdentity = *testIdentity
}

func (rest *TestPlannerBacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *TestPlannerBacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name:    "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
			OwnerID: rest.testIdentity.ID,
		}
		_, err := spacesRepo.Create(rest.Ctx, testSpace)
		require.NoError(rest.T(), err)
		require.NotNil(rest.T(), testSpace.ID)
		log.Info(nil, map[string]interface{}{"space_id": testSpace.ID}, "created space")
		workitemTypesRepo := app.WorkItemTypes()
		workitemType, err := workitemTypesRepo.Create(rest.Ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
		require.NoError(rest.T(), err)
		log.Info(nil, map[string]interface{}{"wit_id": workitemType.ID}, "created workitem type")

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.StateNew,
		}
		iterationsRepo.Create(rest.Ctx, parentIteration)
		log.Info(nil, map[string]interface{}{"parent_iteration_id": parentIteration.ID}, "created parent iteration")

		childIteration := &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.StateStart,
		}
		iterationsRepo.Create(rest.Ctx, childIteration)
		log.Info(nil, map[string]interface{}{"child_iteration_id": childIteration.ID}, "created child iteration")

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.Ctx, testSpace.ID, workitemType.ID, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		createdWI, err = app.WorkItems().Create(rest.Ctx, testSpace.ID, workitemType.ID, fields2, rest.testIdentity.ID)
		require.NoError(rest.T(), err)
		return nil
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
	require.NoError(rest.T(), err)
	assert.Equal(rest.T(), 1, count)
}

func (rest *TestPlannerBacklogREST) TestCountZeroPlannerBacklogWorkItemsOK() {
	// given
	var spaceCount *space.Space
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		spaceCount = &space.Space{
			Name:    "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
			OwnerID: rest.testIdentity.ID,
		}
		_, err := spacesRepo.Create(rest.Ctx, spaceCount)
		require.NoError(rest.T(), err)

		return nil
	})
	svc, _ := rest.UnSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, gormapplication.NewGormDB(rest.DB), spaceCount.ID)
	// we expect the count to be equal to 0
	require.NoError(rest.T(), err)
	assert.Equal(rest.T(), 0, count)
}
