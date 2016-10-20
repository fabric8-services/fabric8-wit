package search

import (
	"context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"github.com/almighty/almighty-core/models"
)

// GormSearchRepository provides a Gorm based repository
type GormSearchRepository struct {
	ts *models.GormTransactionSupport
}

// NewGormSearchRepository creates a new search repository
func NewGormSearchRepository(ts *models.GormTransactionSupport) *GormSearchRepository {
	return &GormSearchRepository{ts}
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormSearchRepository) List(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]*app.WorkItem, error) {
	return nil, nil
}
