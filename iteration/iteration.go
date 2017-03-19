package iteration

import (
	"time"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/path"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Defines "type" string to be used while validating jsonapi spec based payload
const (
	APIStringTypeIteration = "iterations"
	IterationStateNew      = "new"
	IterationStateStart    = "start"
	IterationStateClose    = "close"
	PathSepInService       = "/"
	PathSepInDatabase      = "."
)

// Iteration describes a single iteration
type Iteration struct {
	gormsupport.Lifecycle
	ID          uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	SpaceID     uuid.UUID `sql:"type:uuid"`
	Path        path.Path
	StartAt     *time.Time
	EndAt       *time.Time
	Name        string
	Description *string
	State       string // this tells if iteration is currently running or not
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
	CanStartIteration(ctx context.Context, i *Iteration) (bool, error)
	LoadMultiple(ctx context.Context, ids []uuid.UUID) ([]*Iteration, error)
	ListBacklogIterations(ctx context.Context, spaceID uuid.UUID, start *int, limit *int) ([]*Iteration, error)
}

// NewIterationRepository creates a new storage type.
func NewIterationRepository(db *gorm.DB) Repository {
	return &GormIterationRepository{db: db}
}

// GormIterationRepository is the implementation of the storage interface for Iterations.
type GormIterationRepository struct {
	db *gorm.DB
}

// LoadMultiple returns multiple instances of iteration.Iteration
func (m *GormIterationRepository) LoadMultiple(ctx context.Context, ids []uuid.UUID) ([]*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "getmultiple"}, time.Now())
	var objs []*Iteration

	for i := 0; i < len(ids); i++ {
		m.db = m.db.Or("id = ?", ids[i])
	}
	tx := m.db.Find(&objs)
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return objs, nil
}

// Create creates a new record.
func (m *GormIterationRepository) Create(ctx context.Context, u *Iteration) error {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "create"}, time.Now())

	u.ID = uuid.NewV4()
	u.State = IterationStateNew
	err := m.db.Create(u).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"iterationID": u.ID,
			"err":         err,
		}, "unable to create the iteration")
		return errs.WithStack(err)
	}

	return nil
}

// List all Iterations related to a single item
func (m *GormIterationRepository) List(ctx context.Context, spaceID uuid.UUID) ([]*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "query"}, time.Now())
	var objs []*Iteration

	err := m.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(ctx, map[string]interface{}{
			"spaceID": spaceID,
			"err":     err,
		}, "unable to list the iterations")
		return nil, errs.WithStack(err)
	}
	return objs, nil
}

// Load a single Iteration regardless of parent
func (m *GormIterationRepository) Load(ctx context.Context, id uuid.UUID) (*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "get"}, time.Now())
	var obj Iteration

	tx := m.db.Where("id = ?", id).First(&obj)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"iterationID": id.String(),
		}, "iteration cannot be found")
		return nil, errors.NewNotFoundError("Iteration", id.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"iterationID": id.String(),
			"err":         tx.Error,
		}, "unable to load the iteration")
		return nil, errors.NewInternalError(tx.Error.Error())
	}
	return &obj, nil
}

// Save updates the given iteration in the db. Version must be the same as the one in the stored version
// returns NotFoundError, VersionConflictError or InternalError
func (m *GormIterationRepository) Save(ctx context.Context, i Iteration) (*Iteration, error) {
	itr := Iteration{}
	tx := m.db.Where("id=?", i.ID).First(&itr)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"iterationID": i.ID,
		}, "iteration cannot be found")
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("iteration", i.ID.String())
	}
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"iterationID": i.ID,
			"err":         err,
		}, "unknown error happened when searching the iteration")
		return nil, errors.NewInternalError(err.Error())
	}
	tx = tx.Save(&i)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"iterationID": i.ID,
			"err":         err,
		}, "unable to save the iterations")
		return nil, errors.NewInternalError(err.Error())
	}
	return &i, nil
}

// CanStartIteration checks the rule - Only one iteration from a space can have state=start at a time.
// More rules can be added as needed in this function
func (m *GormIterationRepository) CanStartIteration(ctx context.Context, i *Iteration) (bool, error) {
	var count int64
	m.db.Model(&Iteration{}).Where("space_id=? and state=?", i.SpaceID, IterationStateStart).Count(&count)
	if count != 0 {
		log.Error(ctx, map[string]interface{}{
			"iterationID": i.ID,
			"spaceID":     i.SpaceID,
		}, "one iteration from given space is already running!")
		return false, errors.NewBadParameterError("state", "One iteration from given space is already running")
	}
	return true, nil
}

// List returns backlog items where iteration is in root iteration and status != closed
func (m *GormIterationRepository) ListBacklogIterations(ctx context.Context, spaceID uuid.UUID, start *int, limit *int) ([]*Iteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "iteration", "query"}, time.Now())
	var (
		rows []*Iteration
		err  error
	)
	// FIXME: hector. Ensure that this query is correct
	db := m.db.Where("space_id = ? AND state != ? AND path = ?", spaceID, IterationStateClose, "")
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, errs.WithStack(err)
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(ctx, map[string]interface{}{
			"spaceID": spaceID,
			"err":     err,
		}, "unable to list backlog iterations")
		return nil, errs.WithStack(err)
	}
	return rows, nil
}
