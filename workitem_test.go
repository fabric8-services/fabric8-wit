package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/migration"

	"github.com/jinzhu/gorm"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	var dbhost *string = flag.String("dbhost", "", "-dbhost <hostname>")
	flag.Parse()
	if "" == *dbhost {
		flag.Usage()
		os.Exit(-1)
	}
	var err error
	db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=postgres password=mysecretpassword sslmode=disable", *dbhost))
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	defer db.Close()
	// Migrate the schema
	migration.Perform(db)
	m.Run()
}

func TestGetWorkItem(t *testing.T) {
	controller := WorkitemController{db: db}
	payload := app.CreateWorkitemPayload{
		Name:   "foobar",
		Type: "1",
		Fields: map[string]interface{}{
			"system.owner": "aslak",
			"system.state": "done"},
	}

	_, result := test.CreateWorkitemOK(t, nil, nil, &controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, &controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%d' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %d, but is %s", result.ID, wi.ID)
	}

	wi.Fields["system.owner"] = "thomas"
	payload2 := app.UpdateWorkitemPayload{
		ID:      wi.ID,
		Name:    wi.Name,
		Type:    wi.Type,
		Version: wi.Version,
		Fields:  wi.Fields,
	}
	_, updated := test.UpdateWorkitemOK(t, nil, nil, &controller, &payload2)
	if updated.Version != result.Version+1 {
		t.Errorf("expected version %d, but got %d", result.Version+1, updated.Version)
	}
	if updated.ID != result.ID {
		t.Errorf("id has changed from %d to %d", result.ID, updated.ID)
	}
	if updated.Fields["system.owner"] != "thomas" {
		t.Errorf("expected owner %s, but got %s", "thomas", updated.Fields["system.owner"])
	}

	test.DeleteWorkitemOK(t, nil, nil, &controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	controller := WorkitemController{db: db}
	payload := app.CreateWorkitemPayload{
		Name:   "some name",
		Type: "1",
		Fields: map[string]interface{}{
			"system.owner": "tmaeder",
			"system.state": "open",
		},
	}

	_, created := test.CreateWorkitemOK(t, nil, nil, &controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}
