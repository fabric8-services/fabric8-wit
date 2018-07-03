package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
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

func (rest *TestPlannerBacklogREST) unSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, rest.GormDB, rest.Configuration)
}

func (rest *TestPlannerBacklogREST) TestCountPlannerBacklogWorkItemsOK() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB,
		tf.Spaces(1),
		tf.Iterations(2,
			tf.SetIterationNames("Parent Iteration", "Child Iteration"),
			func(fxt *tf.TestFixture, idx int) error {
				switch idx {
				case 0:
					fxt.Iterations[idx].State = iteration.StateNew
				case 1:
					fxt.Iterations[idx].State = iteration.StateStart
				}
				return nil
			},
		),
		tf.WorkItemTypes(1, func(fxt *tf.TestFixture, idx int) error {
			fxt.WorkItemTypes[idx].Extends = workitem.SystemPlannerItem
			return nil
		}),
		tf.WorkItems(2, func(fxt *tf.TestFixture, idx int) error {
			wi := fxt.WorkItems[idx]
			switch idx {
			case 0:
				wi.Fields[workitem.SystemTitle] = "parentIteration Test"
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Parent Iteration").ID.String()
				wi.Fields[workitem.SystemState] = workitem.SystemStateNew
			case 1:
				wi.Fields[workitem.SystemTitle] = "childIteration Test"
				wi.Fields[workitem.SystemIteration] = fxt.IterationByName("Child Iteration").ID.String()
				wi.Fields[workitem.SystemState] = workitem.SystemStateClosed
			}
			return nil
		}),
	)
	svc, _ := rest.unSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, rest.GormDB, fxt.Spaces[0].ID)
	// we expect the count to be equal to 1
	require.NoError(rest.T(), err)
	assert.Equal(rest.T(), 1, count)
}

func (rest *TestPlannerBacklogREST) TestCountZeroPlannerBacklogWorkItemsOK() {
	// given
	fxt := tf.NewTestFixture(rest.T(), rest.DB, tf.Spaces(1), tf.Iterations(1))
	svc, _ := rest.unSecuredController()
	// when
	count, err := countBacklogItems(svc.Context, rest.GormDB, fxt.Spaces[0].ID)
	// we expect the count to be equal to 0
	require.NoError(rest.T(), err)
	assert.Equal(rest.T(), 0, count)
}
