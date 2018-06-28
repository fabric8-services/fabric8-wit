package suite

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/stretchr/testify/suite"
)

// UnitTestSuite is a base for unit tests
type UnitTestSuite struct {
	suite.Suite
	Config *configuration.Registry
}

// SetupSuite implements suite.SetupAllSuite
func (s *UnitTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.UnitTest)
	s.setupConfig()
}

func (s *UnitTestSuite) setupConfig() {
	config, err := configuration.Get()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}
	s.Config = config
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *UnitTestSuite) TearDownSuite() {
	s.Config = nil // Summon the GC!
}
