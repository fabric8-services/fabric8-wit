//************************************************************************//
// API "alm": Application Resource Href Factories
//
// Generated with goagen v0.0.1, command line:
// $ goagen.exe
// --design=github.com/ALMighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\ALMighty\almighty-core
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import "fmt"

// VersionHref returns the resource href.
func VersionHref() string {
	return fmt.Sprintf("/api/version")
}
