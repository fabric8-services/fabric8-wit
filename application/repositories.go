package application

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	"golang.org/x/net/context"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	Load(ctx context.Context, ID string) (*app.WorkItem, error)
	Save(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, typeID string, fields map[string]interface{}, creator string) (*app.WorkItem, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.WorkItem, uint64, error)
	Patch(ctx context.Context, wi app.WorkItem) (*app.WorkItem, error)
}

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	Load(ctx context.Context, name string) (*app.WorkItemType, error)
	Create(ctx context.Context, extendedTypeID *string, name string, fields map[string]app.FieldDefinition) (*app.WorkItemType, error)
	List(ctx context.Context, start *int, length *int) ([]*app.WorkItemType, error)
}

// TrackerRepository encapsulate storage & retrieval of tracker configuration
type TrackerRepository interface {
	Load(ctx context.Context, ID string) (*app.Tracker, error)
	Save(ctx context.Context, t app.Tracker) (*app.Tracker, error)
	Delete(ctx context.Context, ID string) error
	Create(ctx context.Context, url string, typeID string) (*app.Tracker, error)
	List(ctx context.Context, criteria criteria.Expression, start *int, length *int) ([]*app.Tracker, error)
}

// TrackerQueryRepository encapsulate storage & retrieval of tracker queries
type TrackerQueryRepository interface {
	Create(ctx context.Context, query string, schedule string, tracker string) (*app.TrackerQuery, error)
	Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error)
	Load(ctx context.Context, ID string) (*app.TrackerQuery, error)
	Delete(ctx context.Context, ID string) error
	List(ctx context.Context) ([]*app.TrackerQuery, error)
}

// SearchRepository encapsulates searching of woritems,users,etc
type SearchRepository interface {
	SearchFullText(ctx context.Context, searchStr string, start *int, length *int) ([]*app.WorkItem, uint64, error)
}

// IdentityRepository encapsulates identity
type IdentityRepository interface {
	List(ctx context.Context) (*app.IdentityArray, error)
}
