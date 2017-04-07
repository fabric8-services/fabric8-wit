package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var updateUser = a.MediaType("application/vnd.updateuser+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("UpdateUser")
	a.Description("ALM User Update")
	a.Attributes(func() {
		a.Attribute("data", updateUserData)
		a.Required("data")

	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

// identityData represents an identified user object
var updateUserData = a.Type("UpdateUserData", func() {
	a.Attribute("type", d.String, "type of the user identity")
	a.Attribute("attributes", userDataAttributes, "Attributes of the user identity")
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

// user represents an identified user object
var user = a.MediaType("application/vnd.user+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("User")
	a.Description("ALM User Identity")
	a.Attributes(func() {
		a.Attribute("data", userData)
		a.Required("data")

	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

// identityArray represents an array of identified user objects
var identityArray = a.MediaType("application/vnd.identity-array+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("IdentityArray")
	a.Description("ALM User Identity Array")
	a.Attributes(func() {
		a.Attribute("data", a.ArrayOf(identityData))
		a.Required("data")

	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

// userArray represents an array of user objects
// Deprecated. Use userList instead
var userArray = a.MediaType("application/vnd.user-array+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("UserArray")
	a.Description("User Array")
	a.Attributes(func() {
		a.Attribute("data", a.ArrayOf(userData))
		a.Required("data")

	})
	a.View("default", func() {
		a.Attribute("data")
		a.Required("data")
	})
})

var userListMeta = a.Type("UserListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var userList = JSONList(
	"User", "Holds the paginated response to a user list request",
	userData,
	pagingLinks,
	userListMeta)

var _ = a.Resource("user", func() {
	a.BasePath("/user")

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("Get the authenticated user")
		a.UseTrait("conditional")
		a.Response(d.OK, user)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})

var _ = a.Resource("identity", func() {
	a.BasePath("/identities")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all identities.")
		a.Response(d.OK, func() {
			a.Media(identityArray)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("users", func() {
	a.BasePath("/users")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve user for the given ID.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, user)
		a.Response(d.NotModified)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
	})

	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH(""),
		)
		a.Description("update the authenticated user")
		a.Payload(updateUser)
		a.Response(d.OK, func() {
			a.Media(user)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List all users.")
		a.UseTrait("conditional")
		a.Response(d.OK, userArray)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

// userData represents an identified user object
var userData = a.Type("UserData", func() {
	a.Attribute("id", d.String, "unique id for the user")
	a.Attribute("type", d.String, "type of the user")
	a.Attribute("attributes", userDataAttributes, "Attributes of the user")
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

// userDataAttributes represents an identified user object attributes
var userDataAttributes = a.Type("UserDataAttributes", func() {
	a.Attribute("created-at", d.DateTime, "The date of creation of the user")
	a.Attribute("updated-at", d.DateTime, "The date of update of the user")
	a.Attribute("fullName", d.String, "The user's full name")
	a.Attribute("imageURL", d.String, "The avatar image for the user")
	a.Attribute("username", d.String, "The username")
	a.Attribute("email", d.String, "The email")
	a.Attribute("bio", d.String, "The bio")
	a.Attribute("url", d.String, "The url")
	a.Attribute("userID", d.String, "The id of the corresponding User")
	a.Attribute("identityID", d.String, "The id of the corresponding Identity")
	a.Attribute("providerType", d.String, "The type of provider for the corresponding identity")
	a.Attribute("contextInformation", a.HashOf(d.String, d.Any), "User context information of any type as a json", func() {
		a.Example(map[string]interface{}{"last_visited_url": "https://a.openshift.io", "space": "3d6dab8d-f204-42e8-ab29-cdb1c93130ad"})
	})
})

// identityData represents an identified identity object
var identityData = a.Type("IdentityData", func() {
	a.Attribute("id", d.String, "unique id for the user identity")
	a.Attribute("type", d.String, "type of the user identity")
	a.Attribute("attributes", identityDataAttributes, "Attributes of the user identity")
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

// identityDataAttributes represents an identified identity object attributes
var identityDataAttributes = a.Type("IdentityDataAttributes", func() {
	a.Attribute("created-at", d.DateTime, "The date of creation of the user")
	a.Attribute("updated-at", d.DateTime, "The date of update of the user")
	a.Attribute("username", d.String, "The username")
	a.Attribute("providerType", d.String, "The IDP provided this identity")
})
