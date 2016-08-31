package migration

import (
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

// Perform executes the required migration of the database on startup
func Perform(ctx context.Context, db *gorm.DB, witr models.WorkItemTypeRepository) error {

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")

	db.AutoMigrate(
		&models.WorkItem{},
		&models.WorkItemType{},
		&remoteworkitem.Tracker{},
		&remoteworkitem.TrackerQuery{},
		&remoteworkitem.TrackerItem{},
		&account.Identity{},
		&account.User{})

	if db.Error != nil {
		return db.Error
	}

	// FIXME: Need to add this conditionally
	// q := `ALTER TABLE "tracker_queries" ADD CONSTRAINT "tracker_fk" FOREIGN KEY ("tracker") REFERENCES "trackers" ON DELETE CASCADE`
	// db.Exec(q)

	if err := createSystemUserstory(ctx, witr); err != nil {
		return err
	}
	if err := createSystemValueProposition(ctx, witr); err != nil {
		return err
	}
	if err := createSystemFundamental(ctx, witr); err != nil {
		return err
	}
	if err := createSystemExperience(ctx, witr); err != nil {
		return err
	}
	if err := createSystemFeature(ctx, witr); err != nil {
		return err
	}
	if err := createSystemBug(ctx, witr); err != nil {
		return err
	}
	return nil
}

func createSystemUserstory(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.userstory", ctx, witr)
}

func createSystemValueProposition(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.valueproposition", ctx, witr)
}

func createSystemFundamental(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.fundamental", ctx, witr)
}

func createSystemExperience(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.experience", ctx, witr)
}

func createSystemFeature(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.feature", ctx, witr)
}

func createSystemBug(ctx context.Context, witr models.WorkItemTypeRepository) error {
	return createCommon("system.bug", ctx, witr)
}

func createCommon(typeName string, ctx context.Context, witr models.WorkItemTypeRepository) error {
	_, err := witr.Load(ctx, typeName)
	switch err.(type) {
	case models.NotFoundError:
		stString := "string"
		_, err := witr.Create(ctx, nil, typeName, map[string]app.FieldDefinition{
			"system.title":       app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.description": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
			"system.creator":     app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: true},
			"system.assignee":    app.FieldDefinition{Type: &app.FieldType{Kind: "user"}, Required: false},
			"system.state": app.FieldDefinition{
				Type: &app.FieldType{
					BaseType: &stString,
					Kind:     "enum",
					Values:   []interface{}{"new", "in progress", "resolved", "closed"},
				},
				Required: true,
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}
