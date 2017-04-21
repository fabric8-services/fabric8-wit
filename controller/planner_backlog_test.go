package controller

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/category"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBacklogREST struct {
	gormtestsupport.DBTestSuite
	clean        func()
	testIdentity account.Identity
	ctx          context.Context
}

func TestRunPlannerBacklogREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBacklogREST))
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *TestPlannerBacklogREST) SetupSuite() {
	rest.DBTestSuite.SetupSuite()
	rest.ctx = migration.NewMigrationContext(context.Background())
	rest.DBTestSuite.PopulateDBTestSuite(rest.ctx)
}

func (rest *TestPlannerBacklogREST) SetupTest() {
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = testIdentity
}

func (rest *TestPlannerBacklogREST) TearDownTest() {
	rest.clean()
}

func (rest *TestPlannerBacklogREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *TestPlannerBacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.ctx, testSpace)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), testSpace.ID)
		logrus.Info("Created space with ID=", testSpace.ID)
		workitemTypesRepo := app.WorkItemTypes()

		categoryID := []*uuid.UUID{}
		categoryID = append(categoryID, &category.PlannerRequirementsID)
		workitemType, err := workitemTypesRepo.Create(rest.ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{}, categoryID)
		require.Nil(rest.T(), err)
		logrus.Info("Created workitem type with ID=", workitemType.ID)

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)
		logrus.Info("Created parent iteration with ID=", parentIteration.ID)

		childIteration := &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.IterationStateStart,
		}
		iterationsRepo.Create(rest.ctx, childIteration)
		logrus.Info("Created child iteration with ID=", childIteration.ID)

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		createdWI, err = app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields2, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
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
	assert.Nil(rest.T(), err)
	assert.Equal(rest.T(), 1, count)
}

func (rest *TestPlannerBacklogREST) TestCountZeroPlannerBacklogWorkItemsOK() {
	// given
	var spaceCount *space.Space
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		spaceCount = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.ctx, spaceCount)
		require.Nil(rest.T(), err)

		return nil
	})
	svc, _ := rest.UnSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, gormapplication.NewGormDB(rest.DB), spaceCount.ID)
	// we expect the count to be equal to 0
	assert.Nil(rest.T(), err)
	assert.Equal(rest.T(), 0, count)
}
