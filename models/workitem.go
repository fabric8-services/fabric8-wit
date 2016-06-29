package models

import (
)

type WorkItem struct {
	ID uint `gorm:"primary_key"`

	// User Readable Name of this item
	Name string 
	// Id of the type of this work item
	Type string 
	// Version for optimistic concurrency control
	Version int                    
	Fields  Fields `sql:"type:jsonb"`
}
