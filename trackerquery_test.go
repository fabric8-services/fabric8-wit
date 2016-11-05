package main

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/resource"
)

func TestCreateTrackerQuery(t *testing.T) {
	resource.Require(t, resource.Database)
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	t.Log(result.ID)
	tqController := TrackerqueryController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerQueryAlternatePayload{

		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: result.ID,
	}

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, &tqController, &tqpayload)
	t.Log(tqresult)
	if tqresult.ID == "" {
		t.Error("no id")
	}
}

func TestGetTrackerQuery(t *testing.T) {
	resource.Require(t, resource.Database)
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	tqController := TrackerqueryController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerQueryAlternatePayload{

		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: result.ID,
	}
	fmt.Printf("tq payload %#v", tqpayload)
	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, &tqController, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.ID)
	}
	if tqr.ID != tqresult.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.ID, tqr.ID)
	}
}

func TestUpdateTrackerQuery(t *testing.T) {
	resource.Require(t, resource.Database)
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	tqController := TrackerqueryController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerQueryAlternatePayload{

		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: result.ID,
	}

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, &tqController, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.ID)
	}
	if tqr.ID != tqresult.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.ID, tqr.ID)
	}

	payload2 := app.UpdateTrackerQueryAlternatePayload{
		Query:     tqr.Query,
		Schedule:  tqr.Schedule,
		TrackerID: result.ID,
	}
	_, updated := test.UpdateTrackerqueryOK(t, nil, nil, &tqController, tqr.ID, &payload2)

	if updated.ID != tqresult.ID {
		t.Errorf("Id has changed from %s to %s", tqresult.ID, updated.ID)
	}
	if updated.Query != tqresult.Query {
		t.Errorf("Query has changed from %s to %s", tqresult.Query, updated.Query)
	}
	if updated.Schedule != tqresult.Schedule {
		t.Errorf("Type has changed has from %s to %s", tqresult.Schedule, updated.Schedule)
	}
}
