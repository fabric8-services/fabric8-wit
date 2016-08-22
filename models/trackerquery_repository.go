package models

import (
	"log"

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
func (r *GormTrackerQueryRepository) Create(ctx context.Context, query string, schedule string) (*app.TrackerQuery, error) {
	tq := TrackerQuery{
		Query:    query,
		Schedule: schedule}
	tx := r.ts.tx
	if err := tx.Create(&tq).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Printf("created tracker query %v\n", tq)
	tq2 := app.TrackerQuery{
		ID:       string(tq.ID),
		Query:    query,
		Schedule: schedule}

	return &tq2, nil
}
