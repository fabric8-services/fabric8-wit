package main

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/resource"
)

func TestCreateTrackerQuery(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	t.Log(result.ID)
	tqts := remoteworkitem.NewGormTransactionSupport(db)
	tqrepo := remoteworkitem.NewTrackerQueryRepository(tqts)
	tqController := TrackerqueryController{ts: tqts, tqRepository: tqrepo, scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerqueryPayload{

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
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	tqrepo := remoteworkitem.NewTrackerQueryRepository(ts)
	tqController := TrackerqueryController{ts: ts, tqRepository: tqrepo, scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerqueryPayload{

		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: result.ID,
	}
	fmt.Printf("tq payload %#v", tqpayload)
	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, &tqController, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, &tqController, tqresult.ID)

	fmt.Println("-----------tqr-------", tqr.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.ID)
	}
	if tqr.ID != tqresult.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.ID, tqr.ID)
	}
}

func TestUpdateTrackerQuery(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	tqrepo := remoteworkitem.NewTrackerQueryRepository(ts)
	tqController := TrackerqueryController{ts: ts, tqRepository: tqrepo, scheduler: rwiScheduler}
	tqpayload := app.CreateTrackerqueryPayload{

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

	payload2 := app.UpdateTrackerqueryPayload{
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
