package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// CSVList creates a UserTypeDefinition
func CSVList(name, description string, data *d.UserTypeDefinition) *d.MediaTypeDefinition {
	return a.MediaType("text/csv", func() {
	})
}
