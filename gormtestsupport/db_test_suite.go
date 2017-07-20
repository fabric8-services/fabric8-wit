package gormtestsupport

import (
	"context"
	"flag"
	"os"
	"sync/atomic"
	"testing"
	"unsafe"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
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
	configFile    string
	Configuration *config.ConfigurationData
	DB            *gorm.DB
	numSubTests   map[*testing.T]*int32
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	s.numSubTests = make(map[*testing.T]*int32)
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

var allowParallelSubTests = flag.Bool("allowParallelSubTests", true, "when set, parallel tests are enabled")

// RunParallel does all the setup for running the function t as a parallel
// subtest that takes care of setting up synchronization primitives. See the
// description of waitGroup as well to find out about freeing of resources.
func (s *DBTestSuite) RunParallel(name string, f func(subtest *testing.T)) bool {
	if *allowParallelSubTests {
		var newInt64 int32
		unsafePtr := unsafe.Pointer(s.numSubTests[s.T()])
		atomic.CompareAndSwapPointer(&unsafePtr, unsafe.Pointer(nil), unsafe.Pointer(&newInt64))
		atomic.AddInt32(s.numSubTests[s.T()], 1)
	}
	return s.T().Run(name, func(t *testing.T) {
		if *allowParallelSubTests {
			t.Parallel()
		}
		f(t)
		if *allowParallelSubTests {
			atomic.AddInt32(s.numSubTests[s.T()], -1)
		}
	})
}

// WaitForTests waits for parallel subtests to finish.
func (s *DBTestSuite) WaitForParallelTests() {
	if *allowParallelSubTests {
		numSubTestsPtr := s.numSubTests[s.T()]
		for numSubTestsPtr != nil && *numSubTestsPtr > 0 {
		}
	}
}

// PopulateDBTestSuite populates the DB with common values
func (s *DBTestSuite) PopulateDBTestSuite(ctx context.Context) {
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
// and `updated_at` columns. Call this function and make sure to `defer` the
// returned function.
//
//    resetFn := DisableGormCallbacks()
//    defer resetFn()
func (s *DBTestSuite) DisableGormCallbacks() func() {
	gormCallbackName := "gorm:update_time_stamp"
	// remember old callbacks
	oldCreateCallback := s.DB.Callback().Create().Get(gormCallbackName)
	oldUpdateCallback := s.DB.Callback().Update().Get(gormCallbackName)
	// remove current callbacks
	s.DB.Callback().Create().Remove(gormCallbackName)
	s.DB.Callback().Update().Remove(gormCallbackName)
	// return a function to restore old callbacks
	return func() {
		s.DB.Callback().Create().Register(gormCallbackName, oldCreateCallback)
		s.DB.Callback().Update().Register(gormCallbackName, oldUpdateCallback)
	}
}
