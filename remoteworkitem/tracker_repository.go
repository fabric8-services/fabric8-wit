package remoteworkitem

import (
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	govalidator "gopkg.in/asaskevich/govalidator.v4"
)

// APIStringTypeTracker helps to avoid string literal
const APIStringTypeTrackers = "trackers"

const trackersTableName = "trackers"

// GormTrackerRepository implements TrackerRepository using gorm
type GormTrackerRepository struct {
	db *gorm.DB
}

// TrackerRepository encapsulate storage & retrieval of tracker configuration
type TrackerRepository interface {
	repository.Exister
	Load(ctx context.Context, ID uuid.UUID) (*Tracker, error)
	Save(ctx context.Context, t *Tracker) (*Tracker, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Create(ctx context.Context, t *Tracker) error
	List(ctx context.Context) ([]Tracker, error)
}

// NewTrackerRepository constructs a TrackerRepository
func NewTrackerRepository(db *gorm.DB) *GormTrackerRepository {
	return &GormTrackerRepository{db}
}

// Create creates a new tracker configuration in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerRepository) Create(ctx context.Context, t *Tracker) error {
	//URL Validation
	isValid := govalidator.IsURL(t.URL)
	if isValid != true {
		return BadParameterError{parameter: "url", value: t.URL}
	}

	_, present := RemoteWorkItemImplRegistry[t.Type]
	// Ensure we support this remote tracker.
	if present != true {
		return BadParameterError{parameter: "type", value: t.Type}
	}
	if err := r.db.Create(&t).Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	log.Info(ctx, map[string]interface{}{
		"tracker": t,
	}, "Tracker reposity created")

	return nil
}

// Load returns the tracker configuration for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Load(ctx context.Context, ID uuid.UUID) (*Tracker, error) {
	defer goa.MeasureSince([]string{"goa", "db", "tracker", "load"}, time.Now())

	res := Tracker{}
	tx := r.db.Where("id = ?", ID).Find(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"tracker_id": ID,
		}, "tracker repository not found")
		return nil, errors.NewNotFoundError("tracker", ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"tracker_id": ID,
		}, "unable to load the tracker by ID")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormTrackerRepository) CheckExists(ctx context.Context, id string) error {
	return repository.CheckExists(ctx, r.db, trackersTableName, id)
}

// List returns tracker selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerRepository) List(ctx context.Context) ([]Tracker, error) {
	defer goa.MeasureSince([]string{"goa", "db", "tracker", "query"}, time.Now())
	var objs []Tracker
	err := r.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// Save updates the given tracker in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Save(ctx context.Context, t *Tracker) (*Tracker, error) {
	defer goa.MeasureSince([]string{"goa", "db", "tracker", "save"}, time.Now())
	res := Tracker{}
	tx := r.db.Where("id = ?", t.ID).Find(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"tracker_id": t.ID,
		}, "tracker repository not found")
		return nil, errors.NewNotFoundError("tracker", t.ID.String())
	}
	_, present := RemoteWorkItemImplRegistry[t.Type]
	// Ensure we support this remote tracker.
	if present != true {
		return nil, errors.NewBadParameterError("type", t.Type)
	}

	if err := tx.Save(&t).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": t.ID,
			"err":        err,
		}, "unable to save tracker repository")
		return nil, errors.NewInternalError(ctx, err)
	}
	return t, nil
}

// Delete deletes the tracker with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "tracker", "delete"}, time.Now())
	if ID == uuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"err":        errors.NewNotFoundError("tracker", ID.String()),
			"tracker_id": ID.String(),
		}, "unable to find the tracker by ID")
		return errors.NewNotFoundError("tracker", ID.String())
	}
	var t = Tracker{ID: ID}
	tx := r.db.Delete(t)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        err,
			"tracker_id": ID.String(),
		}, "unable to delete the space")
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"err":      tx.Error,
			"space_id": ID.String(),
		}, "none row was affected by the deletion operation")
		return errors.NewNotFoundError("space", ID.String())
	}
	return nil
}
