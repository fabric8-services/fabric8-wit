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
		a.Params(func() {
			a.Param("link", d.Boolean, "If true then link all available Identity Providers to the user account after successful login")
		})
		a.Description("Authorize with the ALM")
		a.Response(d.Unauthorized, JSONAPIErrors)
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
		a.Payload(refreshToken)
		a.Description("Refresh access token")
		a.Response(d.OK, func() {
			a.Media(AuthToken)
		})
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("link", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/link"),
		)
		a.Params(func() {
			a.Param("provider", d.String, "Identity Provider name to link to the user's account. If not set then link all available providers.")
			a.Param("redirect", d.String, "URL to be redirected to after successful account linking. If not set then will redirect to the referrer instead.")
		})
		a.Description("Link an Identity Provider account to the user account")
		a.Response(d.TemporaryRedirect)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("linksession", func() {
		a.Routing(
			a.GET("/linksession"),
		)
		a.Params(func() {
			a.Param("provider", d.String, "Identity Provider name to link to the user's account. If not set then link all available providers.")
			a.Param("redirect", d.String, "URL to be redirected to after successful account linking. If not set then will redirect to the referrer instead.")
			a.Param("sessionState", d.String, "Session state")
			a.Param("clientSession", d.String, "Client session ID")
		})
		a.Description("Link an Identity Provider account to the user account represented by user's session. This endpoint is to be used for auto linking during login.")
		a.Response(d.TemporaryRedirect)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("linkcallback", func() {
		a.Routing(
			a.GET("/linkcallback"),
		)
		a.Params(func() {
			a.Param("state", d.String, "State generated by the link request")
			a.Param("next", d.String, "Next provider to be linked. If not set then linking is complete.")
			a.Param("sessionState", d.String, "Session state")
			a.Param("clientSession", d.String, "Client session ID")
		})
		a.Description("Callback from Keyckloak when Identity Provider account successfully linked to the user account")
		a.Response(d.TemporaryRedirect)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var refreshToken = a.Type("RefreshToken", func() {
	a.Attribute("refresh_token", d.String, "Refresh token")
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
	a.Attribute("expires_in", d.Integer, "Access token expires in seconds")
	a.Attribute("refresh_expires_in", d.Integer, "Refresh token expires in seconds")
	a.Attribute("refresh_token", d.String, "Refresh token")
	a.Attribute("token_type", d.String, "Token type")
	a.Attribute("not-before-policy", d.Integer, "Token is not valid if issued before this date")
})
