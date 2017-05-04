package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// createWorkItemLinkTypePayload defines the structure of work item link type combination payload in JSONAPI format during creation
var createWorkItemLinkTypeCombinationPayload = a.Type("CreateWorkItemLinkTypeCombinationPayload", func() {
	a.Attribute("data", workItemLinkTypeCombinationData)
	a.Required("data")
})

// workItemLinkTypeCombinationData is the JSONAPI store for the data of a work item link type combination.
var workItemLinkTypeCombinationData = a.Type("WorkItemLinkTypeCombinationData", func() {
	a.Description(`JSONAPI store for the data of a work item link type combination.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinktypecombinations")
	})
	a.Attribute("id", d.UUID, "ID of work item link type combination(optional during creation)")
	a.Attribute("attributes", workItemLinkTypeCombinationAttributes)
	a.Attribute("relationships", workItemLinkTypeCombinationRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes", "relationships")
})

// workItemLinkTypeCombinationAttributes is the JSONAPI store for all the "attributes" of a work item link type combination.
var workItemLinkTypeCombinationAttributes = a.Type("WorkItemLinkTypeCombinationAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link type combination.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})
	a.Attribute("created-at", d.DateTime, "Time of creation of the given work item type")
	a.Attribute("updated-at", d.DateTime, "Time of last update of the given work item type")
})

// workItemLinkTypeCombinationRelationships is the JSONAPI store for the relationships of a work item link type combination.
var workItemLinkTypeCombinationRelationships = a.Type("WorkItemLinkTypeCombinationRelationships", func() {
	a.Description(`JSONAPI store for the data of a work item link type combination.
See also http://jsonapi.org/format/#document-resource-object-relationships`)
	a.Attribute("link_type", relationWorkItemLinkType, "The type of link that this link belongs to.")
	a.Attribute("source_type", relationWorkItemType, "The source type specifies the type of work item that can be used as a source.")
	a.Attribute("target_type", relationWorkItemType, "The target type specifies the type of work item that can be used as a target.")
	a.Attribute("space", relationSpaces, "This defines the owning space of this work item link type combination.")
	a.Required("source_type", "target_type", "space", "link_type")
})

// relationWorkItemType is the JSONAPI store for the work item type relationship objects
var relationWorkItemType = a.Type("RelationWorkItemType", func() {
	a.Attribute("data", relationWorkItemTypeData)
	a.Attribute("links", genericLinks)
})

// relationWorkItemTypeData is the JSONAPI data object of the the work item type relationship objects
var relationWorkItemTypeData = a.Type("RelationWorkItemTypeData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.UUID, "ID of a work item type")
	a.Required("type", "id")
})

// relationWorkItemLinkTypeCombination is the JSONAPI store for the links
var relationWorkItemLinkTypeCombination = a.Type("RelationWorkItemLinkTypeCombination", func() {
	a.Attribute("data", relationWorkItemLinkTypeCombinationData)
})

// relationWorkItemLinkTypeCombinationData is the JSONAPI data object of the the work item link type combinationrelationship objects
var relationWorkItemLinkTypeCombinationData = a.Type("RelationWorkItemLinkTypeCombinationData", func() {
	a.Attribute("type", d.String, "The type of the related source", func() {
		a.Enum("workItemLinkTypeCombinations")
	})
	a.Attribute("id", d.UUID, "ID of work item link type combination")
	a.Required("type", "id")
})

// ############################################################################
//
//  Media Type Definition
//
// ############################################################################

var workItemLinkTypeCombinationList = JSONList(
	"WorkItemLinkTypeCombination", "Holds the list of work item link type combinations",
	workItemLinkTypeCombinationData,
	pagingLinks,
	meta)

var workItemLinkTypeCombinationSingle = JSONSingle(
	"WorkItemLinkTypeCombination", "Holds a single work item link type combination",
	workItemLinkTypeCombinationData,
	nil)

// ############################################################################
//
//  Resource Definition
//
// ############################################################################

var _ = a.Resource("work_item_link_type_combination", func() {
	a.BasePath("/workitemlinktypecombinations")
	a.Parent("space")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:wiltcId"),
		)
		a.Description("Retrieve work item link type combination (as JSONAPI) for the given link ID.")
		a.Params(func() {
			a.Param("wiltcId", d.UUID, "ID of the work item link type combination")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemLinkTypeCombinationSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	// a.Action("list", func() {
	// 	a.Routing(
	// 		a.GET(""),
	// 	)
	// 	a.Routing(
	// 		a.GET("/:wiltId"),
	// 	)
	// 	a.Description("Retrieve work item link type combination (as JSONAPI) for the given link type ID.")
	// 	a.Params(func() {
	// 		a.Param("wiltId", d.UUID, "ID of the work item link type")
	// 	})
	// 	a.UseTrait("conditional")
	// 	a.Response(d.OK, workItemLinkTypeCombinationList)
	// 	a.Response(d.NotModified)
	// 	a.Response(d.BadRequest, JSONAPIErrors)
	// 	a.Response(d.InternalServerError, JSONAPIErrors)
	// })

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a work item link type combination")
		a.Payload(createWorkItemLinkTypeCombinationPayload)
		a.Response(d.Created, "/workItemLinkTypeCombinations/.*", func() {
			a.Media(workItemLinkTypeCombinationSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})

var _ = a.Resource("redirect_work_item_link_type_combination", func() {
	a.BasePath("/workitemlinktypecombinations")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:wiltcId"),
		)
		a.Params(func() {
			a.Param("wiltcId", d.UUID, "ID of the work item link type combination")
		})
		a.Response(d.MovedPermanently)
	})

	a.Action("create", func() {
		a.Routing(
			a.POST(""),
		)
		a.Response(d.MovedPermanently)
	})
})
