package remoteworkitem

import (
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	if _, c := os.LookupEnv(resource.Database); c == false {
		fmt.Printf(resource.StSkipReasonNotSet+"\n", resource.Database)
		return
	}

	dbhost := os.Getenv("ALMIGHTY_DB_HOST")
	if "" == dbhost {
		panic("The environment variable ALMIGHTY_DB_HOST is not specified or empty.")
	}
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", dbhost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()
	// Migrate the schema
	db.AutoMigrate(
		&Tracker{},
		&TrackerQuery{},
		&TrackerItem{})
	q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker") REFERENCES "trackers" ON DELETE CASCADE`
	db.Exec(q)

	os.Exit(m.Run())
}

func TestNewScheduler(t *testing.T) {
	resource.Require(t, resource.Database)

	s := NewScheduler(db)
	if s.db != db {
		t.Error("DB not set as an attribute")
	}
}
