package models

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
func (r *GormTrackerQueryRepository) Create(ctx context.Context, query string, schedule string, tracker int) (*app.TrackerQuery, error) {
	tid := tracker
	fmt.Printf("tracker id: %v", tid)
	tq := TrackerQuery{
		Query:    query,
		Schedule: schedule,
		Tracker:  tid}
	tx := r.ts.tx
	if err := tx.Debug().Create(&tq).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker query %v\n", tq)
	tq2 := app.TrackerQuery{
		ID:       string(tq.ID),
		Query:    query,
		Schedule: schedule,
		Tracker:  tid}

	return &tq2, nil
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
