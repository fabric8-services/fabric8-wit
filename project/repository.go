package project

import (
	satoriuuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

// Repository encapsulate storage & retrieval of projects
type Repository interface {
	Create(ctx context.Context, name string) (*Project, error)
	Save(ctx context.Context, project Project) (*Project, error)
	Load(ctx context.Context, ID satoriuuid.UUID) (*Project, error)
	Delete(ctx context.Context, ID satoriuuid.UUID) error
	List(ctx context.Context, start *int, length *int) ([]Project, uint64, error)
}
