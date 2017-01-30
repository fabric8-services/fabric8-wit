package area

import (
	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

const APIStringTypeAreas = "areas"

// Area describes a single Area
type Area struct {
	gormsupport.Lifecycle
	ID      uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID uuid.UUID `sql:"type:uuid"`
	Path    string
	Name    string
	Version int
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *Area) TableName() string {
	return "areas"
}

// Repository describes interactions with Areas
type Repository interface {
	Create(ctx context.Context, u *Area) error
	List(ctx context.Context, spaceID uuid.UUID) ([]*Area, error)
	Load(ctx context.Context, id uuid.UUID) (*Area, error)
}

// NewAreaRepository creates a new storage type.
func NewAreaRepository(db *gorm.DB) Repository {
	return &GormAreaRepository{db: db}
}

// GormAreaRepository is the implementation of the storage interface for Areas.
type GormAreaRepository struct {
	db *gorm.DB
}

// Create creates a new record.
func (m *GormAreaRepository) Create(ctx context.Context, u *Area) error {
	defer goa.MeasureSince([]string{"goa", "db", "area", "create"}, time.Now())

	u.ID = uuid.NewV4()

	err := m.db.Create(u).Error
	if err != nil {
		goa.LogError(ctx, "error adding Area", "error", err.Error())
		return err
	}

	return nil
}

// List all Areas related to a single item
func (m *GormAreaRepository) List(ctx context.Context, spaceID uuid.UUID) ([]*Area, error) {
	defer goa.MeasureSince([]string{"goa", "db", "Area", "query"}, time.Now())
	var objs []*Area

	err := m.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// Load a single Area regardless of parent
func (m *GormAreaRepository) Load(ctx context.Context, id uuid.UUID) (*Area, error) {
	defer goa.MeasureSince([]string{"goa", "db", "Area", "get"}, time.Now())
	var obj Area

	tx := m.db.Where("id = ?", id).First(&obj)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("Area", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &obj, nil
}
