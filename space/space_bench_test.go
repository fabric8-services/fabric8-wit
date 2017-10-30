package space_test

import (
	"context"
	"testing"

	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/test"
	uuid "github.com/satori/go.uuid"
)

func BenchmarkRunRepoBBBench(b *testing.B) {
	test.Run(b, &repoSpaceBench{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

type repoSpaceBench struct {
	gormbench.DBBenchSuite
	repo space.Repository
}

func (bench *repoSpaceBench) SetupBenchmark() {
	bench.repo = space.NewRepository(bench.DB)
}

func (bench *repoSpaceBench) BenchmarkCreate() {
	// given
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		newSpace := space.Space{
			Name:    test.CreateRandomValidTestName("BenchmarkCreate"),
			OwnerID: uuid.Nil,
		}
		if s, err := bench.repo.Create(context.Background(), &newSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceByName() {
	newSpace := space.Space{
		Name:    test.CreateRandomValidTestName("BenchmarkLoadSpaceByName"),
		OwnerID: uuid.Nil,
	}
	if s, err := bench.repo.Create(context.Background(), &newSpace); err != nil || (err == nil && s == nil) {
		bench.B().Fail()
	}

	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.LoadByOwnerAndName(context.Background(), &newSpace.OwnerID, &newSpace.Name); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceById() {
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.Load(context.Background(), space.SystemSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkList() {
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, _, err := bench.repo.List(context.Background(), nil, nil); err != nil || (err == nil && len(s) == 0) {
			bench.B().Fail()
		}
	}
}
