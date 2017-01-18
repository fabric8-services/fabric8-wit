package remoteworkitem

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/workitem"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c {
		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()

		// Make sure the database is populated with the correct types (e.g. bug etc.)
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(context.Background(), tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			panic(err.Error())
		}
	}
	os.Exit(m.Run())
}

func TestNewScheduler(t *testing.T) {
	resource.Require(t, resource.Database)

	s := NewScheduler(db)
	if s.db != db {
		t.Error("DB not set as an attribute")
	}
	s.Stop()
}

func TestLookupProvider(t *testing.T) {
	resource.Require(t, resource.Database)
	ts1 := trackerSchedule{TrackerType: ProviderGithub}
	tp1 := lookupProvider(ts1)
	require.NotNil(t, tp1)

	ts2 := trackerSchedule{TrackerType: ProviderJira}
	tp2 := lookupProvider(ts2)
	require.NotNil(t, tp2)

	ts3 := trackerSchedule{TrackerType: "unknown"}
	tp3 := lookupProvider(ts3)
	require.Nil(t, tp3)
}
