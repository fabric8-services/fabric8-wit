//************************************************************************//
// API "alm": Application Resource Href Factories
//
// Generated with goagen v0.0.1, command line:
// $ goagen
// --out=$(GOPATH)/src/github.com/almighty/almighty-design
// --design=github.com/almighty/almighty-design/design
// --pkg=app
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import "fmt"

// VersionHref returns the resource href.
func VersionHref() string {
	return fmt.Sprintf("/api/version")
}
