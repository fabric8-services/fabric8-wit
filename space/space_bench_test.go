package space_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	gormbench "github.com/almighty/almighty-core/gormtestsupport/benchmark"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/test"
	uuid "github.com/satori/go.uuid"
)

var testSpace string = uuid.NewV4().String()

func BenchmarkRunRepoBBBench(b *testing.B) {
	test.Run(b, &repoSpaceBench{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

type repoSpaceBench struct {
	gormbench.DBBenchSuite
	repo  space.Repository
	clean func()
}

func (bench *repoSpaceBench) SetupBenchmark() {
	bench.repo = space.NewRepository(bench.DB)
	bench.clean = cleaner.DeleteCreatedEntities(bench.DB)
}

func (bench *repoSpaceBench) TearDownBenchmark() {
	bench.clean()
}

func (bench *repoSpaceBench) BenchmarkCreate() {
	// given
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		newSpace := space.Space{
			Name:    test.CreateRandomValidTestName("BenchmarkCreate"),
			OwnerId: uuid.Nil,
		}
		if s, err := bench.repo.Create(context.Background(), &newSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceByName() {
	name := "system.space"
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.LoadByOwnerAndName(context.Background(), &uuid.Nil, &name); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceById() {
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.Load(context.Background(), space.SystemSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkList() {
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		if s, _, err := bench.repo.List(context.Background(), nil, nil); err != nil || (err == nil && len(s) == 0) {
			bench.B().Fail()
		}
	}
}
