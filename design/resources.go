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
		Response(NotFound)
	})

	Action("create", func() {
		Routing(
			POST(""),
		)
		Description("create work item with type and id.")
		Payload(CreateWorkItemPayload)
		Response(OK, func() {
			Media(WorkItem)
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
		Response(BadRequest)
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
