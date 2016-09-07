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
	ec := m.Run()
	os.Exit(ec)
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
	ts1 := trackerSchedule{TrackerType: ProviderGithub}
	tp1 := LookupProvider(ts1)
	if tp1 == nil {
		t.Error("nil provider")
	}
	ts2 := trackerSchedule{TrackerType: ProviderJira}
	tp2 := LookupProvider(ts2)
	if tp2 == nil {
		t.Error("nil provider")
	}
	ts3 := trackerSchedule{TrackerType: "unknown"}
	tp3 := LookupProvider(ts3)
	if tp3 != nil {
		t.Error("non-nil provider")
	}
}
