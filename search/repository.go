package search

import (
	"context"

	"github.com/almighty/almighty-core/app"
)

// Repository encapsulates searching of woritems,users,etc
type Repository interface {
	SearchFullText(ctx context.Context, q string) ([]*app.WorkItem, error)
}
