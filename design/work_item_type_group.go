package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var workItemTypeGroupSingle = JSONSingle(
	"workItemTypeGroup",
	`A group of work item types`,
	workItemTypeGroupData,
	workItemTypeGroupLinks,
)

var workItemTypeGroupList = JSONList(
	"workItemTypeGroup",
	`List of work item type groups`,
	workItemTypeGroupData,
	workItemTypeGroupLinks,
	nil,
)

var workItemTypeGroupLinks = a.Type("WorkItemTypeGroupLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/spacetemplates/2d98c73d-6969-4ea6-958a-812c832b6c18/workitemtypegroups")
	})
	a.Required("self")
})

var workItemTypeGroupData = a.Type("WorkItemTypeGroupData", func() {
	a.Description(`a type group bundles different work item type together`)
	a.Attribute("type", d.String, "The type string of the work item type group", func() {
		a.Enum("workitemtypegroups")
	})
	a.Attribute("id", d.UUID, "ID of the work item type group", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", workItemTypeGroupAttributes)
	a.Attribute("relationships", workItemTypeGroupsRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var workItemTypeGroupAttributes = a.Type("WorkItemTypeGroupAttributes", func() {
	a.Attribute("bucket", d.String, "Name of the bucket this group belongs to")
	a.Attribute("name", d.String)
	a.Attribute("show-in-sidebar", d.Boolean, "Whether or not to render a link for this type group in the sidebar")
	a.Attribute("icon", d.String, "CSS property value for icon of the group")
	a.Attribute("created-at", d.DateTime, "timestamp of entity creation")
	a.Attribute("updated-at", d.DateTime, "timestamp of last entity update")
	a.Required("bucket", "name", "icon")
})

var workItemTypeGroupsRelationships = a.Type("WorkItemTypeGroupRelationships", func() {
	a.Attribute("defaultType", relationGeneric, "The default work item type from the type list")
	a.Attribute("typeList", relationGenericList, "List of work item types attached to the type group")
	a.Attribute("spaceTemplate", relationGeneric, "The space template to which this group belongs")
	a.Attribute("nextGroup", relationGeneric, "The type group (if any) that comes after this one in the list")
	a.Attribute("prevGroup", relationGeneric, "The type group (if any) that comes before this one in the list")
})

var _ = a.Resource("work_item_type_groups", func() {
	a.BasePath("/workitemtypegroups")
	a.Parent("space_template")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List of work item type groups")
		a.Response(d.OK, workItemTypeGroupList)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})

var _ = a.Resource("work_item_type_group", func() {
	a.BasePath("/workitemtypegroups")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:groupID"),
		)
		a.Params(func() {
			a.Param("groupID", d.UUID, "ID of the work item type")
		})
		a.Description("Show work item type group for given ID")
		a.Response(d.OK, workItemTypeGroupSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
