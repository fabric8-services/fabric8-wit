package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// CreateWorkItemPayload defines the structure of work item payload
var CreateWorkItemPayload = a.Type("CreateWorkItemPayload", func() {
	a.Attribute("type", d.String, "The type of the newly created work item", func() {
		a.Example("system.userstory")
		a.MinLength(1)
		a.Pattern("^[\\p{L}.]+$")
	})
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
	a.Attribute("type", d.String, "The type of the newly created work item", func() {
		a.Example("system.userstory")
		a.MinLength(1)
		a.Pattern("^[\\p{L}.]+$")
	})
	a.Attribute("fields", a.HashOf(d.String, d.Any), "The field values, must conform to the type", func() {
		a.Example(map[string]interface{}{"system.creator": "user-ref", "system.state": "new", "system.title": "Example story"})
		a.MinLength(1)
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control", func() {
		a.Example(0)
	})
	a.Required("type", "fields", "version")
})

// UpdateWorkItemJSONAPIPayload defines top level structure from jsonapi specs
// visit : http://jsonapi.org/format/#document-top-level
var updateWorkItemJSONAPIPayload = a.Type("UpdateWorkItemJSONAPIPayload", func() {
	a.Attribute("data", workItemDataForUpdate)
	a.Required("data")
})

// WorkItemDataForUpdate defines how an update payload will look like
var workItemDataForUpdate = a.Type("WorkItemDataForUpdate", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitems")
	})
	a.Attribute("id", d.String, "ID of the work item which is being updated", func() {
		a.Example("42")
	})
	a.Attribute("attributes", a.HashOf(d.String, d.Any), func() {
		a.Example(map[string]interface{}{"version": "1", "system.state": "new", "system.title": "Example story"})
	})
	a.Attribute("relationships", workItemRelationships)
	// relationships must be required becasue we MUST have workItemType during PATCh
	a.Required("type", "id", "attributes")
})

// WorkItemRelationships defines only `assignee` as of now. To be updated
var workItemRelationships = a.Type("WorkItemRelationships", func() {
	a.Attribute("assignee", RelationAssignee, "This deinfes assignees of the WI")
	a.Attribute("baseType", RelationBaseType, "This defines type of Work Item")
	// baseType relationship must present while updating work item
})

// RelationAssignee is a top level structure for assignee relationship
var RelationAssignee = a.Type("RelationAssignee", func() {
	a.Attribute("data", AssigneeData)
	a.Required("data")
})

// AssigneeData defines what is needed inside Assignee Relationship
var AssigneeData = a.Type("AssigneeData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("identities")
	})
	a.Attribute("id", d.String, "UUID of the identity", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Required("type")
	// a.Required("id") if ID is nil then we remove assignee
})

// RelationBaseType is top level block for WorkItemType relationship
var RelationBaseType = a.Type("RelationshipBaseType", func() {
	a.Attribute("data", BaseTypeData)
	a.Required("data")
})

// BaseTypeData is data block for `type` of a work item
var BaseTypeData = a.Type("BaseTypeData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.String, func() {
		a.Example("system.userstory")
	})
	a.Required("type", "id")
})

// CreateWorkItemTypePayload explains how input payload should look like
var CreateWorkItemTypePayload = a.Type("CreateWorkItemTypePayload", func() {
	a.Attribute("name", d.String, "Readable name of the type like Task, Issue, Bug, Epic etc.", func() {
		a.Example("Epic")
		a.Pattern("^[\\p{L}.]+$")
		a.MinLength(1)
	})
	a.Attribute("fields", a.HashOf(d.String, fieldDefinition), "Type fields those must be followed by respective Work Items.", func() {
		a.Example(map[string]interface{}{
			"system.administrator": map[string]interface{}{
				"Type": map[string]interface{}{
					"Kind": "string",
				},
				"Required": true,
			},
		})
		a.MinLength(1)
	})
	a.Attribute("extendedTypeName", d.String, "If newly created type extends any existing type", func() {
		a.Example("(optional field)Parent type name")
		a.MinLength(1)
		a.Pattern("^[\\p{L}.]+$")
	})
	a.Required("name", "fields")
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
	a.Required("query", "schedule", "trackerID")
})

// identityDataAttributes represents an identified user object attributes
var identityDataAttributes = a.Type("IdentityDataAttributes", func() {
	a.Attribute("fullName", d.String, "The users full name")
	a.Attribute("imageURL", d.String, "The avatar image for the user")
})

// identityData represents an identified user object
var identityData = a.Type("IdentityData", func() {
	a.Attribute("id", d.String, "unique id for the user identity")
	a.Attribute("type", d.String, "type of the user identity")
	a.Attribute("attributes", identityDataAttributes, "Attributes of the user identity")
	a.Required("type", "attributes")
})

//#############################################################################
//
// 			JSONAPI common
//
//#############################################################################

// JSONAPILink represents a JSONAPI link object (see http://jsonapi.org/format/#document-links)
var JSONAPILink = a.Type("JSONAPILink", func() {
	a.Description(`See also http://jsonapi.org/format/#document-links.`)
	a.Attribute("href", d.String, "a string containing the link's URL.", func() {
		a.Example("http://example.com/articles/1/comments")
	})
	a.Attribute("meta", a.HashOf(d.String, d.Any), "a meta object containing non-standard meta-information about the link.")
})

// JSONAPIError represents a JSONAPI error object (see http://jsonapi.org/format/#error-objects)
var JSONAPIError = a.Type("JSONAPIError", func() {
	a.Description(`Error objects provide additional information about problems encountered while
performing an operation. Error objects MUST be returned as an array keyed by errors in the
top level of a JSON API document.

See. also http://jsonapi.org/format/#error-objects.`)

	a.Attribute("id", d.String, "a unique identifier for this particular occurrence of the problem.")
	a.Attribute("links", a.HashOf(d.String, JSONAPILink), `a links object containing the following members:
* about: a link that leads to further details about this particular occurrence of the problem.`)
	a.Attribute("status", d.String, "the HTTP status code applicable to this problem, expressed as a string value.")
	a.Attribute("code", d.String, "an application-specific error code, expressed as a string value.")
	a.Attribute("title", d.String, `a short, human-readable summary of the problem that SHOULD NOT
change from occurrence to occurrence of the problem, except for purposes of localization.`)
	a.Attribute("detail", d.String, `a human-readable explanation specific to this occurrence of the problem.
Like title, this field’s value can be localized.`)
	a.Attribute("source", a.HashOf(d.String, d.Any), `an object containing references to the source of the error,
optionally including any of the following members

* pointer: a JSON Pointer [RFC6901] to the associated entity in the request document [e.g. "/data" for a primary data object,
           or "/data/attributes/title" for a specific attribute].
* parameter: a string indicating which URI query parameter caused the error.`)
	a.Attribute("meta", a.HashOf(d.String, d.Any), "a meta object containing non-standard meta-information about the error")

	a.Required("detail")
})

// WorkItemResourceLinksForJSONAPI has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var WorkItemResourceLinksForJSONAPI = a.Type("WorkItemResourceLinksForJSONAPI", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.almighty.io/api/workitems.2/1")
	})
})
