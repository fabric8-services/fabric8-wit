package main

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/resource"
)

var scheduler *remoteworkitem.Scheduler

func TestCreateTracker(t *testing.T) {
	resource.Require(t, resource.Database)

	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: scheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}

	_, created := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func TestGetTracker(t *testing.T) {
	resource.Require(t, resource.Database)

	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: scheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}

	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	test.ShowTrackerOK(t, nil, nil, &controller, result.ID)
	_, tr := test.ShowTrackerOK(t, nil, nil, &controller, result.ID)
	if tr == nil {
		t.Fatalf("Tracker '%s' not present", result.ID)
	}

	if tr.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, tr.ID)
	}

	payload2 := app.UpdateTrackerPayload{
		URL:  tr.URL,
		Type: tr.Type,
	}
	_, updated := test.UpdateTrackerOK(t, nil, nil, &controller, tr.ID, &payload2)
	if updated.ID != result.ID {
		t.Errorf("Id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.URL != result.URL {
		t.Errorf("URL has changed from %s to %s", result.URL, updated.URL)
	}
	if updated.Type != result.Type {
		t.Errorf("Type has changed has from %s to %s", result.Type, updated.Type)
	}

	test.DeleteTrackerOK(t, nil, nil, &controller, result.ID)
}
