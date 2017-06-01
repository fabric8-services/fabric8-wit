package area_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	gormbench "github.com/almighty/almighty-core/gormtestsupport/benchmark"

	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/test"
)

type BenchmarkAreaRepository struct {
	gormbench.DBBenchSuite
	repo  area.Repository
	clean func()
}

func BenchmarkRunAreaRepository(b *testing.B) {
	resource.Require(b, resource.Database)
	test.Run(b, &BenchmarkAreaRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (bench *BenchmarkAreaRepository) SetupTest() {
	bench.clean = cleaner.DeleteCreatedEntities(bench.DB)
	bench.repo = area.NewAreaRepository(bench.DB)
}

func (bench *BenchmarkAreaRepository) TearDownTest() {
	bench.clean()
}

func (bench *BenchmarkAreaRepository) BenchmarkRootArea() {
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		if a, err := bench.repo.Root(context.Background(), space.SystemSpace); err != nil || (err == nil || a == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *BenchmarkAreaRepository) BenchmarkCreateArea() {
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		a := area.Area{
			Name:    "TestCreateArea",
			SpaceID: space.SystemSpace,
		}
		if err := bench.repo.Create(context.Background(), &a); err != nil {
			bench.B().Fail()
		}
	}
}

func (bench *BenchmarkAreaRepository) BenchmarkListAreaBySpace() {
	if err := bench.repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space.SystemSpace,
	}); err != nil {
		bench.B().Fail()
	}
	// when
	bench.B().ResetTimer()
	for n := 0; n < bench.B().N; n++ {
		if its, err := bench.repo.List(context.Background(), space.SystemSpace); err != nil || (err == nil && len(its) == 0) {
			bench.B().Fail()
		}
	}
}
