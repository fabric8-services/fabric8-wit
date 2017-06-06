package controller

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	gormbench "github.com/almighty/almighty-core/gormtestsupport/benchmark"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

type BenchPlannerBacklogREST struct {
	gormbench.DBBenchSuite
	clean        func()
	testIdentity account.Identity
	ctx          context.Context
	testSpace    *space.Space
	svc          *goa.Service
}

func BenchRunPlannerBacklogREST(b *testing.B) {
	resource.Require(b, resource.Database)
	testsupport.Run(b, new(BenchPlannerBacklogREST))
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *BenchPlannerBacklogREST) SetupSuite() {
	rest.DBBenchSuite.SetupSuite()
	rest.ctx = migration.NewMigrationContext(context.Background())
	rest.DBBenchSuite.PopulateDBBenchSuite(rest.ctx)
}

func (rest *BenchPlannerBacklogREST) SetupBenchmark() {
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	// create a test identity
	var err error
	rest.testIdentity, err = testsupport.CreateTestIdentity(rest.DB, "BenchPlannerBacklogREST user", "test provider")
	if err != nil {
		rest.B().Fail()
	}
	rest.svc = goa.New("PlannerBacklog-Service")
	rest.testSpace, _, _ = rest.setupPlannerBacklogWorkItems()
}

func (rest *BenchPlannerBacklogREST) TearDownBenchmark() {
	rest.clean()
}

func (rest *BenchPlannerBacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.ctx, testSpace)
		if err != nil {
			rest.B().Fail()
		}
		log.Info(nil, map[string]interface{}{"space_id": testSpace.ID}, "created space")
		workitemTypesRepo := app.WorkItemTypes()
		workitemType, err := workitemTypesRepo.Create(rest.ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
		if err != nil {
			rest.B().Fail()
		}
		log.Info(nil, map[string]interface{}{"wit_id": workitemType.ID}, "created workitem type")

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)
		log.Info(nil, map[string]interface{}{"parent_iteration_id": parentIteration.ID}, "created parent iteration")

		childIteration := &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.IterationStateStart,
		}
		iterationsRepo.Create(rest.ctx, childIteration)
		log.Info(nil, map[string]interface{}{"child_iteration_id": childIteration.ID}, "created child iteration")

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
		if err != nil {
			rest.B().Fail()
		}
		return nil
	})
	return
}

func (rest *BenchPlannerBacklogREST) BenchmarkCountPlannerBacklogWorkItemsOK() {
	// given
	rest.B().ResetTimer()
	rest.B().ReportAllocs()
	for n := 0; n < rest.B().N; n++ {
		count, err := countBacklogItems(rest.svc.Context, gormapplication.NewGormDB(rest.DB), rest.testSpace.ID)
		if count != 1 || err != nil {
			rest.B().Fail()
		}
	}
}
