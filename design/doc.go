// Package design contains the generic API machinery code of our adder API generated
// using goa framework. This generated API accepts HTTP GET/POST/PATCH/PUT/DELETE requests
// from multiple clients.
package design

import "strings"

func mandatoryOnCreate(desc string) string {
	if !strings.HasSuffix(strings.TrimSpace(desc), ".") {
		desc += ". "
	}
	return desc + "\n This is MANDATORY on creation of resource."
}

func mandatoryOnUpdate(desc string) string {
	if !strings.HasSuffix(strings.TrimSpace(desc), ".") {
		desc += ". "
	}
	return desc + "\n This is MANDATORY on update of resource."
}
