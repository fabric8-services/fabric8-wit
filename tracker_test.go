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
	defer gormsupport.DeleteCreatedEntities(DB)()
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
	if *tr.Data.ID != resultID {
		t.Errorf("Id should be %s, but is %s", resultID, *tr.Data.ID)
	}

	payload2 := app.UpdateTrackerPayload{
		Data: &app.TrackerUpdateData{
			Attributes: &app.TrackerAttributesToUpdate{
				URL:  &tr.Data.Attributes.URL,
				Type: &tr.Data.Attributes.Type,
			},
			ID:   *tr.Data.ID,
			Type: APIStringTypeTracker,
		},
	}
	_, updated := test.UpdateTrackerOK(t, nil, nil, &controller, *tr.Data.ID, &payload2)
	if *updated.Data.ID != resultID {
		t.Errorf("Id has changed from %s to %s", resultID, *updated.Data.ID)
	}
	resultURL := result.Data.Attributes.URL
	if updated.Data.Attributes.URL != resultURL {
		t.Errorf("URL has changed from %s to %s", resultURL, updated.Data.Attributes.URL)
	}
	resultType := result.Data.Attributes.Type
	if updated.Data.Attributes.Type != resultType {
		t.Errorf("Type has changed has from %s to %s", resultType, updated.Data.Attributes.Type)
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
	_, item1 := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, item2 := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, list := test.ListTrackerOK(t, nil, nil, &controller, nil, nil)

	for _, tracker := range list.Data {
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
	if created != nil && *created.Data.ID != trackerID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", trackerID, " Expected, ", *created.Data.ID)
	}
}
