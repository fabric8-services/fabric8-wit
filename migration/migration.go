package migration

import (
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"
)

// Perform executes the required migration of the database on startup
func Perform(db *gorm.DB) {

	db.AutoMigrate(
		&models.WorkItem{})
}
