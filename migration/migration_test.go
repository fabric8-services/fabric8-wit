package migration

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

// Test with
// docker-compose -f ./.make/docker-compose.integration-test.yaml stop -t 0 && docker-compose -f ./.make/docker-compose.integration-test.yaml rm -v --all -f && make integration-test-env-prepare && export ALMIGHTY_POSTGRES_HOST=$(docker inspect --format '{{ .NetworkSettings.IPAddress }}' make_postgres_integration_test_1); export ALMIGHTY_RESOURCE_DATABASE=1; go test github.com/almighty/almighty-core/migration -v

func TestConcurrentMigrations(t *testing.T) {
	resource.Require(t, resource.Database)

	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			db, err := sql.Open("postgres", configuration.GetPostgresConfigString())
			if err != nil {
				t.Fatalf("Cannot connect to DB: %s\n", err)
			}
			err = Migrate(db)
			assert.Nil(t, err)
		}()

	}
	wg.Wait()
}
