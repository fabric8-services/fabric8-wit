package gormtestsupport

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"testing"

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
	configFile     string
	Configuration  *config.ConfigurationData
	DB             *gorm.DB
	waitGroups     map[*testing.T]*sync.WaitGroup
	waitGroupsLock sync.RWMutex
}

// SetupSuite implements suite.SetupAllSuite
func (s *DBTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Database)
	fmt.Println("SetupSuite(): lock")
	s.waitGroupsLock.Lock()
	fmt.Println("SetupSuite(): lock DONE")
	defer func() {
		fmt.Println("SetupSuite(): unlock")
		s.waitGroupsLock.Unlock()
		fmt.Println("SetupSuite(): unlock DONE")
	}()
	s.waitGroups = make(map[*testing.T]*sync.WaitGroup)
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

// WaitGroup returns the WaitGroup associated with the current suite test. It
// can be called from subtests as well. If no wait group is associated with a
// test yet, one will be created on the fly.
//
// In the TearDownTest of each suite make s.WaitGroup().Wait() the first call
// before cleaning up after each test:
//
// 	func (s *yourSuite) TearDownTest() {
//		s.WaitGroup().Wait()
//		/* cleanup resources only after waiting */
//	}
//
// To Add a parallel subtest to the current suite's test, do this:
//
//	s.RunParallel("my subtest", func(t *testing.T){
//		/*just do your normal testing here*/
//	})
func (s *DBTestSuite) waitGroup() *sync.WaitGroup {
	fmt.Println("waitGroup(): lock")
	s.waitGroupsLock.Lock()
	fmt.Println("waitGroup(): lock DONE")
	defer func() {
		fmt.Println("waitGroup(): unlock")
		s.waitGroupsLock.Unlock()
		fmt.Println("waitGroup(): unlock DONE")
	}()

	wg, ok := s.waitGroups[s.T()]
	if ok {
		return wg
	}
	// No wait group available for this test yet
	wg = &sync.WaitGroup{}
	s.waitGroups[s.T()] = wg
	return wg
}

var allowParallelSubTests = flag.Bool("allowParallelSubTests", true, "when set, parallel tests are enabled")

// RunParallel does all the setup for running the function t as a parallel
// subtest that takes care of setting up synchronization primitives. See the
// description of waitGroup as well to find out about freeing of resources.
func (s *DBTestSuite) RunParallel(name string, f func(subtest *testing.T)) bool {
	var wg *sync.WaitGroup
	if *allowParallelSubTests {
		wg = s.waitGroup()
		fmt.Println("RunParallel(): Add")
		wg.Add(1)
		fmt.Println("RunParallel(): Add DONE")
	}
	return s.T().Run(name, func(t *testing.T) {
		if *allowParallelSubTests {
			// Make the outer suite's test wait for this subtest
			defer func() {
				fmt.Println("RunParallel(): Done")
				wg.Done()
				fmt.Println("RunParallel(): Done DONE")
			}()
			fmt.Println("RunParallel(): signaling parallelism")
			t.Parallel()
			fmt.Println("RunParallel(): signaling parallelism DONE")
		}
		fmt.Println("RunParallel(): executing user function")
		f(t)
		fmt.Println("RunParallel(): executing user function DONE")
	})
}

// WaitForTests waits for parallel subtests to finish.
func (s *DBTestSuite) WaitForParallelTests() {
	if *allowParallelSubTests {

		fmt.Println("WaitForParallelTests(): Lock")
		s.waitGroupsLock.RLock()
		fmt.Println("WaitForParallelTests(): Lock DONE")
		defer func() {
			fmt.Println("WaitForParallelTests(): RUnlock")
			s.waitGroupsLock.RUnlock()
			fmt.Println("WaitForParallelTests(): RUnlock DONE")
		}()
		wg, ok := s.waitGroups[s.T()]
		if ok {
			fmt.Println("WaitForParallelTests(): Wait")
			wg.Wait()
			fmt.Println("WaitForParallelTests(): Wait DONE")
		}
		//s.waitGroup().Wait()
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
