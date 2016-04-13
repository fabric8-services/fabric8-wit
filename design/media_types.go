package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

// ALMVersion defines therunning ALM Version MediaType
var ALMVersion = MediaType("application/vnd.version+json", func() {
	Description("The current running version")
	Attributes(func() {
		Attribute("commit", String, "Commit SHA this build is based on")
		Attribute("build_time", String, "The date when build")
		Required("commit", "build_time")
	})
	View("default", func() {
		Attribute("commit")
		Attribute("build_time")
	})
})

// AuthToken represents an authentication JWT Token
var AuthToken = MediaType("application/vnd.authtoken+json", func() {
	TypeName("AuthToken")
	Description("JWT Token")
	Attributes(func() {
		Attribute("token", String, "JWT Token")
		Required("token")
	})
	View("default", func() {
		Attribute("token")
	})
})
