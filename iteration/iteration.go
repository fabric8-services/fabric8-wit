package iteration

import (
	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Iteration describes a single iteration
type Iteration struct {
	gormsupport.Lifecycle
	ID       uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID  uuid.UUID `sql:"type:uuid"`
	ParentID uuid.UUID `sql:"type:uuid"` // TODO: This should be * to support nil ?
	StartAt  *time.Time
	EndAt    *time.Time
	Name     string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *Iteration) TableName() string {
	return "iterations"
}

// Repository describes interactions with Iterations
type Repository interface {
	Create(ctx context.Context, u *Iteration) error
	List(ctx context.Context, spaceID uuid.UUID) ([]*Iteration, error)
	Load(ctx context.Context, id uuid.UUID) (*Iteration, error)
}

// NewIterationRepository creates a new storage type.
func NewIterationRepository(db *gorm.DB) Repository {
	return &GormIterationRepository{db: db}
}

// GormIterationRepository is the implementation of the storage interface for Iterations.
type GormIterationRepository struct {
	db *gorm.DB
}

// Create creates a new record.
func (m *GormIterationRepository) Create(ctx context.Context, u *Iteration) error {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "create"}, time.Now())

	u.ID = uuid.NewV4()

	err := m.db.Create(u).Error
	if err != nil {
		goa.LogError(ctx, "error adding Iteration", "error", err.Error())
		return err
	}

	return nil
}

// List all Iterations related to a single item
func (m *GormIterationRepository) List(ctx context.Context, spaceID uuid.UUID) ([]*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "query"}, time.Now())
	var objs []*Iteration

	err := m.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// Load a single Iteration regardless of parent
func (m *GormIterationRepository) Load(ctx context.Context, id uuid.UUID) (*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "get"}, time.Now())
	var obj Iteration

	tx := m.db.Where("id = ?", id).First(&obj)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("Iteration", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &obj, nil
}
