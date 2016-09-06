package remoteworkitem

import (
	"fmt"
	"log"
	"strconv"

	"github.com/almighty/almighty-core/app"
	"golang.org/x/net/context"
)

// GormTrackerQueryRepository implements TrackerRepository using gorm
type GormTrackerQueryRepository struct {
	ts *GormTransactionSupport
}

// NewTrackerQueryRepository constructs a TrackerQueryRepository
func NewTrackerQueryRepository(ts *GormTransactionSupport) *GormTrackerQueryRepository {
	return &GormTrackerQueryRepository{ts}
}

// Create creates a new tracker query in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Create(ctx context.Context, query string, schedule string, tracker uint64) (*app.TrackerQuery, error) {
	tid := tracker
	fmt.Printf("tracker id: %v", tid)
	tq := TrackerQuery{
		Query:     query,
		Schedule:  schedule,
		TrackerID: tid}
	tx := r.ts.tx
	if err := tx.Create(&tq).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker query %v\n", tq)
	tq2 := app.TrackerQuery{
		ID:        string(tq.ID),
		Query:     query,
		Schedule:  schedule,
		TrackerID: string(tid)}

	return &tq2, nil
}

// Load returns the tracker query for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Load(ctx context.Context, ID string) (*app.TrackerQuery, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker query", ID}
	}

	log.Printf("loading tracker query %d", id)
	res := TrackerQuery{}
	if r.ts.tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{"tracker query", ID}
	}
	tq := app.TrackerQuery{
		ID:        string(res.ID),
		Query:     res.Query,
		Schedule:  res.Schedule,
		TrackerID: string(res.TrackerID)}

	return &tq, nil
}

// Save updates the given tracker query in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error) {
	res := TrackerQuery{}
	id, err := strconv.ParseUint(tq.ID, 10, 64)
	if err != nil {
		return nil, NotFoundError{entity: "trackerquery", ID: tq.ID}
	}

	log.Printf("looking for id %d", id)
	tx := r.ts.tx
	if tx.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return nil, NotFoundError{entity: "tracker", ID: tq.ID}
	}

	newTq := TrackerQuery{
		ID:       id,
		Schedule: tq.Schedule,
		Query:    tq.Query}

	if err := tx.Save(&newTq).Error; err != nil {
		log.Print(err.Error())
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("updated tracker query to %v\n", newTq)
	t2 := app.TrackerQuery{
		ID:       string(id),
		Schedule: tq.Schedule,
		Query:    tq.Query}

	return &t2, nil
}
