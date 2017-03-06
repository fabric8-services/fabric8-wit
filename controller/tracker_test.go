package controller

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"

	"github.com/almighty/almighty-core/gormtestsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
)

var trTestConfiguration *config.ConfigurationData

func init() {
	var err error
	trTestConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestCreateTracker(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := app.CreateTrackerAlternatePayload{
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
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := app.CreateTrackerAlternatePayload{
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

	payload2 := app.UpdateTrackerAlternatePayload{
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

}

// This test ensures that List does not return NIL items.
// refer : https://github.com/almighty/almighty-core/issues/191
func TestTrackerListItemsNotNil(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
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
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
	_, tracker := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

	_, created := test.ShowTrackerOK(t, nil, nil, &controller, tracker.ID)
	if created != nil && created.ID != tracker.ID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", tracker.ID, " Expected, ", created.ID)
	}
}
