package application_test

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
)

type BenchTransactional struct {
	gormbench.DBBenchSuite
	repo  space.Repository
	appDB application.DB
	dbPq  *sql.DB
}

func BenchmarkRunTransactional(b *testing.B) {
	testsupport.Run(b, &BenchTransactional{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchTransactional) SetupBenchmark() {
	s.DBBenchSuite.SetupBenchmark()
	s.repo = space.NewRepository(s.DB)
	s.appDB = gormapplication.NewGormDB(s.DB)
}

func (s *BenchTransactional) transactionLoadSpace() {
	err := application.Transactional(s.appDB, func(appl application.Application) error {
		_, err := s.repo.Load(s.Ctx, space.SystemSpace)
		return err
	})
	if err != nil {
		s.B().Fail()
	}
}

func (s *BenchTransactional) BenchmarkApplTransaction() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		s.transactionLoadSpace()
	}
}
