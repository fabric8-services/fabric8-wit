package remoteworkitem

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// APIStringTypeTrackerQueries helps to avoid string literal
const APIStringTypeTrackerQuery = "trackerquery"

const trackerQueriesTableName = "tracker_queries"

// GormTrackerQueryRepository implements TrackerRepository using gorm
type GormTrackerQueryRepository struct {
	db *gorm.DB
}

// NewTrackerQueryRepository constructs a TrackerQueryRepository
func NewTrackerQueryRepository(db *gorm.DB) *GormTrackerQueryRepository {
	return &GormTrackerQueryRepository{db}
}

// TrackerQueryRepository encapsulate storage & retrieval of tracker queries
type TrackerQueryRepository interface {
	repository.Exister
	Create(ctx context.Context, tq *TrackerQuery) error
	Save(ctx context.Context, tq TrackerQuery) (*TrackerQuery, error)
	Load(ctx context.Context, ID uuid.UUID) (*TrackerQuery, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	List(ctx context.Context) ([]TrackerQuery, error)
}

// Create creates a new tracker query in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Create(ctx context.Context, tq *TrackerQuery) error {
	if err := r.db.Create(&tq).Error; err != nil {
		return errors.NewInternalError(ctx, r.db.Error)
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_query": tq,
	}, "Created tracker query")

	return nil
}

// Load returns the tracker query for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Load(ctx context.Context, ID uuid.UUID) (*TrackerQuery, error) {
	defer goa.MeasureSince([]string{"goa", "db", "trackerquery", "load"}, time.Now())
	res := TrackerQuery{}
	tx := r.db.Where("id = ?", ID).Find(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": ID.String(),
		}, "tracker resource not found")
		return nil, errors.NewNotFoundError("tracker query", ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":             tx.Error,
			"trackerquery_id": ID,
		}, "unable to load the trackerquery by ID")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormTrackerQueryRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	return repository.CheckExists(ctx, r.db, trackerQueriesTableName, id)
}

// Save updates the given tracker query in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Save(ctx context.Context, tq TrackerQuery) (*TrackerQuery, error) {
	defer goa.MeasureSince([]string{"goa", "db", "trackerquery", "save"}, time.Now())
	res := TrackerQuery{}

	tx := r.db.Where("id = ?", tq.ID).Find(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"tracker_id": tq.ID.String(),
		}, "tracker query not found")

		return nil, errors.NewNotFoundError("TrackerQuery", tq.ID.String())
	}

	tx = r.db.Where("tracker_id = ?", tq.TrackerID).Find(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"tracker_id": tq.TrackerID,
		}, "tracker ID not found")
		return nil, errors.NewNotFoundError("Tracker", tq.TrackerID.String())
	}

	if err := tx.Save(&res).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"trackerquery_id": tq.ID,
			"err":             err,
		}, "unable to save the tracker query")
		return nil, errors.NewInternalError(ctx, err)
	}

	return &res, nil
}

// Delete deletes the tracker query with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerQueryRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "trackerquery", "delete"}, time.Now())
	if ID == uuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"err":        errors.NewNotFoundError("trackerquery", ID.String()),
			"tracker_id": ID.String(),
		}, "unable to find the tracker query by ID")
		return errors.NewNotFoundError("trackerquery", ID.String())
	}
	var tq = TrackerQuery{ID: ID}
	tx := r.db.Delete(tq)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        err,
			"tracker_id": ID.String(),
		}, "unable to delete the space")
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"err":             tx.Error,
			"trackerquery_id": ID.String(),
		}, "none row was affected by the deletion operation")
		return errors.NewNotFoundError("trackerquery", ID.String())
	}
	return nil
}

// List returns tracker query selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerQueryRepository) List(ctx context.Context) ([]TrackerQuery, error) {
	defer goa.MeasureSince([]string{"goa", "db", "trackerquery", "query"}, time.Now())
	var objs []TrackerQuery
	err := r.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}
