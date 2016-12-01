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

//#############################################################################
//
// 			work item link category
//
//#############################################################################

// CreateWorkItemLinkCategoryPayload defines the structure of work item link category payload in JSONAPI format during creation
var CreateWorkItemLinkCategoryPayload = a.Type("CreateWorkItemLinkCategoryPayload", func() {
	a.Attribute("data", WorkItemLinkCategoryData)
	a.Required("data")
})

// UpdateWorkItemLinkCategoryPayload defines the structure of work item link category payload in JSONAPI format during update
var UpdateWorkItemLinkCategoryPayload = a.Type("UpdateWorkItemLinkCategoryPayload", func() {
	a.Attribute("data", WorkItemLinkCategoryData)
	a.Required("data")
})

// WorkItemLinkCategoryArrayMeta holds meta information for a work item link category array response
var WorkItemLinkCategoryArrayMeta = a.Type("WorkItemLinkCategoryArrayMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// WorkItemLinkCategoryData is the JSONAPI store for the data of a work item link category.
var WorkItemLinkCategoryData = a.Type("WorkItemLinkCategoryData", func() {
	a.Description(`JSONAPI store the data of a work item link category.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinkcategories")
	})
	a.Attribute("id", d.String, "ID of work item link category (optional during creation)", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Attribute("attributes", WorkItemLinkCategoryAttributes)
	a.Required("type", "attributes")
})

// WorkItemLinkCategoryAttributes is the JSONAPI store for all the "attributes" of a work item link category.
var WorkItemLinkCategoryAttributes = a.Type("WorkItemLinkCategoryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link category.
See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "Name of the work item link category (required on creation, optional on update)", func() {
		a.Example("system")
	})
	a.Attribute("description", d.String, "Description of the work item link category (optional)", func() {
		a.Example("A work item link category that is meant only for work item link types goverened by the system alone.")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})

	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link category.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("name")
})

//#############################################################################
//
// 			work item link type
//
//#############################################################################

// CreateWorkItemLinkTypePayload defines the structure of work item link type payload in JSONAPI format during creation
var CreateWorkItemLinkTypePayload = a.Type("CreateWorkItemLinkTypePayload", func() {
	a.Attribute("data", WorkItemLinkTypeData)
	a.Required("data")
})

// UpdateWorkItemLinkTypePayload defines the structure of work item link type payload in JSONAPI format during update
var UpdateWorkItemLinkTypePayload = a.Type("UpdateWorkItemLinkTypePayload", func() {
	a.Attribute("data", WorkItemLinkTypeData)
	a.Required("data")
})

// WorkItemLinkTypeArrayMeta holds meta information for a work item link type array response
var WorkItemLinkTypeArrayMeta = a.Type("WorkItemLinkTypeArrayMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// WorkItemLinkTypeData is the JSONAPI store for the data of a work item link type.
var WorkItemLinkTypeData = a.Type("WorkItemLinkTypeData", func() {
	a.Description(`JSONAPI store for the data of a work item link type.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinktypes")
	})
	a.Attribute("id", d.String, "ID of work item link type (optional during creation)", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", WorkItemLinkTypeAttributes)
	a.Attribute("relationships", WorkItemLinkTypeRelationships)
	a.Required("type", "attributes")
})

// WorkItemLinkTypeAttributes is the JSONAPI store for all the "attributes" of a work item link type.
var WorkItemLinkTypeAttributes = a.Type("WorkItemLinkTypeAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link type.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "Name of the work item link type (required on creation, optional on update)", func() {
		a.Example("tested-by-link-type")
	})
	a.Attribute("description", d.String, "Description of the work item link type (optional)", func() {
		a.Example("A test work item can 'test' if a the code in a pull request passes the tests.")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})
	a.Attribute("forward_name", d.String, `The forward oriented path from source to target is described with the forward name.
For example, if a bug blocks a user story, the forward name is "blocks". See also reverse name.`, func() {
		a.Example("test-workitemtype")
	})
	a.Attribute("reverse_name", d.String, `The backwards oriented path from target to source is described with the reverse name.
For example, if a bug blocks a user story, the reverse name name is "blocked by" as in: a user story is blocked by a bug. See also forward name.`, func() {
		a.Example("tested by")
	})
	a.Attribute("topology", d.String, `The topology determines the restrictions placed on the usage of each work item link type.`, func() {
		a.Enum("network")
	})

	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link type.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("name")
})

// WorkItemLinkTypeRelationships is the JSONAPI store for the relationships of a work item link type.
var WorkItemLinkTypeRelationships = a.Type("WorkItemLinkTypeRelationships", func() {
	a.Description(`JSONAPI store for the data of a work item link type.
See also http://jsonapi.org/format/#document-resource-object-relationships`)
	a.Attribute("link_category", RelationWorkItemLinkCategory, "The work item link category of this work item link type.")
	a.Attribute("source_type", RelationWorkItemType, "The source type specifies the type of work item that can be used as a source.")
	a.Attribute("target_type", RelationWorkItemType, "The target type specifies the type of work item that can be used as a target.")
})

