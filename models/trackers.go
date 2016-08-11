package models

import ()

type Trackers struct {
	ID uint64 `gorm:"primary_key"`
	// Id of the type of this work item
	Type string
	// Validation Information
	Credentials string
	// URL of the issue tracker instance
	URL string
}
