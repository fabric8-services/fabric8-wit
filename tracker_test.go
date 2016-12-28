package main

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/resource"
)

type trackerAttr struct {
	Type string
	URL  string
}

func getTrackerPayload(attr trackerAttr) app.CreateTrackerPayload {
	return app.CreateTrackerPayload{
		Data: &app.TrackerData{
			Type: APIStringTypeTracker,
			Attributes: &app.TrackerAttributes{
				Type: attr.Type,
				URL:  attr.URL,
			},
		},
	}
}

func TestCreateTracker(t *testing.T) {
	resource.Require(t, resource.Database)
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler}

	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})
	_, created := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	if *created.Data.ID == "" {
		t.Error("no id")
	}
}

func TestGetTracker(t *testing.T) {
	resource.Require(t, resource.Database)
	defer gormsupport.DeleteCreatedEntities(DB)()

	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})

	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	resultID := *result.Data.ID
	test.ShowTrackerOK(t, nil, nil, &controller, resultID)
	_, tr := test.ShowTrackerOK(t, nil, nil, &controller, resultID)
	if tr == nil {
		t.Fatalf("Tracker '%s' not present", resultID)
	}
	if tr.ID != resultID {
		t.Errorf("Id should be %s, but is %s", resultID, tr.ID)
	}

	payload2 := app.UpdateTrackerAlternatePayload{
		URL:  tr.URL,
		Type: tr.Type,
	}
	_, updated := test.UpdateTrackerOK(t, nil, nil, &controller, tr.ID, &payload2)
	if updated.ID != resultID {
		t.Errorf("Id has changed from %s to %s", resultID, updated.ID)
	}
	resultURL := result.Data.Attributes.URL
	if updated.URL != resultURL {
		t.Errorf("URL has changed from %s to %s", resultURL, updated.URL)
	}
	resultType := result.Data.Attributes.Type
	if updated.Type != resultType {
		t.Errorf("Type has changed has from %s to %s", resultType, updated.Type)
	}
}

// This test ensures that List does not return NIL items.
// refer : https://github.com/almighty/almighty-core/issues/191
func TestTrackerListItemsNotNil(t *testing.T) {
	resource.Require(t, resource.Database)
	defer gormsupport.DeleteCreatedEntities(DB)()

	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})
	test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, list := test.ListTrackerOK(t, nil, nil, &controller, nil, nil)

	for _, tracker := range list {
		if tracker == nil {
			t.Error("Returned Tracker found nil")
		}
	}
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/almighty/almighty-core/issues/189
func TestCreateTrackerValidId(t *testing.T) {
	resource.Require(t, resource.Database)
	defer gormsupport.DeleteCreatedEntities(DB)()

	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})
	_, tracker := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	trackerID := *tracker.Data.ID
	_, created := test.ShowTrackerOK(t, nil, nil, &controller, trackerID)
	if created != nil && created.ID != trackerID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", trackerID, " Expected, ", created.ID)
	}
}
