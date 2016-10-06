package migration

import (
	"fmt"
	"sync"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentMigrations(t *testing.T) {
	resource.Require(t, resource.MigrationTest)

	var err error
	if err = configuration.Setup(""); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		db, err := gorm.Open("postgres",
			fmt.Sprintf("host=%s port=%d user=%s password=%s DB.name=%s sslmode=%s",
				configuration.GetPostgresHost(),
				configuration.GetPostgresPort(),
				configuration.GetPostgresUser(),
				configuration.GetPostgresPassword(),
				configuration.GetPostgresDatabase(),
				configuration.GetPostgresSSLMode(),
			))
		if err != nil {
			t.Fatal("Cannot connect to DB", err)
		}

		go func(db *gorm.DB) {
			defer wg.Done()
			err = Migrate(db)
			assert.Nil(t, err)
		}(db)

	}
	wg.Wait()
}
