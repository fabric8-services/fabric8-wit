package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var searchWorkItemList = JSONList(
	"SearchWorkItem", "Holds the paginated response to a search request",
	workItem,
	pagingLinks,
	meta)

var searchSpaceList = JSONList(
	"SearchSpace", "Holds the paginated response to a search for spaces request",
	space,
	pagingLinks,
	spaceListMeta)

// var searchCodebaseList = JSONList(
// 	"SearchCodebase", "Holds the paginated response to a search for codebases request",
// 	searchCodebaseType,
// 	pagingLinks,
// 	searchCodebaseListMeta)

// var searchCodebaseType = a.Type("SearchCodebase", func() {
// 	a.Description(`JSONAPI store for the data of a codebase.  See also http://jsonapi.org/format/#document-resource-object`)
// 	a.Attribute("type", d.String, func() {
// 		a.Enum("codebases")
// 	})
// 	a.Attribute("id", d.UUID, "ID of codebase", func() {
// 		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
// 	})
// 	a.Attribute("attributes", searchCodebaseAttributes)
// 	a.Attribute("relationships", searchCodebaseRelationships)
// 	a.Attribute("links", searchCodebaseLinks)
// 	a.Required("type", "attributes")
// })

// var searchCodebaseAttributes = a.Type("SearchCodebaseAttributes", func() {
// 	a.Description(`JSONAPI store for all the "attributes" of a codebase. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
// 	a.Attribute("type", d.String, "The codebase type", func() {
// 		a.Example("git")
// 	})
// 	a.Attribute("url", d.String, "The URL of the codebase ", func() {
// 		a.Example("git@github.com:fabric8-services/fabric8-wit.git")
// 	})
// 	a.Attribute("stackId", d.String, "The stack id of the codebase ", func() {
// 		a.Example("java-centos")
// 	})
// 	a.Attribute("createdAt", d.DateTime, "When the codebase was created", func() {
// 		a.Example("2016-11-29T23:18:14Z")
// 	})
// 	a.Attribute("last_used_workspace", d.String, "The last used workspace name of the codebase ", func() {
// 		a.Example("java-centos")
// 	})
// })

// var searchCodebaseLinks = a.Type("SearchCodebaseLinks", func() {
// 	a.UseTrait("GenericLinksTrait")
// })
// var searchCodebaseRelationships = a.Type("SearchCodebaseRelations", func() {
// 	a.Attribute("space", relationGeneric, "This defines the owning space")
// })

// var searchCodebaseListMeta = a.Type("SearchCodebaseListMeta", func() {
// 	a.Attribute("totalCount", d.Integer)
// 	a.Required("totalCount")
// })

var _ = a.Resource("search", func() {
	a.BasePath("/search")

	a.Action("show", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Search by ID, URL, full text capability")
		a.Params(func() {
			a.Param("q", d.String,
				`Following are valid input for search query
				1) "id:100" :- Look for work item hainvg id 100
				2) "url:http://demo.openshift.io/details/500" :- Search on WI having id 500 and check 
					if this URL is mentioned in searchable columns of work item
				3) "simple keywords separated by space" :- Search in Work Items based on these keywords.`)
			a.Param("page[offset]", d.String, "Paging start position") // #428
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Param("filter[parentexists]", d.Boolean, "if false list work items without any parent")
			a.Param("filter[expression]", d.String, "Filter expression in JSON format", func() {
				a.Example(`{$AND: [{"space": "f73988a2-1916-4572-910b-2df23df4dcc3"}, {"state": "NEW"}]}`)
			})
			a.Param("spaceID", d.String, "The optional space ID of the space to be searched in, if the filter[expression] query parameter is not provided")
		})
		a.Response(d.OK, func() {
			a.Media(searchWorkItemList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("spaces", func() {
		a.Routing(
			a.GET("spaces"),
		)
		a.Description("Search for spaces by name or description")
		a.Params(func() {
			a.Param("q", d.String, "Text to match against Name or description")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Required("q")
		})
		a.Response(d.OK, func() {
			a.Media(searchSpaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("users", func() {
		a.Routing(
			a.GET("users"),
		)
		a.Description("Search by fullname")
		a.Response(d.OK)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("codebases", func() {
		a.Routing(
			a.GET("codebases"),
		)
		a.Description("Search by URL")
		a.Params(func() {
			a.Param("url", d.String)
			a.Param("page[offset]", d.String, "Paging start position") // #428
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Required("url")
		})
		a.Response(d.OK, func() {
			a.Media(codebaseList)
		})

		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})

		a.Response(d.InternalServerError)
	})
})
