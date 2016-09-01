package migration

import (
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/jinzhu/gorm"
)

// Perform executes the required migration of the database on startup
func Perform(db *gorm.DB) {

	db.AutoMigrate(
		&models.WorkItem{},
		&remoteworkitem.Tracker{},
		&remoteworkitem.TrackerQuery{},
		&remoteworkitem.TrackerItem{})
	q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker") REFERENCES "trackers" ON DELETE CASCADE`
	db.Exec(q)
}
