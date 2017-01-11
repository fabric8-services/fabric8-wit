package gormsupport

import (
	"fmt"
	"os"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

func NewDBTestSuite(configFilePath string) DBTestSuite {
	return DBTestSuite{configFile: configFilePath}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	configFile string
	DB         *gorm.DB
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	var err error
	if err = configuration.Setup(s.configFile); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {
		s.DB, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}

		// Make sure the database is populated with the correct types (e.g. system.bug etc.)
		if configuration.GetPopulateCommonTypes() {
			if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
				return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
			}); err != nil {
				panic(err.Error())
			}
		}
	}

}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	s.DB.Close()
}
