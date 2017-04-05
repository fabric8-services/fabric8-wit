package gormtestsupport

import (
	"github.com/Sirupsen/logrus"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &RemoteTestSuite{}
var _ suite.TearDownAllSuite = &RemoteTestSuite{}

// NewRemoteTestSuite instanciate a new RemoteTestSuite
func NewRemoteTestSuite(configFilePath string) RemoteTestSuite {
	return RemoteTestSuite{configFile: configFilePath}
}

// RemoteTestSuite is a base for tests using a gorm Remote
type RemoteTestSuite struct {
	suite.Suite
	configFile    string
	Configuration *config.ConfigurationData
}

// SetupSuite implements suite.SetupAllSuite
func (s *RemoteTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Remote)
	configuration, err := config.NewConfigurationData(s.configFile)
	if err != nil {
		logrus.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Configuration = configuration
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *RemoteTestSuite) TearDownSuite() {
	s.Configuration = nil // Summon the GC!
}
