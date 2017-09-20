package codebase_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/codebase"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	uuid "github.com/satori/go.uuid"
)

type BenchCodebaseRepository struct {
	gormbench.DBBenchSuite
	repo codebase.Repository
}

func BenchmarkRunCodebaseRepository(b *testing.B) {
	testsupport.Run(b, &BenchCodebaseRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchCodebaseRepository) SetupBenchmark() {
	s.DBBenchSuite.SetupBenchmark()
	s.repo = codebase.NewCodebaseRepository(s.DB)
}

func newCodebase(spaceID uuid.UUID, stackID, lastUsedWorkspace, repotype, url string) *codebase.Codebase {
	return &codebase.Codebase{
		SpaceID:           spaceID,
		Type:              repotype,
		URL:               url,
		StackID:           &stackID,
		LastUsedWorkspace: lastUsedWorkspace,
	}
}

func (s *BenchCodebaseRepository) createCodebase(c *codebase.Codebase) {
	err := s.repo.Create(s.Ctx, c)
	if err != nil {
		s.B().Fail()
	}
}

func (s *BenchCodebaseRepository) BenchmarkCreateCodebases() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		codebase2 := newCodebase(space.SystemSpace, "python-default", "my-used-last-workspace", "git", "git@github.com:aslakknutsen/fabric8-wit.git")
		s.createCodebase(codebase2)
	}
}

func (s *BenchCodebaseRepository) BenchmarkListCodebases() {
	// given
	codebase := newCodebase(space.SystemSpace, "java-default", "my-used-last-workspace", "git", "git@github.com:aslakknutsen/fabric8-wit.git")
	s.createCodebase(codebase)
	// when
	offset := 0
	limit := 1
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		if codebases, _, err := s.repo.List(s.Ctx, space.SystemSpace, &offset, &limit); err != nil || (err == nil && len(codebases) == 0) {
			s.B().Fail()
		}
	}
}

func (s *BenchCodebaseRepository) BenchmarkLoadCodebase() {
	// given
	codebaseTest := newCodebase(space.SystemSpace, "golang-default", "my-used-hector-workspace", "git", "git@github.com:hectorj2f/fabric8-wit.git")
	s.createCodebase(codebaseTest)
	// when
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		if loadedCodebase, err := s.repo.Load(s.Ctx, codebaseTest.ID); err != nil || (err == nil && loadedCodebase == nil) {
			s.B().Fail()
		}
	}
}
