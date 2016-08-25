package migration

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

// Perform executes the required migration of the database on startup
func Perform(ctx context.Context, db *gorm.DB, witr models.WorkItemTypeRepository) error {
	db.AutoMigrate(
		&models.WorkItem{},
		&models.WorkItemType{},
		&models.Tracker{},
		&models.TrackerQuery{},
		&models.TrackerItem{})
	if db.Error != nil {
		return db.Error
	}

	_, err := witr.Load(ctx, "system.issue")
	switch err.(type) {
	case models.NotFoundError:
		_, err := witr.Create(ctx, nil, "system.issue", map[string]app.FieldDefinition{
			"system.title": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.owner": app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: true},
			"system.state": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
		})
		if err != nil {
			return err
		}
	}
	q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker_refer") REFERENCES "trackers" ON DELETE CASCADE`
	db.Exec(q)
	return db.Error
}
