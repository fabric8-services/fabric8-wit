package iteration

import (
	"log"
	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Defines "type" string to be used while validating jsonapi spec based payload
const (
	APIStringTypeIteration = "iterations"
)

// Iteration describes a single iteration
type Iteration struct {
	gormsupport.Lifecycle
	ID          uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID     uuid.UUID `sql:"type:uuid"`
	ParentID    uuid.UUID `sql:"type:uuid"` // TODO: This should be * to support nil ?
	StartAt     *time.Time
	EndAt       *time.Time
	Name        string
	Description string
	Version     int
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
	Save(ctx context.Context, i Iteration) (*Iteration, error)
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

// Save updates the given iteration in the db. Version must be the same as the one in the stored version
// returns NotFoundError, VersionConflictError or InternalError
func (m *GormIterationRepository) Save(ctx context.Context, i Iteration) (*Iteration, error) {
	itr := Iteration{}
	tx := m.db.Where("id=?", i.ID).First(&itr)
	oldVersion := i.Version
	i.Version++
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("iteration", i.ID.String())
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	tx = tx.Where("Version = ?", oldVersion).Save(&i)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	log.Printf("updated iteration to %v\n", i)
	return &i, nil
}
