package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

var _ = Resource("workitem", func() {
	BasePath("/workitems")

	Action("show", func() {
		Routing(
			GET("/:id"),
		)
		Description("Retrieve work item with given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK, func() {
			Media(workItem)
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
			Param("filter", String, "a query language expression restricting the set of found work items")
			Param("page", String, "Paging in the format <start>,<limit>")
		})
		Response(OK, func() {
			Media(CollectionOf(workItem))
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
		Response(Created, "/workitems/.*", func() {
			Media(workItem)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
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
			Media(workItem)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

})

var _ = Resource("workitemtype", func() {

	BasePath("/workitemtypes")

	Action("show", func() {

		Routing(
			GET("/:name"),
		)
		Description("Retrieve work item type with given name.")
		Params(func() {
			Param("name", String, "name")
		})
		Response(OK, func() {
			Media(workItemType)
		})
		Response(NotFound)
	})

	Action("create", func() {
		Routing(
			POST(""),
		)
		Description("Create work item type.")
		Payload(CreateWorkItemTypePayload)
		Response(Created, "/workitemtypes/.*", func() {
			Media(workItemType)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
	})

	Action("list", func() {
		Routing(
			GET(""),
		)
		Description("List work item types.")
		Params(func() {
			Param("page", String, "Paging in the format <start>,<limit>")
		})
		Response(OK, func() {
			Media(CollectionOf(workItemType))
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
	})
})

var _ = Resource("user", func() {
	BasePath("/user")

	Action("show", func() {
		Security("jwt")
		Routing(
			GET(""),
		)
		Description("Get the authenticated user")
		Response(OK, func() {
			Media(User)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(Unauthorized)
	})

})

var _ = Resource("status", func() {

	DefaultMedia(ALMStatus)
	BasePath("/status")

	Action("show", func() {
		Routing(
			GET(""),
		)
		Description("Show the status of the current running instance")
		Response(OK)
		Response(ServiceUnavailable, ALMStatus)
	})
})

var _ = Resource("login", func() {

	BasePath("/login")

	Action("authorize", func() {
		Routing(
			GET("authorize"),
		)
		Description("Authorize with the ALM")
		Response(Unauthorized)
		Response(TemporaryRedirect)
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

var _ = Resource("tracker", func() {
	BasePath("/trackers")

	Action("list", func() {
		Routing(
			GET(""),
		)
		Description("List all tracker configurations.")
		Params(func() {
			Param("filter", String, "a query language expression restricting the set of found items")
			Param("page", String, "Paging in the format <start>,<limit>")
		})
		Response(OK, func() {
			Media(CollectionOf(Tracker))
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

	Action("show", func() {
		Routing(
			GET("/:id"),
		)
		Description("Retrieve tracker configuration for the given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK, func() {
			Media(Tracker)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

	Action("create", func() {
		Routing(
			POST(""),
		)
		Description("Add new tracker configuration.")
		Payload(CreateTrackerAlternatePayload)
		Response(Created, "/trackers/.*", func() {
			Media(Tracker)
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
		Description("Delete tracker configuration.")
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
		Description("Update tracker configuration.")
		Payload(UpdateTrackerAlternatePayload)
		Response(OK, func() {
			Media(Tracker)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

})

var _ = Resource("trackerquery", func() {
	BasePath("/trackerqueries")
	Action("show", func() {
		Routing(
			GET("/:id"),
		)
		Description("Retrieve tracker configuration for the given id.")
		Params(func() {
			Param("id", String, "id")
		})
		Response(OK, func() {
			Media(TrackerQuery)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

	Action("create", func() {
		Routing(
			POST(""),
		)
		Description("Add new tracker query.")
		Payload(CreateTrackerQueryAlternatePayload)
		Response(Created, "/trackerqueries/.*", func() {
			Media(TrackerQuery)
		})
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
		Description("Update tracker query.")
		Payload(UpdateTrackerQueryAlternatePayload)
		Response(OK, func() {
			Media(TrackerQuery)
		})
		Response(BadRequest, func() {
			Media(ErrorMedia)
		})
		Response(InternalServerError)
		Response(NotFound)
	})

})
