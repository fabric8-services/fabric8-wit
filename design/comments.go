package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var comment = a.Type("Comment", func() {
	a.Description(`JSONAPI store for the data of a comment.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("comments")
	})
	a.Attribute("id", d.UUID, "ID of comment", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", commentAttributes)
	a.Attribute("relationships", commentRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var createComment = a.Type("CreateComment", func() {
	a.Description(`JSONAPI store for the data of a comment.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("comments")
	})
	a.Attribute("attributes", createCommentAttributes)
	a.Required("type", "attributes")
})

var commentAttributes = a.Type("CommentAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a comment. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("created-at", d.DateTime, "When the comment was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("body", d.String, "The comment body", func() {
		a.Example("This is really interesting")
	})
})

var createCommentAttributes = a.Type("CreateCommentAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" for creating a comment. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("body", d.String, "The comment body", func() {
		a.MinLength(1) // Empty comment not allowed
		a.Example("This is really interesting")
	})
	a.Required("body")
})

var commentRelationships = a.Type("CommentRelations", func() {
	a.Attribute("created-by", commentCreatedBy, "This defines the created by relation")
	a.Attribute("parent", relationGeneric, "This defines the owning resource of the comment")
})

var commentCreatedBy = a.Type("CommentCreatedBy", func() {
	a.Attribute("data", identityRelationData)
	a.Required("data")
})

var identityRelationData = a.Type("IdentityRelationData", func() {
	a.Attribute("id", d.UUID, "unique id for the user identity")
	a.Attribute("type", d.String, "type of the user identity", func() {
		a.Enum("identities")
	})
	a.Required("type")
})

var commentArray = a.MediaType("application/vnd.comments+json", func() {
	a.TypeName("CommentArray")
	a.Description("Holds the response of comments")
	a.Attribute("meta", a.HashOf(d.String, d.Any))
	a.Attribute("data", a.ArrayOf(comment))

	a.Required("data")

	a.View("default", func() {
		a.Attribute("data")
		a.Attribute("meta")
	})
})

var commentSingle = a.MediaType("application/vnd.comment+json", func() {
	a.TypeName("CommentSingle")
	a.Description("Holds the response of a single comment")
	a.Attribute("data", comment)

	a.Required("data")

	a.View("default", func() {
		a.Attribute("data")
	})
})

var createSingleComment = a.MediaType("application/vnd.comments-create+json", func() {
	a.TypeName("CreateSingleComment")
	a.Description("Holds the create data for a comment")
	a.Attribute("data", createComment)

	a.Required("data")

	a.View("default", func() {
		a.Attribute("data")
	})
})

var _ = a.Resource("comments", func() {
	a.BasePath("/comments")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Description("Retrieve comment with given id.")
		a.Response(d.OK, func() {
			a.Media(commentSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})

var _ = a.Resource("work-item-comments", func() {
	a.Parent("workitem")

	a.Action("list", func() {
		a.Routing(
			a.GET("comments"),
		)
		a.Description("List comments associated with the given work item")
		a.Response(d.OK, func() {
			a.Media(commentArray)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("relations", func() {
		a.Routing(
			a.GET("relationships/comments"),
		)
		a.Description("List comments associated with the given work item")
		a.Response(d.OK, func() {
			a.Media(commentArray)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("comments"),
		)
		a.Description("List comments associated with the given work item")
		a.Response(d.OK, func() {
			a.Media(commentSingle)
		})
		a.Payload(createSingleComment)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
