package migration

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

func TestConcurrentMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	configuration, err := config.Get()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			db, err := sql.Open("postgres", configuration.GetPostgresConfigString())
			if err != nil {
				require.NoError(t, err, "cannot connect to DB")
			}
			err = Migrate(db, configuration.GetPostgresDatabase())
			require.NoError(t, err, "%+v", err)
		}()

	}
	wg.Wait()
}

func TestMigrationWithExpectedVersion(t *testing.T) {
	resource.Require(t, resource.Database)

	configuration, err := config.Get()
	if err != nil {
		panic(fmt.Errorf("failed to setup the configuration: %s", err.Error()))
	}

	db, err := sql.Open("postgres", configuration.GetPostgresConfigString())
	if err != nil {
		require.NoError(t, err, "cannot connect to DB")
	}

	t.Run("ok", func(t *testing.T) {
		err := Migrate(db, configuration.GetPostgresDatabase())
		require.NoError(t, err, "%+v", err)
	})
	t.Run("fail due to newer db version", func(t *testing.T) {
		// given
		artificialDBVersion := len(GetMigrations()) + 1 + 100
		_, err = db.Exec("INSERT INTO version(version) VALUES($1) ON CONFLICT DO NOTHING", artificialDBVersion)
		require.NoError(t, err, "failed to insert artificial version: %d", artificialDBVersion)
		defer func() {
			_, err = db.Exec("DELETE FROM version WHERE version = $1", artificialDBVersion)
			require.NoError(t, err, "failed to remove any potentially existing artificial version: %d", artificialDBVersion)
		}()
		// when
		err = Migrate(db, configuration.GetPostgresDatabase())
		// then
		require.Error(t, err, "the migration should have failed because the db was artificially moved by 100 revisions to %d and core expects version %d: %+v", artificialDBVersion, len(GetMigrations())+1, err)
	})
}
