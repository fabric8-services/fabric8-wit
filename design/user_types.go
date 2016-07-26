package design

import (
	. "github.com/goadesign/goa/design"
	. "github.com/goadesign/goa/design/apidsl"
)

var CreateWorkItemPayload = Type("CreateWorkItemPayload", func() {
	Attribute("type", String, "The type of the newly created work item")
	Attribute("name", String, "User Readable Name of this item")
	Attribute("fields", HashOf(String, Any), "The field values, must conform to the type")
	Required("type", "name", "fields")
})
