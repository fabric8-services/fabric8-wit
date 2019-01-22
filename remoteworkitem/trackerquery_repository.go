package remoteworkitem

import (
	"context"
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// APIStringTypeTrackerQueries helps to avoid string literal
const APIStringTypeTrackerQuery = "trackerquery"

const trackerQueriesTableName = "tracker_queries"

// GormTrackerQueryRepository implements TrackerRepository using gorm
type GormTrackerQueryRepository struct {
	db   *gorm.DB
	witr *workitem.GormWorkItemTypeRepository
	wir  *workitem.GormWorkItemRepository
}

// NewTrackerQueryRepository constructs a TrackerQueryRepository
func NewTrackerQueryRepository(db *gorm.DB) *GormTrackerQueryRepository {
	return &GormTrackerQueryRepository{
		db:   db,
		witr: workitem.NewWorkItemTypeRepository(db),
		wir:  workitem.NewWorkItemRepository(db),
	}
}

// GetETagData returns the field values to use to generate the ETag
func (tq TrackerQuery) GetETagData() []interface{} {
	// using the 'ID' and 'UpdatedAt' (converted to number of seconds since epoch) fields
	return []interface{}{tq.ID, strconv.FormatInt(tq.UpdatedAt.Unix(), 10)}
}

// GetLastModified returns the last modification time
func (tq TrackerQuery) GetLastModified() time.Time {
	return tq.UpdatedAt.Truncate(time.Second)
}

// TrackerQueryRepository encapsulate storage & retrieval of tracker queries
type TrackerQueryRepository interface {
	repository.Exister
	Create(ctx context.Context, tq TrackerQuery) (*TrackerQuery, error)
	Load(ctx context.Context, ID uuid.UUID) (*TrackerQuery, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	List(ctx context.Context, spaceID uuid.UUID) ([]TrackerQuery, error)
}

// Create creates a new tracker query in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Create(ctx context.Context, tq TrackerQuery) (*TrackerQuery, error) {
	wiType, err := r.witr.Load(ctx, tq.WorkItemTypeID)
	if err != nil {
		return nil, errors.NewBadParameterError("WorkItemTypeID", tq.WorkItemTypeID)
	}

	allowedWIT, err := r.wir.CheckTypeAndSpaceShareTemplate(ctx, wiType, tq.SpaceID)
	if err != nil {
		return nil, err

	}
	if !allowedWIT {
		return nil, err
	}

	if err := r.db.Create(&tq).Error; err != nil {
		return nil, errors.NewInternalError(ctx, r.db.Error)
	}
	return &tq, nil
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
		return nil, errors.NewInternalError(ctx, errs.Wrapf(tx.Error, "failed to load the trackerquery by ID: %s", ID))
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormTrackerQueryRepository) CheckExists(ctx context.Context, id uuid.UUID) error {
	return repository.CheckExists(ctx, r.db, trackerQueriesTableName, id)
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

// List returns tracker queries that belong to a space
func (r *GormTrackerQueryRepository) List(ctx context.Context, spaceID uuid.UUID) ([]TrackerQuery, error) {
	defer goa.MeasureSince([]string{"goa", "db", "trackerquery", "query"}, time.Now())
	var objs []TrackerQuery
	err := r.db.Where("space_id = ?", spaceID).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}
