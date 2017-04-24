package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// CreateWorkItemPayload defines the structure of work item payload
var CreateWorkItemPayload = a.Type("CreateWorkItemPayload", func() {
	a.Attribute("type", d.UUID, "ID of the work item type of the newly created work item")
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, must conform to the type", func() {
		a.Example(map[string]interface{}{"system.creator": "user-ref", "system.state": "new", "system.title": "Example story"})
		a.MinLength(1)
	})
	a.Required("type", "fields")
})

// UpdateWorkItemPayload has been added because the design.WorkItem could
// not be used since it mandated the presence of the ID in the payload
// which ideally should be optional. The ID should be passed on to REST URL.
var UpdateWorkItemPayload = a.Type("UpdateWorkItemPayload", func() {
	a.Attribute("type", d.UUID, "ID of the work item type")
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, must conform to the type", func() {
		a.Example(map[string]interface{}{"system.creator": "user-ref", "system.state": "new", "system.title": "Example story"})
		a.MinLength(1)
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control", func() {
		a.Example(0)
	})
	a.Attribute("executionorder", d.Number, "The order of execution of workitem", func() {
		a.Example(1000)
	})
	a.Attribute("relationships", workItemRelationships)

	a.Required("type", "fields", "version", "executionorder")
})

// CreateTrackerAlternatePayload defines the structure of tracker payload for create
var CreateTrackerAlternatePayload = a.Type("CreateTrackerAlternatePayload", func() {
	a.Attribute("url", d.String, "URL of the tracker", func() {
		a.Example("https://api.github.com/")
		a.MinLength(1)
	})
	a.Attribute("type", d.String, "Type of the tracker", func() {
		a.Example("github")
		a.Pattern("^[\\p{L}]+$")
		a.MinLength(1)
	})
	a.Required("url", "type")
})

// UpdateTrackerAlternatePayload defines the structure of tracker payload for update
var UpdateTrackerAlternatePayload = a.Type("UpdateTrackerAlternatePayload", func() {
	a.Attribute("url", d.String, "URL of the tracker", func() {
		a.Example("https://api.github.com/")
		a.MinLength(1)
	})
	a.Attribute("type", d.String, "Type of the tracker", func() {
		a.Example("github")
		a.MinLength(1)
		a.Pattern("^[\\p{L}]+$")
	})
	a.Required("url", "type")
})

// CreateTrackerQueryAlternatePayload defines the structure of tracker query payload for create
var CreateTrackerQueryAlternatePayload = a.Type("CreateTrackerQueryAlternatePayload", func() {
	a.Attribute("query", d.String, "Search query", func() {
		a.Example("is:open is:issue user:almighty")
		a.MinLength(1)
	})
	a.Attribute("schedule", d.String, "Schedule for fetch and import", func() {
		a.Example("0 0/15 * * * *")
		a.Pattern("^[\\d]+|[\\d]+[\\/][\\d]+|\\*|\\-|\\?\\s{0,6}$")
		a.MinLength(1)
	})
	a.Attribute("trackerID", d.String, "Tracker ID", func() {
		a.Example("1")
		a.MinLength(1)
		a.Pattern("^[\\p{N}]+$")
	})
	a.Attribute("relationships", trackerQueryRelationships)

	a.Required("query", "schedule", "trackerID")
})

// UpdateTrackerQueryAlternatePayload defines the structure of tracker query payload for update
var UpdateTrackerQueryAlternatePayload = a.Type("UpdateTrackerQueryAlternatePayload", func() {
	a.Attribute("query", d.String, "Search query", func() {
		a.Example("is:open is:issue user:almighty")
		a.MinLength(1)
	})
	a.Attribute("schedule", d.String, "Schedule for fetch and import", func() {
		a.Example("0 0/15 * * * *")
		a.Pattern("^[\\d]+|[\\d]+[\\/][\\d]+|\\*|\\-|\\?\\s{0,6}$")
		a.MinLength(1)
	})
	a.Attribute("trackerID", d.String, "Tracker ID", func() {
		a.Example("1")
		a.MinLength(1)
		a.Pattern("[\\p{N}]+")
	})
	a.Attribute("relationships", trackerQueryRelationships)

	a.Required("query", "schedule", "trackerID")
})

// Users represents a user object (TODO: add better description)
var Users = a.Type("Users", func() {
	a.Description(`JSONAPI store for the data of a user.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("users")
	})
	a.Attribute("id", d.String, "ID of work item link type (optional during creation)", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", UsersAttributes)
	a.Required("type", "attributes")
})

// UserAttributes is the JSONAPI store for all the "attributes" of a user.
var UsersAttributes = a.Type("UserAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a user. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("fullname", d.String, "The users full name", func() {
		a.Example("John Smith")
	})
	a.Attribute("image-url", d.String, "The avatar image for the user", func() {
		a.Example("http://example.org/user/image.png")
	})
})
