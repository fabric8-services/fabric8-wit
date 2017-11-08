package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
)

type BenchPlannerBacklogREST struct {
	gormbench.DBBenchSuite
	testIdentity account.Identity
	testSpace    *space.Space
	svc          *goa.Service
}

func BenchRunPlannerBacklogREST(b *testing.B) {
	resource.Require(b, resource.Database)
	testsupport.Run(b, new(BenchPlannerBacklogREST))
}

func (rest *BenchPlannerBacklogREST) SetupBenchmark() {
	rest.DBBenchSuite.SetupBenchmark()
	// create a test identity
	var err error
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "BenchPlannerBacklogREST user", "test provider")
	if err != nil {
		rest.B().Fail()
	}
	rest.testIdentity = *testIdentity
	rest.svc = goa.New("PlannerBacklog-Service")
	rest.testSpace, _, _ = rest.setupPlannerBacklogWorkItems()
}

func (rest *BenchPlannerBacklogREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	fxt := tf.NewTestFixture(rest.B(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Iterations(2, tf.SetIterationNames("root", "child")),
		tf.WorkItems(2,
			tf.SetWorkItemField(workitem.SystemState, workitem.SystemStateNew, workitem.SystemStateClosed),
			tf.SetWorkItemIterationsByName("parent", "child"),
		),
	)
	return fxt.Spaces[0], fxt.IterationByName("parent"), fxt.WorkItems[1]
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
