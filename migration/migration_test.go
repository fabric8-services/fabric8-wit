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

		go func() {
			defer wg.Done()
			db, err := sql.Open("postgres", configuration.GetPostgresConfigString())
			if err != nil {
				t.Fatal("Cannot connect to DB", err)
			}
			//		db.DB().SetMaxIdleConns(0)
			err = Migrate1(db)
			assert.Nil(t, err)
		}()

	}
	wg.Wait()
}
