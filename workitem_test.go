package main

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/transaction"

	"github.com/almighty/almighty-core/resource"
	"github.com/jinzhu/gorm"
)

var db *gorm.DB
var rwiScheduler *remoteworkitem.Scheduler

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
	ts := models.NewGormTransactionSupport(db)
	witRepo := models.NewWorkItemTypeRepository(ts)

	if err := transaction.Do(ts, func() error {
		return migration.Perform(context.Background(), ts.TX(), witRepo)
	}); err != nil {
		panic(err.Error())
	}
	// RemoteWorkItemScheduler now available for all other test cases
	rwiScheduler = remoteworkitem.NewScheduler(db)
	os.Exit(m.Run())
}

func TestGetWorkItem(t *testing.T) {
	resource.Require(t, resource.Database)

	ts := models.NewGormTransactionSupport(db)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "Test WI",
			"system.creator": "aslak",
			"system.state":   "closed"},
	}

	_, result := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)

	_, wi := test.ShowWorkitemOK(t, nil, nil, &controller, result.ID)

	if wi == nil {
		t.Fatalf("Work Item '%s' not present", result.ID)
	}

	if wi.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, wi.ID)
	}

	wi.Fields["system.creator"] = "thomas"
	payload2 := app.UpdateWorkitemPayload{
		Type:    wi.Type,
		Version: wi.Version,
		Fields:  wi.Fields,
	}
	_, updated := test.UpdateWorkitemOK(t, nil, nil, &controller, wi.ID, &payload2)
	if updated.Version != result.Version+1 {
		t.Errorf("expected version %d, but got %d", result.Version+1, updated.Version)
	}
	if updated.ID != result.ID {
		t.Errorf("id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.Fields["system.creator"] != "thomas" {
		t.Errorf("expected creator %s, but got %s", "thomas", updated.Fields["system.creator"])
	}

	test.DeleteWorkitemOK(t, nil, nil, &controller, result.ID)
}

func TestCreateWI(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "Test WI",
			"system.creator": "tmaeder",
			"system.state":   "new",
		},
	}

	_, created := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func TestListByFields(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := models.NewGormTransactionSupport(db)
	wir := models.NewWorkItemTypeRepository(ts)
	repo := models.NewWorkItemRepository(ts, wir)
	controller := WorkitemController{ts: ts, wiRepository: repo}
	payload := app.CreateWorkitemPayload{
		Type: "system.bug",
		Fields: map[string]interface{}{
			"system.title":   "run integration test",
			"system.creator": "aslak",
			"system.state":   "closed"},
	}

	_, wi := test.CreateWorkitemCreated(t, nil, nil, &controller, &payload)

	filter := "{\"system.title\":\"run integration test\"}"
	page := "0,1"
	_, result := test.ListWorkitemOK(t, nil, nil, &controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d", 1, len(result))
	}

	filter = "{\"system.creator\":\"aslak\"}"
	_, result = test.ListWorkitemOK(t, nil, nil, &controller, &filter, &page)

	if result == nil {
		t.Errorf("nil result")
	}

	if len(result) != 1 {
		t.Errorf("unexpected length, should be %d but is %d ", 1, len(result))
	}

	test.DeleteWorkitemOK(t, nil, nil, &controller, wi.ID)
}
