package models_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	var err error

	if err = configuration.Setup("../config.yaml"); err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}

	if _, c := os.LookupEnv(resource.Database); c != false {
		db, err = gorm.Open("postgres",
			fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
				configuration.GetPostgresHost(),
				configuration.GetPostgresPort(),
				configuration.GetPostgresUser(),
				configuration.GetPostgresPassword(),
				configuration.GetPostgresSSLMode(),
			))
		if err != nil {
			panic("Failed to connect database: " + err.Error())
		}
		defer db.Close()
	}
	os.Exit(m.Run())
}
