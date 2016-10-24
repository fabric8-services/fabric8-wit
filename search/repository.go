package search

import (
	"context"

	"github.com/almighty/almighty-core/app"
)

// Repository encapsulate storage & retrieval of tracker configuration
type Repository interface {
	Search(ctx context.Context, q string) ([]*app.WorkItem, error)
}