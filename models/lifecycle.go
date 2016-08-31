package models

import (
	"time"
)

// The Lifecycle struct contains all the items from gorm.Model except the ID field,
// hence we can embed the Lifecycle struct into Models that needs soft delete and alike.
type Lifecycle struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
