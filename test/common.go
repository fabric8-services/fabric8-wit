// This file was generated by counterfeiter
package test

import (
	uuid "github.com/satori/go.uuid"
)

const maxValidNameLength = 62

// CreateRandomValidTestName functions creates a valid lenght name
func CreateRandomValidTestName(name string) string {
	randomName := name + uuid.NewV4().String()
	if len(randomName) > 62 {
		return randomName[:61]
	}
	return randomName
}
