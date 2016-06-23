//************************************************************************//
// API "alm": Application User Types
//
// Generated with goagen v0.2.dev, command line:
// $ goagen.exe
// --design=github.com/almighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\almighty\almighty-core
// --version=v0.2.dev
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import "github.com/goadesign/goa"

// field user type.
type field struct {
	Name *string `json:"name,omitempty" xml:"name,omitempty" form:"name,omitempty"`
	Type *string `json:"type,omitempty" xml:"type,omitempty" form:"type,omitempty"`
}

// Validate validates the field type instance.
func (ut *field) Validate() (err error) {
	if ut.Name == nil {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "name"))
	}
	if ut.Type == nil {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "type"))
	}

	return
}

// Publicize creates Field from field
func (ut *field) Publicize() *Field {
	var pub Field
	if ut.Name != nil {
		pub.Name = *ut.Name
	}
	if ut.Type != nil {
		pub.Type = *ut.Type
	}
	return &pub
}

// Field user type.
type Field struct {
	Name string `json:"name" xml:"name" form:"name"`
	Type string `json:"type" xml:"type" form:"type"`
}

// Validate validates the Field type instance.
func (ut *Field) Validate() (err error) {
	if ut.Name == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "name"))
	}
	if ut.Type == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "type"))
	}

	return
}
