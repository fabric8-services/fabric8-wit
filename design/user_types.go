package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// CreateWorkItemPayload defines the structure of work item payload
var CreateWorkItemPayload = Type("CreateWorkItemPayload", func() {
	Attribute("type", String, "The type of the newly created work item")
	Attribute("name", String, "User Readable Name of this item")
	Attribute("fields", HashOf(String, Any), "The field values, must conform to the type")
	Required("type", "name", "fields")
})

// UpdateWorkItemPayload has been added because the design.WorkItem could
// not be used since it mandated the presence of the ID in the payload
// which ideally should be optional. The ID should be passed on to REST URL.
var UpdateWorkItemPayload = Type("UpdateWorkItemPayload", func() {
	Attribute("type", String, "The type of the newly created work item")
	Attribute("name", String, "User Readable Name of this item")
	Attribute("fields", HashOf(String, Any), "The field values, must conform to the type")
	Attribute("version", Integer, "Version for optimistic concurrency control")
	Required("type", "name", "fields", "version")
})

// CreateTrackerAlternatePayload defines the structure of tracker payload for create
var CreateTrackerAlternatePayload = Type("CreateTrackerAlternatePayload", func() {
	Attribute("url", String, "URL of the tracker")
	Attribute("credentials", String, "Credential to access the tracker")
	Attribute("type", String, "Type of the tracker")
	Required("url", "credentials", "type")
})

// UpdateTrackerAlternatePayload defines the structure of tracker payload for update
var UpdateTrackerAlternatePayload = Type("UpdateTrackerAlternatePayload", func() {
	Attribute("url", String, "URL of the tracker")
	Attribute("credentials", String, "Credential to access the tracker")
	Attribute("type", String, "Type of the tracker")
	Required("url", "credentials", "type")
})
