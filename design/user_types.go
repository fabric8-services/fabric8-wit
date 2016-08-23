package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// CreateWorkItemPayload defines the structure of work item payload
var CreateWorkItemPayload = Type("CreateWorkItemPayload", func() {
	Attribute("type", String, "The type of the newly created work item", func() {
		Example("1")
	})
	Attribute("fields", HashOf(String, Any), "The field values, must conform to the type", func() {
		Example(map[string]interface{}{"system.owner": "user-ref", "system.state": "open"})
	})
	Required("type",  "fields")
})

// UpdateWorkItemPayload has been added because the design.WorkItem could
// not be used since it mandated the presence of the ID in the payload
// which ideally should be optional. The ID should be passed on to REST URL.
var UpdateWorkItemPayload = Type("UpdateWorkItemPayload", func() {
	Attribute("type", String, "The type of the newly created work item", func() {
		Example("1")
	})
	Attribute("fields", HashOf(String, Any), "The field values, must conform to the type", func() {
		Example(map[string]interface{}{"system.owner": "user-ref", "system.state": "open"})
	})
	Attribute("version", Integer, "Version for optimistic concurrency control", func() {
		Example(0)
	})
	Required("type", "fields", "version")
})

var CreateWorkItemTypePayload = Type("CreateWorkItemTypePayload", func() {
	Attribute("name", String, "Readable name of the type like Task, Issue, Bug, Epic etc.")
	Attribute("fields", HashOf(String, fieldDefinition), "Type fields those must be followed by respective Work Items.")
	Attribute("extendedTypeName", String, "If newly created type extends any existing type")
	Required("name", "fields")
})

// CreateTrackerAlternatePayload defines the structure of tracker payload for create
var CreateTrackerAlternatePayload = Type("CreateTrackerAlternatePayload", func() {
	Attribute("url", String, "URL of the tracker")
	Attribute("type", String, "Type of the tracker")
	Required("url", "type")
})

// UpdateTrackerAlternatePayload defines the structure of tracker payload for update
var UpdateTrackerAlternatePayload = Type("UpdateTrackerAlternatePayload", func() {
	Attribute("url", String, "URL of the tracker")
	Attribute("type", String, "Type of the tracker")
	Required("url", "type")
})

// CreateTrackerQueryAlternatePayload defines the structure of tracker query payload for create
var CreateTrackerQueryAlternatePayload = Type("CreateTrackerQueryAlternatePayload", func() {
	Attribute("query", String, "Search query")
	Attribute("schedule", String, "Schedule for fetch and import")
	Required("query", "schedule")
})

// UpdateTrackerQueryAlternatePayload defines the structure of tracker query payload for update
var UpdateTrackerQueryAlternatePayload = Type("UpdateTrackerQueryAlternatePayload", func() {
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Attribute("query", String, "Search query")
	Attribute("schedule", String, "Schedule for fetch and import")
	Required("query", "schedule", "version")
})
