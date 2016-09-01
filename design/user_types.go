package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

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
// not be used since it mand, wi.IDated the presence of the ID in the payload
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
