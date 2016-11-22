package gormsupport

import (
	"fmt"
	"os"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	DB *gorm.DB
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	var err error
	if err = configuration.Setup("../config.yaml"); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {
		s.DB, err = gorm.Open("postgres", configuration.GetPostgresConfigString)
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}

	}

}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	s.DB.Close()
}
