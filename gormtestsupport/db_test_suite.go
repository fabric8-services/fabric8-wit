package gormtestsupport

import (
	"os"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// NewDBTestSuite instanciate a new DBTestSuite
func NewDBTestSuite(configFilePath string) DBTestSuite {
	return DBTestSuite{configFile: configFilePath}
}

// DBTestSuite is a base for tests using a gorm db
type DBTestSuite struct {
	suite.Suite
	configFile           string
	Configuration        *config.ConfigurationData
	DB                   *gorm.DB
	clean                func()
	restoreGormCallbacks func()
	Ctx                  context.Context
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
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
	s.Ctx = migration.NewMigrationContext(context.Background())
	s.populateDBTestSuite(s.Ctx)
}

// SetupTest implements suite.SetupTest
func (s *DBTestSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.DisableGormCallbacks()
}

// TearDownTest implements suite.TearDownTest
func (s *DBTestSuite) TearDownTest() {
	s.clean()
	s.RestoreGormCallbacks()
}

// RestoreGormCallbacks restores gorm callbacks. For more information see
// DisableGormCallbacks(). You can call this function multiple times but it will
// only restore the callbacks once. For now this function is NOT thread-safe, so
// don't use it in parallel tests!
func (s *DBTestSuite) RestoreGormCallbacks() {
	if s.restoreGormCallbacks != nil {
		s.restoreGormCallbacks()
	}
	s.restoreGormCallbacks = nil
}

// populateDBTestSuite populates the DB with common values
func (s *DBTestSuite) populateDBTestSuite(ctx context.Context) {
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
func (s *DBTestSuite) TearDownSuite() {
	s.DB.Close()
}

// DisableGormCallbacks will turn off gorm's automatic setting of `created_at`
// and `updated_at` columns. Call this function and make sure to call
// RestoreGormCallbacks() when you want to restore the normal callbacks (e.g.
// after each test).
func (s *DBTestSuite) DisableGormCallbacks() {
	// To avoid disabling callbacks entirely, first see if there's something to
	// restore already.
	s.RestoreGormCallbacks()

	gormCallbackName := "gorm:update_time_stamp"
	// remember old callbacks
	oldCreateCallback := s.DB.Callback().Create().Get(gormCallbackName)
	oldUpdateCallback := s.DB.Callback().Update().Get(gormCallbackName)
	// remove current callbacks
	s.DB.Callback().Create().Remove(gormCallbackName)
	s.DB.Callback().Update().Remove(gormCallbackName)
	// return a function to restore old callbacks
	s.restoreGormCallbacks = func() {
		s.DB.Callback().Create().Register(gormCallbackName, oldCreateCallback)
		s.DB.Callback().Update().Register(gormCallbackName, oldUpdateCallback)
	}
}
