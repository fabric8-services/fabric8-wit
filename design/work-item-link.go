package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

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
