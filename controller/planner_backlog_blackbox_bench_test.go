package controller_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
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

type BenchPlannerBacklogBlackboxREST struct {
	gormbench.DBBenchSuite
	testIdentity account.Identity
	svc          *goa.Service
	ctrl         *PlannerBacklogController
}

var testBench *testing.T

func TestRunPlannerBacklogBlackboxBenchREST(t *testing.T) {
	testBench = t
}

func BenchmarkRunPlannerBacklogBlackboxREST(b *testing.B) {
	resource.Require(b, resource.Database)
	testsupport.Run(b, &BenchPlannerBacklogBlackboxREST{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (rest *BenchPlannerBacklogBlackboxREST) SetupSuite() {
	rest.DBBenchSuite.SetupSuite()
	rest.svc = goa.New("PlannerBacklog-Service")
	rest.ctrl = NewPlannerBacklogController(rest.svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *BenchPlannerBacklogBlackboxREST) SetupBenchmark() {
	rest.DBBenchSuite.SetupBenchmark()
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogBlackboxREST user", "test provider")
	if err != nil {
		rest.B().Fail()
	}
	rest.testIdentity = *testIdentity
}

func (rest *BenchPlannerBacklogBlackboxREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	fxt := tf.NewTestFixture(rest.B(), rest.DB,
		tf.CreateWorkItemEnvironment(),
		tf.Iterations(2, tf.SetIterationNames("Parent Iteration", "Child Iteration")),
		tf.WorkItems(2,
			tf.SetWorkItemField(workitem.SystemState, workitem.SystemStateNew, workitem.SystemStateClosed),
			tf.SetWorkItemIterationsByName("Parent Iteration", "Child Iteration"),
		),
	)
	return fxt.Spaces[0], fxt.IterationByName("Parent Iteration"), fxt.WorkItems[1]
}

func (rest *BenchPlannerBacklogBlackboxREST) BenchmarkListPlannerBacklogWorkItemsOK() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	// when
	offset := "0"
	filter := ""
	limit := -1
	// when
	rest.B().ResetTimer()
	rest.B().ReportAllocs()
	for n := 0; n < rest.B().N; n++ {
		if _, workitems := test.ListPlannerBacklogOK(testBench, rest.svc.Context, rest.svc, rest.ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil); len(workitems.Data) != 1 {
			rest.B().Fail()
		}
	}
}
