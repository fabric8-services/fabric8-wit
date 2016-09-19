package main

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/almighty/almighty-core/resource"
)

func TestCreateTracker(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
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
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
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

// This test ensures that List does not return NIL items.
// refer : https://github.com/almighty/almighty-core/issues/191
func TestTrackerListItemsNotNil(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
	_, item1 := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, item2 := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, list := test.ListTrackerOK(t, nil, nil, &controller, nil, nil)

	for _, tracker := range list {
		if tracker == nil {
			t.Error("Returned Tracker found nil")
		}
	}
	test.DeleteTrackerOK(t, nil, nil, &controller, item1.ID)
	test.DeleteTrackerOK(t, nil, nil, &controller, item2.ID)
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/almighty/almighty-core/issues/189
func TestCreateTrackerValidId(t *testing.T) {
	resource.Require(t, resource.Database)
	ts := remoteworkitem.NewGormTransactionSupport(db)
	repo := remoteworkitem.NewTrackerRepository(ts)
	controller := TrackerController{ts: ts, tRepository: repo, scheduler: rwiScheduler}
	payload := app.CreateTrackerPayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
	_, tracker := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, created := test.ShowTrackerOK(t, nil, nil, &controller, tracker.ID)
	if created != nil && created.ID != tracker.ID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", tracker.ID, " Expected, ", created.ID)
	}
	test.DeleteTrackerOK(t, nil, nil, &controller, tracker.ID)
}
