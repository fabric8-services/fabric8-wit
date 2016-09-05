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
		&models.WorkItemType{})
	if db.Error != nil {
		return db.Error
	}

	if err := createSystemUserstory(ctx, witr); err != nil {
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
	_, err := witr.Load(ctx, "system.userstory")
	switch err.(type) {
	case models.NotFoundError:
		stString := "string"
		_, err := witr.Create(ctx, nil, "system.userstory", map[string]app.FieldDefinition{
			"system.title":       app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.description": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
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

func createSystemFeature(ctx context.Context, witr models.WorkItemTypeRepository) error {
	_, err := witr.Load(ctx, "system.feature")
	switch err.(type) {
	case models.NotFoundError:
		stString := "string"
		_, err := witr.Create(ctx, nil, "system.feature", map[string]app.FieldDefinition{
			"system.title":       app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.description": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
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

func createSystemBug(ctx context.Context, witr models.WorkItemTypeRepository) error {
	_, err := witr.Load(ctx, "system.bug")
	switch err.(type) {
	case models.NotFoundError:
		stString := "string"
		_, err := witr.Create(ctx, nil, "system.bug", map[string]app.FieldDefinition{
			"system.title":       app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: true},
			"system.description": app.FieldDefinition{Type: &app.FieldType{Kind: "string"}, Required: false},
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
