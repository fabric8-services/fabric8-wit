package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("login", func() {

	a.BasePath("/login")

	a.Action("authorize", func() {
		a.Routing(
			a.GET("authorize"),
		)
		a.Description("Authorize with the WIT")
		a.Response(d.TemporaryRedirect)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("generate", func() {
		a.Routing(
			a.GET("generate"),
		)
		a.Description("Generate a set of Tokens for different Auth levels. NOT FOR PRODUCTION. Only available if server is running in dev mode")
		a.Response(d.OK, func() {
			a.Media(a.CollectionOf(AuthToken))
		})
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("refresh", func() {
		a.Routing(
			a.POST("refresh"),
		)
		a.Description("Refresh access token")
		a.Response(d.OK)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("link", func() {
		a.Routing(
			a.GET("/link"),
		)
		a.Description("Link an Identity Provider account to the user account")
		a.Response(d.TemporaryRedirect)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("linksession", func() {
		a.Routing(
			a.GET("/linksession"),
		)
		a.Description("Link an Identity Provider account to the user account represented by user's session. This endpoint is to be used for auto linking during login.")
		a.Response(d.TemporaryRedirect)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("logout", func() {

	a.BasePath("/logout")

	a.Action("logout", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Logout user")
		a.Response(d.TemporaryRedirect)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

// AuthToken represents an authentication JWT Token
var AuthToken = a.MediaType("application/vnd.authtoken+json", func() {
	a.TypeName("AuthToken")
	a.Description("JWT Token")
	a.Attributes(func() {
		a.Attribute("token", tokenData)
		a.Required("token")
	})
	a.View("default", func() {
		a.Attribute("token")
	})
})

var tokenData = a.Type("TokenData", func() {
	a.Attribute("access_token", d.String, "Access token")
	a.Attribute("expires_in", d.Any, "Access token expires in seconds")
	a.Attribute("refresh_expires_in", d.Any, "Refresh token expires in seconds")
	a.Attribute("refresh_token", d.String, "Refresh token")
	a.Attribute("token_type", d.String, "Token type")
	a.Attribute("not-before-policy", d.Any, "Token is not valid if issued before this date")
	a.Required("expires_in")
	a.Required("refresh_expires_in")
	a.Required("not-before-policy")
})
