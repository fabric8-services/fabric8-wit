package label

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"

	"fmt"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// APIStringTypeLabels helps to avoid string literal
const APIStringTypeLabels = "labels"

// Label describes a single Label
type Label struct {
	gormsupport.Lifecycle
	ID              uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID         uuid.UUID `sql:"type:uuid"`
	Name            string
	TextColor       string `sql:"DEFAULT:#000000"`
	BackgroundColor string `sql:"DEFAULT:#FFFFFF"`
	Version         int
}

// GetETagData returns the field values to use to generate the ETag
func (m Label) GetETagData() []interface{} {
	return []interface{}{m.ID, m.Version}
}

// GetLastModified returns the last modification time
func (m Label) GetLastModified() time.Time {
	return m.UpdatedAt.Truncate(time.Second)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m Label) TableName() string {
	return "labels"
}

// Repository describes interactions with Labels
type Repository interface {
	Create(ctx context.Context, u *Label) error
	List(ctx context.Context, spaceID uuid.UUID) ([]Label, error)
	Load(ctx context.Context, spaceID uuid.UUID, labelID uuid.UUID) (*Label, error)
}

// NewLabelRepository creates a new storage type.
func NewLabelRepository(db *gorm.DB) Repository {
	return &GormLabelRepository{db: db}
}

// GormLabelRepository is the implementation of the storage interface for Labels.
type GormLabelRepository struct {
	db *gorm.DB
}

// Create a new label
func (m *GormLabelRepository) Create(ctx context.Context, u *Label) error {
	defer goa.MeasureSince([]string{"goa", "db", "label", "create"}, time.Now())
	u.ID = uuid.NewV4()
	err := m.db.Create(u).Error
	if err != nil {
		// combination of name and space ID should be unique
		if gormsupport.IsUniqueViolation(err, "labels_name_space_id_unique") {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"name":     u.Name,
				"space_id": u.SpaceID,
			}, "unable to create label because a label with same already exists in the space")
			return errors.NewDataConflictError(fmt.Sprintf("label already exists with name = %s , space_id = %s", u.Name, u.SpaceID.String()))
		}
		log.Error(ctx, map[string]interface{}{}, "error adding Label: %s", err.Error())
		return err
	}
	return nil
}

// List all labels in a space
func (m *GormLabelRepository) List(ctx context.Context, spaceID uuid.UUID) ([]Label, error) {
	defer goa.MeasureSince([]string{"goa", "db", "label", "query"}, time.Now())
	var objs []Label
	err := m.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// Load label in a space
func (m *GormLabelRepository) Load(ctx context.Context, spaceID uuid.UUID, labelID uuid.UUID) (*Label, error) {
	defer goa.MeasureSince([]string{"goa", "db", "label", "show"}, time.Now())
	var lbl Label
	err := m.db.Where("space_id = ? and id = ?", spaceID, labelID).Find(&lbl).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return &lbl, nil
}
