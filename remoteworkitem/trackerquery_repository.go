package remoteworkitem

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// GormTrackerQueryRepository implements TrackerRepository using gorm
type GormTrackerQueryRepository struct {
	db *gorm.DB
}

// NewTrackerQueryRepository constructs a TrackerQueryRepository
func NewTrackerQueryRepository(db *gorm.DB) *GormTrackerQueryRepository {
	return &GormTrackerQueryRepository{db}
}

func updateTrackerQuery(db *gorm.DB, tqID int, lu *time.Time) error {
	tq := TrackerQuery{}
	tx := db.First(&tq, tqID)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", tq)
		return NotFoundError{entity: "tracker_query", ID: string(tq.ID)}
	}
	tq.LastUpdated = lu
	if err := tx.Save(&tq).Error; err != nil {
		log.Print(err.Error())
		return InternalError{simpleError{err.Error()}}
	}
	return nil
}

// Create creates a new tracker query in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Create(ctx context.Context, query string, schedule string, tracker string) (*app.TrackerQuery, error) {
	tid, err := strconv.ParseUint(tracker, 10, 64)
	if err != nil || tid == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", tracker}
	}
	fmt.Printf("tracker id: %v", tid)
	tq := TrackerQuery{
		Query:     query,
		Schedule:  schedule,
		TrackerID: tid}
	tx := r.db
	if err := tx.Create(&tq).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker query %v\n", tq)
	tq2 := app.TrackerQuery{
		ID:        strconv.FormatUint(tq.ID, 10),
		Query:     query,
		Schedule:  schedule,
		TrackerID: tracker}

	return &tq2, nil
}

// Load returns the tracker query for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Load(ctx context.Context, ID string) (*app.TrackerQuery, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker query", ID}
	}

	log.Printf("loading tracker query %d", id)
	res := TrackerQuery{}
	if r.db.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"tracker query", ID}
	}
	tq := app.TrackerQuery{
		ID:        strconv.FormatUint(res.ID, 10),
		Query:     res.Query,
		Schedule:  res.Schedule,
		TrackerID: strconv.FormatUint(res.TrackerID, 10)}

	return &tq, nil
}

// Save updates the given tracker query in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error) {
	res := TrackerQuery{}
	id, err := strconv.ParseUint(tq.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, NotFoundError{entity: "trackerquery", ID: tq.ID}
	}

	tid, err := strconv.ParseUint(tq.TrackerID, 10, 64)
	if err != nil || tid == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", tq.TrackerID}
	}

	log.Printf("looking for id %d", id)
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "TrackerQuery", ID: tq.ID}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{fmt.Sprintf("could not load tracker query: %s", tx.Error.Error())}}
	}

	tx = r.db.First(&Tracker{}, tid)
	if tx.RecordNotFound() {
		log.Printf("not found, id=%d", id)
		return nil, NotFoundError{entity: "tracker", ID: tq.TrackerID}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{fmt.Sprintf("could not load tracker: %s", tx.Error.Error())}}
	}

	newTq := TrackerQuery{
		ID:        id,
		Schedule:  tq.Schedule,
		Query:     tq.Query,
		TrackerID: tid}

	if err := tx.Save(&newTq).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated tracker query to %v\n", newTq)
	t2 := app.TrackerQuery{
		ID:        tq.ID,
		Schedule:  tq.Schedule,
		Query:     tq.Query,
		TrackerID: tq.TrackerID}

	return &t2, nil
}

// Delete deletes the tracker query with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerQueryRepository) Delete(ctx context.Context, ID string) error {
	var tq = TrackerQuery{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treat as not found: clients don't know it must be a number
		return NotFoundError{entity: "trackerquery", ID: ID}
	}
	tq.ID = id
	tx := r.db
	tx = tx.Delete(tq)
	if err = tx.Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return NotFoundError{entity: "trackerquery", ID: ID}
	}
	return nil
}

// List returns tracker query selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerQueryRepository) List(ctx context.Context) ([]*app.TrackerQuery, error) {
	var rows []TrackerQuery
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]*app.TrackerQuery, len(rows))
	for i, tq := range rows {
		t := app.TrackerQuery{
			ID:        strconv.FormatUint(tq.ID, 10),
			Schedule:  tq.Schedule,
			Query:     tq.Query,
			TrackerID: strconv.FormatUint(tq.TrackerID, 10)}
		result[i] = &t
	}
	return result, nil
}
