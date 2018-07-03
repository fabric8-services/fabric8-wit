package gormtestsupport

import (
	"os"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/resource"

	"context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// NewDBTestSuite instanciate a new DBTestSuite
func NewDBTestSuite() DBTestSuite {
	return DBTestSuite{}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	Configuration *config.Registry
	DB            *gorm.DB
	GormDB        *gormapplication.GormDB
	clean         func()
	Ctx           context.Context
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	// Get default configuration
	configuration, err := config.Get()
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
		s.GormDB = gormapplication.NewGormDB(s.DB)
	}
	s.Ctx = migration.NewMigrationContext(context.Background())
	s.populateDBTestSuite(s.Ctx)
}

// SetupTest implements suite.SetupTest
func (s *DBTestSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

// TearDownTest implements suite.TearDownTest
func (s *DBTestSuite) TearDownTest() {
	if s.clean != nil {
		s.clean()
	}
}

// populateDBTestSuite populates the DB with common values
func (s *DBTestSuite) populateDBTestSuite(ctx context.Context) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx)
		})
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": s.Configuration.GetPostgresConfigString(),
			}, "failed to populate the database with common types")
		}
	}
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *DBTestSuite) TearDownSuite() {
	s.DB.Close()
}
