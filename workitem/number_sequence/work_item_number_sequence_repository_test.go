package numbersequence_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	. "github.com/fabric8-services/fabric8-wit/workitem/number_sequence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type workItemNumberSequenceTest struct {
	gormtestsupport.DBTestSuite
	repo WorkItemNumberSequenceRepository
}

func TestWorkItemNumberSequenceTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemNumberSequenceTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *workItemNumberSequenceTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = NewWorkItemNumberSequenceRepository(s.DB)
}

func (s *workItemNumberSequenceTest) TestConcurrentNextVal() {
	// given
	fxt := tf.NewTestFixture(s.T(), s.DB, tf.Spaces(1))
	type Report struct {
		id       int
		total    int
		failures int
	}
	routines := 10
	itemsPerRoutine := 50
	reports := make([]Report, routines)
	// when running concurrent go routines simultaneously
	var wg sync.WaitGroup
	for i := 0; i < routines; i++ {
		wg.Add(1)
		// in each go rountine, run 10 creations
		go func(routineID int) {
			defer wg.Done()
			report := Report{id: routineID}
			for j := 0; j < itemsPerRoutine; j++ {
				if err := application.Transactional(gormapplication.NewGormDB(s.DB), func(app application.Application) error {
					_, err := s.repo.NextVal(context.Background(), fxt.Spaces[0].ID)
					return err
				}); err != nil {
					s.T().Logf("Creation failed: %s", err.Error())
					report.failures++
				}
				report.total++
			}
			reports[routineID] = report
		}(i)
	}
	wg.Wait()
	// then
	// wait for all items to be created
	for _, report := range reports {
		fmt.Printf("Routine #%d done: %d creations, including %d failure(s)\n", report.id, report.total, report.failures)
		assert.Equal(s.T(), itemsPerRoutine, report.total)
		assert.Equal(s.T(), 0, report.failures)
	}

}
