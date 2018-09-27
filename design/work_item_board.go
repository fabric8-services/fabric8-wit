package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var workItemBoardSingle = JSONSingle(
	"workItemBoard",
	`A board for work item types`,
	workItemBoardData,
	workItemBoardLinks,
)

var workItemBoardList = JSONList(
	"workItemBoard",
	`List of work item boards`,
	workItemBoardData,
	workItemBoardLinks,
	nil,
)

var workItemBoardLinks = a.Type("WorkItemBoardLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/spacetemplates/2d98c73d-6969-4ea6-958a-812c832b6c18/workitemboards")
	})
	a.Required("self")
})

var _ = a.Type("WorkItemBoardColumnData", func() {
	a.Description(`a column represents a vertical lane in a board`)
	a.Attribute("type", d.String, "The type string of the work item board column", func() {
		a.Enum("boardcolumns")
	})
	a.Attribute("id", d.UUID, "ID of the work item board column", func() {
		a.Example("712f20e4-2202-4469-9a02-b892b7051b2b")
	})
	a.Attribute("attributes", workItemBoardColumnAttributes)
	a.Required("id", "type", "attributes")
})

var workItemBoardColumnAttributes = a.Type("WorkItemBoardColumnAttributes", func() {
	a.Attribute("name", d.String)
	a.Attribute("order", d.Integer)
	// TODO(michaelkleinhenz): as soon as we allow column customization, we need
	// to also provide transRuleKey and transRuleArguments.
	a.Required("name")
})

var workItemBoardData = a.Type("WorkItemBoardData", func() {
	a.Description(`a board shows different work item type together in a board`)
	a.Attribute("type", d.String, "The type string of the work item board", func() {
		a.Enum("workitemboards")
	})
	a.Attribute("id", d.UUID, "ID of the work item board", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", workItemBoardAttributes)
	a.Attribute("relationships", workItemBoardRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var workItemBoardAttributes = a.Type("WorkItemBoardAttributes", func() {
	a.Attribute("name", d.String)
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control")
	a.Attribute("created-at", d.DateTime, "timestamp of entity creation")
	a.Attribute("updated-at", d.DateTime, "timestamp of last entity update")
	a.Attribute("context", d.String, "Context of this board")
	a.Attribute("contextType", d.String, "Type of the context, used in addition to the context value", func() {
		// TODO(kwk): once we allow more context types, this can be relaxed.
		a.Enum("TypeLevelContext")
	})
	a.Required("name", "context", "contextType")
})

var workItemBoardRelationships = a.Type("WorkItemBoardRelationships", func() {
	a.Attribute("columns", relationGenericList, "List of work item board columns attached to the board")
	a.Attribute("spaceTemplate", relationGeneric, "The space template to which this board belongs")
})

var _ = a.Resource("work_item_boards", func() {
	a.BasePath("/workitemboards")
	a.Parent("space_template")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item boards")
		a.Response(d.OK, workItemBoardList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})

var _ = a.Resource("work_item_board", func() {
	a.BasePath("/workitemboards")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:boardID"),
		)
		a.Params(func() {
			a.Param("boardID", d.UUID, "ID of the work item board")
		})
		a.Description("Show work item board for given ID")
		a.Response(d.OK, workItemBoardSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
