package search

import (
	"context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
)

// Repository encapsulate storage & retrieval of tracker configuration
type Repository interface {
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, error)
}
