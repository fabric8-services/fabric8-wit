//************************************************************************//
// API "alm": Application Media Types
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

// AuthToken media type.
//
// Identifier: application/vnd.authtoken+json
type AuthToken struct {
	// JWT Token
	Token string `json:"token" xml:"token" form:"token"`
}

// Validate validates the AuthToken media type instance.
func (mt *AuthToken) Validate() (err error) {
	if mt.Token == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "token"))
	}

	return
}

// AuthTokenCollection media type is a collection of AuthToken.
//
// Identifier: application/vnd.authtoken+json; type=collection
type AuthTokenCollection []*AuthToken

// Validate validates the AuthTokenCollection media type instance.
func (mt AuthTokenCollection) Validate() (err error) {
	for _, e := range mt {
		if e.Token == "" {
			err = goa.MergeErrors(err, goa.MissingAttributeError(`response[*]`, "token"))
		}

	}
	return
}

// Version media type.
//
// Identifier: application/vnd.version+json
type Version struct {
	// The date when build
	BuildTime string `json:"build_time" xml:"build_time" form:"build_time"`
	// Commit SHA this build is based on
	Commit string `json:"commit" xml:"commit" form:"commit"`
}

// Validate validates the Version media type instance.
func (mt *Version) Validate() (err error) {
	if mt.Commit == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "commit"))
	}
	if mt.BuildTime == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "build_time"))
	}

	return
}

// WorkItem media type.
//
// Identifier: application/vnd.workitem+json
type WorkItem struct {
	Fields map[string]interface{} `json:"fields" xml:"fields" form:"fields"`
	// unique id per installation
	ID string `json:"id" xml:"id" form:"id"`
	// User Readable Name of this item
	Name string `json:"name" xml:"name" form:"name"`
	// Id of the type of this work item
	Type string `json:"type" xml:"type" form:"type"`
	// Version for optimistic concurrency control
	Version int `json:"version" xml:"version" form:"version"`
}

// Validate validates the WorkItem media type instance.
func (mt *WorkItem) Validate() (err error) {
	if mt.ID == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "id"))
	}
	if mt.Name == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "name"))
	}
	if mt.Type == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "type"))
	}
	if mt.Fields == nil {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "fields"))
	}

	return
}

// WorkItemType media type.
//
// Identifier: application/vnd.workitemtype+json
type WorkItemType struct {
	Fields []*Field `json:"fields" xml:"fields" form:"fields"`
	// unique id per installation
	ID string `json:"id" xml:"id" form:"id"`
	// User Readable Name of this item
	Name string `json:"name" xml:"name" form:"name"`
	// Version for optimistic concurrency control
	Version int `json:"version" xml:"version" form:"version"`
}

// Validate validates the WorkItemType media type instance.
func (mt *WorkItemType) Validate() (err error) {
	if mt.ID == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "id"))
	}
	if mt.Name == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "name"))
	}
	if mt.Fields == nil {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "fields"))
	}

	for _, e := range mt.Fields {
		if e.Name == "" {
			err = goa.MergeErrors(err, goa.MissingAttributeError(`response.fields[*]`, "name"))
		}
		if e.Type == "" {
			err = goa.MergeErrors(err, goa.MissingAttributeError(`response.fields[*]`, "type"))
		}

	}
	return
}

// WorkItemTypeLink media type.
//
// Identifier: application/vnd.workitemtype+json
type WorkItemTypeLink struct {
	// unique id per installation
	ID string `json:"id" xml:"id" form:"id"`
}

// Validate validates the WorkItemTypeLink media type instance.
func (mt *WorkItemTypeLink) Validate() (err error) {
	if mt.ID == "" {
		err = goa.MergeErrors(err, goa.MissingAttributeError(`response`, "id"))
	}

	return
}
