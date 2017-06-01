package benchmark

import (
	"os"

	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver

	"golang.org/x/net/context"
)

var _ test.SetupAllSuite = &DBBenchSuite{}
var _ test.TearDownAllSuite = &DBBenchSuite{}

// NewDBBenchSuite instanciate a new DBBenchSuite
func NewDBBenchSuite(configFilePath string) DBBenchSuite {
	return DBBenchSuite{configFile: configFilePath}
}

// DBBenchSuite is a base for tests using a gorm db
type DBBenchSuite struct {
	test.Suite
	configFile    string
	Configuration *config.ConfigurationData
	DB            *gorm.DB
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBBenchSuite) SetupSuite() {
	resource.Require(s.B(), resource.Database)
	configuration, err := config.NewConfigurationData(s.configFile)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Configuration = configuration
	if _, c := os.LookupEnv(resource.Database); c != false {
		s.DB, err = gorm.Open("postgres", s.Configuration.GetPostgresConfigString())
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": configuration.GetPostgresConfigString(),
			}, "failed to connect to the database")
		}
	}
}

// PopulateDBBenchSuite populates the DB with common values
func (s *DBBenchSuite) PopulateDBBenchSuite(ctx context.Context) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": s.Configuration.GetPostgresConfigString(),
			}, "failed to populate the database with common types")
		}
	}
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBBenchSuite) TearDownSuite() {
	s.DB.Close()
}
