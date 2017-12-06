package label

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
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
	BorderColor     string `sql:"DEFAULT:#000000"`
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
	IsValid(ctx context.Context, id uuid.UUID) bool
	Load(ctx context.Context, labelID uuid.UUID) (*Label, error)
	Save(ctx context.Context, lbl Label) (*Label, error)
}

// NewLabelRepository creates a new storage type.
func NewLabelRepository(db *gorm.DB) Repository {
	return &GormLabelRepository{db: db}
}

// GormLabelRepository is the implementation of the storage interface for Labels.
type GormLabelRepository struct {
	db *gorm.DB
}

// LabelTableName constant that holds table name of Labels
const LabelTableName = "labels"

// Create a new label
func (m *GormLabelRepository) Create(ctx context.Context, u *Label) error {
	defer goa.MeasureSince([]string{"goa", "db", "label", "create"}, time.Now())
	u.ID = uuid.NewV4()
	if strings.TrimSpace(u.Name) == "" {
		return errors.NewBadParameterError("label name cannot be empty string", u.Name).Expected("non empty string")
	}
	err := m.db.Create(u).Error
	if err != nil {
		// combination of name and space ID should be unique
		if gormsupport.IsUniqueViolation(err, "labels_name_space_id_unique_idx") {
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

// Save update the given label
func (m *GormLabelRepository) Save(ctx context.Context, l Label) (*Label, error) {
	defer goa.MeasureSince([]string{"goa", "db", "label", "save"}, time.Now())
	if strings.TrimSpace(l.Name) == "" {
		return nil, errors.NewBadParameterError("label name cannot be empty string", l.Name).Expected("non empty string")
	}
	lbl := Label{}
	tx := m.db.Where("id = ?", l.ID).First(&lbl)
	oldVersion := l.Version
	l.Version = lbl.Version + 1
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"label_id": l.ID,
		}, "label cannot be found")
		return nil, errors.NewNotFoundError("label", l.ID.String())
	}
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"label_id": l.ID,
			"err":      err,
		}, "unknown error happened when searching the label")
		return nil, errors.NewInternalError(ctx, err)
	}
	tx = tx.Where("Version = ?", oldVersion).Save(&l)
	if err := tx.Error; err != nil {
		// combination of name and space ID should be unique
		if gormsupport.IsUniqueViolation(err, "labels_name_space_id_unique_idx") {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"name":     l.Name,
				"space_id": l.SpaceID,
			}, "unable to create label because a label with same already exists in the space")
			return nil, errors.NewDataConflictError(fmt.Sprintf("label already exists with name = %s , space_id = %s", l.Name, l.SpaceID.String()))
		}
		log.Error(ctx, map[string]interface{}{
			"label_id": l.ID,
			"err":      err,
		}, "unable to save the label")
		return nil, errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	log.Debug(ctx, map[string]interface{}{
		"label_id": l.ID,
	}, "label updated successfully")

	return &l, nil
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

// IsValid returns true if the identity exists
func (m *GormLabelRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	return repository.CheckExists(ctx, m.db, LabelTableName, id.String()) == nil
}

// Load label in a space
func (m *GormLabelRepository) Load(ctx context.Context, ID uuid.UUID) (*Label, error) {
	defer goa.MeasureSince([]string{"goa", "db", "label", "show"}, time.Now())
	lbl := Label{}
	tx := m.db.Where("id = ?", ID).First(&lbl)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"label_id": ID.String(),
		}, "state or known referer was empty")
		return nil, errors.NewNotFoundError("label", ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      tx.Error,
			"label_id": ID.String(),
		}, "unable to load the label by ID")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &lbl, nil
}
