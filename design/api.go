package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

var _ = API("alm", func() {
	Title("ALMighty: One to rule them all")
	Description("The next big thing")
	Version("1.0")
	Host("almighty.io")
	Scheme("http")
	BasePath("/api")
	Consumes("application/json")
	Produces("application/json")

	License(func() {
		Name("Apache License Version 2.0")
		URL("http://www.apache.org/licenses/LICENSE-2.0")
	})
	Origin("/[.*almighty.io|localhost]/", func() {
		Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		Headers("X-Request-Id", "Content-Type", "Authorization")
		MaxAge(600)
		Credentials()
	})

	JWTSecurity("jwt", func() {
		Description("JWT Token Auth")
		TokenURL("/api/login/authorize")
		Header("Authorization")
	})

	ResponseTemplate(Created, func(pattern string) {
		Description("Resource created")
		Status(201)
		Headers(func() {
			Header("Location", String, "href to created resource", func() {
				Pattern(pattern)
			})
		})
	})
})
