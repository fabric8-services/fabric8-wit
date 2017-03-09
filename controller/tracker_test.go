package controller

import (
	"fmt"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	config "github.com/almighty/almighty-core/configuration"
	"github.com/almighty/almighty-core/gormapplication"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/resource"
	"github.com/stretchr/testify/assert"
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
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})

	_, result := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	resultID := *result.Data.ID
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
	assert.Equal(t, *updated.Data.ID, resultID)
	assert.Equal(t, result.Data.Attributes.URL, result.Data.Attributes.URL)
	assert.Equal(t, updated.Data.Attributes.Type, result.Data.Attributes.Type)
}

// This test ensures that List does not return NIL items.
// refer : https://github.com/almighty/almighty-core/issues/191
func TestTrackerListItemsNotNil(t *testing.T) {
	resource.Require(t, resource.Database)
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})
	test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	test.CreateTrackerCreated(t, nil, nil, &controller, &payload)

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
	defer cleaner.DeleteCreatedEntities(DB)()
	controller := TrackerController{Controller: nil, db: gormapplication.NewGormDB(DB), scheduler: RwiScheduler, configuration: trTestConfiguration}
	payload := getTrackerPayload(trackerAttr{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	})
	_, tracker := test.CreateTrackerCreated(t, nil, nil, &controller, &payload)
	trackerID := *tracker.Data.ID
	_, created := test.ShowTrackerOK(t, nil, nil, &controller, trackerID)
	assert.NotNil(t, created)
	assert.Equal(t, *created.Data.ID, trackerID)
}
