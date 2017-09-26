package workitem_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	tf "github.com/fabric8-services/fabric8-wit/test/testfixture"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

type BenchWorkItemRepository struct {
	gormbench.DBBenchSuite
	repo workitem.WorkItemRepository
}

func BenchmarkRunWorkItemRepository(b *testing.B) {
	testsupport.Run(b, &BenchWorkItemRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchWorkItemRepository) SetupBenchmark() {
	s.DBBenchSuite.SetupBenchmark()
	s.repo = workitem.NewWorkItemRepository(s.DB)
}

func (r *BenchWorkItemRepository) BenchmarkLoadWorkItem() {
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItems(1))
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, err := r.repo.LoadByID(context.Background(), fxt.WorkItems[0].ID); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemRepository) BenchmarkListWorkItems() {
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItems(10))
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, _, err := r.repo.List(context.Background(), fxt.WorkItems[0].SpaceID, criteria.Literal(true), nil, nil, nil); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemRepository) BenchmarkListWorkItemsTransaction() {
	fxt := tf.NewTestFixture(r.B(), r.DB, tf.WorkItems(10))
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if err := application.Transactional(gormapplication.NewGormDB(r.DB), func(app application.Application) error {
			_, _, err := r.repo.List(context.Background(), fxt.WorkItems[0].SpaceID, criteria.Literal(true), nil, nil, nil)
			return err
		}); err != nil {
			r.B().Fail()
		}
	}
}
