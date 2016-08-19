package migration

import (
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
)

// Perform executes the required migration of the database on startup
func Perform(db *gorm.DB) error {
	db.AutoMigrate(
		&models.WorkItem{},
		&models.WorkItemType{})
	if db.Error != nil {
		return db.Error
	}

	return db.Error
}
