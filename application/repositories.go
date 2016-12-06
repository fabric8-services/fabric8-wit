package application

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/criteria"
	satoriuuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
)

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

// WorkItemLinkCategoryRepository encapsulates storage & retrieval of work item link categories
type WorkItemLinkCategoryRepository interface {
	Create(ctx context.Context, name *string, description *string) (*app.WorkItemLinkCategory, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLinkCategory, error)
	List(ctx context.Context) (*app.WorkItemLinkCategoryArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLinkCategory) (*app.WorkItemLinkCategory, error)
}

// WorkItemLinkTypeRepository encapsulates storage & retrieval of work item link types
type WorkItemLinkTypeRepository interface {
	Create(ctx context.Context, name string, description *string, sourceTypeName, targetTypeName, forwardName, reverseName, topology string, linkCategory satoriuuid.UUID) (*app.WorkItemLinkType, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLinkType, error)
	List(ctx context.Context) (*app.WorkItemLinkTypeArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLinkType) (*app.WorkItemLinkType, error)
}

// WorkItemLinkRepository encapsulates storage & retrieval of work item links
type WorkItemLinkRepository interface {
	Create(ctx context.Context, sourceID, targetID uint64, linkTypeID satoriuuid.UUID) (*app.WorkItemLink, error)
	Load(ctx context.Context, ID string) (*app.WorkItemLink, error)
	List(ctx context.Context) (*app.WorkItemLinkArray, error)
	Delete(ctx context.Context, ID string) error
	Save(ctx context.Context, linkCat app.WorkItemLink) (*app.WorkItemLink, error)
}
