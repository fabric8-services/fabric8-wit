package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var workItemTypeHierarchySigle = JSONSingle(
	"workItemTypeHierarchy",
	`Group of the work item types groups`,
	workItemTypeHierarchyData,
	workItemTypeHierarchyLinks,
)

// workItemLinkTypeListMeta holds meta information for a work item link type array response
// var workItemTypeHierarchyListMeta = a.Type("WorkItemTypeHierarchyListMeta", func() {
// 	a.Attribute("totalCount", d.Integer, func() {
// 		a.Minimum(0)
// 	})
// 	a.Required("totalCount")
// })

// workItemTypeHierarchyList contains work item type groups
var workItemTypeHierarchyList = JSONList(
	"WorkItemTypeHierarchyList",
	"...",
	workItemTypeHierarchyData,
	nil, //pagingLinks,
	// workItemTypeHierarchyListMeta,
	nil,
)

// workItemLinkTypeLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemTypeHierarchyLinks = a.Type("WorkItemTypeHierarchyLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitemlinktypes/2d98c73d-6969-4ea6-958a-812c832b6c18")
	})
	a.Required("self")
})

var workItemTypeHierarchy = a.Type("WorkItemTypeHierarchy", func() {
	a.Attribute("level", d.Integer)
	a.Attribute("sublevel", d.Integer, "ID of work item type (optional during creation)")
	a.Attribute("group", d.String)
	a.Attribute("name", d.String)
	a.Attribute("wit_collection", a.ArrayOf(d.UUID))
	// a.Required("attributes", "relationships")
})

// workItemLinkTypeAttributes is the JSONAPI store for all the "attributes" of a work item link type.
var workItemTypeHierarchyAttributes = a.Type("WorkItemTypeHierarchyAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link type.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("hierarchy", a.ArrayOf(workItemTypeHierarchy))
})

// workItemTypeHierarchyData is the JSONAPI store for the data of a work item link type.
var workItemTypeHierarchyData = a.Type("WorkItemTypeHierarchyData", func() {
	a.Description(`JSONAPI store for the data of a work item link type.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("id", d.UUID, "ID of work item link type (optional during creation)")
	a.Attribute("attributes", workItemTypeHierarchyAttributes)
	a.Attribute("included", a.ArrayOf(d.Any), "An array of mixed types")
	a.Required("attributes", "id")
})

var _ = a.Resource("work_item_type_group", func() {
	a.BasePath("/workitemtypegroups")
	a.Parent("space_template")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List of work item type groups.")
		// a.UseTrait("conditional")
		a.Response(d.OK, workItemTypeHierarchyList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
