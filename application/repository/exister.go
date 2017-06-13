package repository

import (
	"context"

	uuid "github.com/satori/go.uuid"
)

type Exister interface {
	// Exists returns true as the first parameter if the object with the given ID exists;
	// otherwise false is returned. The error is either nil in case of success or not nil
	// if there has been an issue.
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}
