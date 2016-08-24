package migration

import (
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
)

// Perform executes the required migration of the database on startup
func Perform(db *gorm.DB) {

	db.AutoMigrate(
		&models.WorkItem{},
		&models.Tracker{},
		&models.TrackerQuery{},
		&models.TrackerItem{})
}