// RelationWorkItemLinkCategory is the JSONAPI store for the links
var RelationWorkItemLinkCategory = a.Type("RelationWorkItemLinkCategory", func() {
	a.Attribute("data", RelationWorkItemLinkCategoryData)
})

// RelationWorkItemType is the JSONAPI store for the work item type relationship objects
var RelationWorkItemType = a.Type("RelationWorkItemType", func() {
	a.Attribute("data", RelationWorkItemTypeData)
})

// RelationWorkItemTypeData is the JSONAPI data object of the the work item link category relationship objects
var RelationWorkItemLinkCategoryData = a.Type("RelationWorkItemLinkCategoryData", func() {
	a.Attribute("type", d.String, "The type of the related source", func() {
		a.Enum("workitemlinkcategories")
	})
	a.Attribute("id", d.String, "ID of work item link category", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Required("type", "id")
})

// RelationWorkItemTypeData is the JSONAPI data object of the the work item type relationship objects
var RelationWorkItemTypeData = a.Type("RelationWorkItemTypeData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.String, "Name work item type", func() {
		a.Example("system.bug")
	})
	a.Required("type", "id")
})

//#############################################################################
//
// 			work item link
//
//#############################################################################

// CreateWorkItemLinkPayload defines the structure of work item link payload in JSONAPI format during creation
var CreateWorkItemLinkPayload = a.Type("CreateWorkItemLinkPayload", func() {
	a.Attribute("data", WorkItemLinkData)
	a.Required("data")
})

// UpdateWorkItemLinkPayload defines the structure of work item link payload in JSONAPI format during update
var UpdateWorkItemLinkPayload = a.Type("UpdateWorkItemLinkPayload", func() {
	a.Attribute("data", WorkItemLinkData)
	a.Required("data")
})

// WorkItemLinkArrayMeta holds meta information for a work item link array response
var WorkItemLinkArrayMeta = a.Type("WorkItemLinkArrayMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// WorkItemLinkData is the JSONAPI store for the data of a work item link.
var WorkItemLinkData = a.Type("WorkItemLinkData", func() {
	a.Description(`JSONAPI store for the data of a work item.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinks")
	})
	a.Attribute("id", d.String, "ID of work item link (optional during creation)", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", WorkItemLinkAttributes)
	a.Attribute("relationships", WorkItemLinkRelationships)
	a.Required("type", "relationships")
})

// WorkItemLinkAttributes is the JSONAPI store for all the "attributes" of a work item link type.
var WorkItemLinkAttributes = a.Type("WorkItemLinkAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})
	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link type.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("version")
})

// WorkItemLinkRelationships is the JSONAPI store for the relationships of a work item link.
var WorkItemLinkRelationships = a.Type("WorkItemLinkRelationships", func() {
	a.Description(`JSONAPI store for the data of a work item link.
See also http://jsonapi.org/format/#document-resource-object-relationships`)
	a.Attribute("link_type", RelationWorkItemLinkType, "The work item link type of this work item link.")
	a.Attribute("source", RelationWorkItem, "Work item where the connection starts.")
	a.Attribute("target", RelationWorkItem, "Work item where the connection ends.")
})

// RelationWorkItemLinkType is the JSONAPI store for the links
var RelationWorkItemLinkType = a.Type("RelationWorkItemLinkType", func() {
	a.Attribute("data", RelationWorkItemLinkTypeData)
})

// RelationWorkItem is the JSONAPI store for the links
var RelationWorkItem = a.Type("RelationWorkItem", func() {
	a.Attribute("data", RelationWorkItemData)
})

// RelationWorkItemLinkTypeData is the JSONAPI data object of the the work item link type relationship objects
var RelationWorkItemLinkTypeData = a.Type("RelationWorkItemLinkTypeData", func() {
	a.Attribute("type", d.String, "The type of the related source", func() {
		a.Enum("workitemlinktypes")
	})
	a.Attribute("id", d.String, "ID of work item link type", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Required("type", "id")
})

// RelationWorkItemData is the JSONAPI data object of the the work item relationship objects
var RelationWorkItemData = a.Type("RelationWorkItemData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("workitems")
	})
	a.Attribute("id", d.String, "ID of the work item", func() {
		a.Example("1234")
	})
	a.Required("type", "id")
})

// WorkItemResourceLinksForJSONAPI has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var WorkItemResourceLinksForJSONAPI = a.Type("WorkItemResourceLinksForJSONAPI", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.almighty.io/api/workitems.2/1")
	})
})
