package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var elementAccess = a.Type("ElementAccess", func() {
	a.Description(`defines access rights for an element in an object`)
	a.Attribute("name", d.String)
	a.Attribute("permissions", d.String, func() {
		a.Enum("NA", "WRITE")
	})
	a.Required("name", "permissions")
})

var internalAccess = a.Type("InternalAccess", func() {
	a.Description(`defines access rights for all elements of an object`)
	a.Attribute("access", a.ArrayOf(elementAccess))
	a.Required("access")
})

var endpointAccess = a.Type("EndpointAccess", func() {
	a.Description(`defines access rights for HTTP methods on an endpoint`)
	a.Attribute("methods", a.ArrayOf(d.String, func() {
		a.Enum("GET", "PUT", "PATCH", "DELETE")
	}))
	a.Required("methods")
})

var linkWithAccess = a.Type("LinkWithAccess", func() {
	a.Attribute("href", d.String, "URL for the link")
	a.Attribute("meta", endpointAccess)
})
