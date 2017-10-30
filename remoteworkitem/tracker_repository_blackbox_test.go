package remoteworkitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/stretchr/testify/suite"
)

type trackerRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo remoteworkitem.TrackerRepository
}

func TestRunTrackerRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerRepoBlackBoxTest) SetupTest() {
	s.DBTestSuite.SetupTest()
	s.repo = remoteworkitem.NewTrackerRepository(s.DB)
}
