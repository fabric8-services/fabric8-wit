package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

var _ = Resource("workitem", func() {
	BasePath("/workitem")

	Action("show", func() {
		Routing(
			GET("/:id"),
		)
		Description("Retrieve work item with given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK, func() {
			Media(WorkItem)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

	Action("list", func() {
		Routing(
			GET(""),
		)
		Description("List work items.")
		Params(func() {
			Param("filter", String, "a query language expression restricting the set of found items")
			Param("page", String, "Paging in the format <start>,<limit>")
		})
		Response(OK, func() {
			Media(CollectionOf(WorkItem))
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
	})

	Action("create", func() {
		Routing(
			POST(""),
		)
		Description("create work item with type and id.")
		Payload(CreateWorkItemPayload)
		Response(Created, "/workitem/.*", func() {
			Media(WorkItem)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})
	Action("delete", func() {
		Routing(
			DELETE("/:id"),
		)
		Description("Delete work item with given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK)
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})
	Action("update", func() {
		Routing(
			PUT("/:id"),
		)
		Description("update the given work item with given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Payload(UpdateWorkItemPayload)
		Response(OK, func() {
			Media(WorkItem)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

})

var _ = Resource("workitemtype", func() {

	BasePath("/workitemtype")

	Action("show", func() {

		Routing(
			GET("/:id"),
		)
		Description("Retrieve work item type with given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK, func() {
			Media(WorkItemType)
		})
		Response(NotFound)
	})
})

var _ = Resource("version", func() {

	DefaultMedia(ALMVersion)
	BasePath("/version")

	Action("show", func() {
		Security("jwt", func() {
			Scope("system")
		})
		Routing(
			GET(""),
		)
		Description("Show current running version")
		Response(OK)
	})
})

var _ = Resource("login", func() {

	BasePath("/login")

	Action("authorize", func() {
		Routing(
			GET("authorize"),
		)
		Description("Authorize with the ALM")
		Response(OK, func() {
			Media(AuthToken)
		})
		Response(Unauthorized)
	})

	Action("generate", func() {
		Routing(
			GET("generate"),
		)
		Description("Generates a set of Tokens for different Auth levels. NOT FOR PRODUCTION. Only available if server is running in dev mode")
		Response(OK, func() {
			Media(CollectionOf(AuthToken))
		})
		Response(Unauthorized)
	})
})
