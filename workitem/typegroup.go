package workitem

import (
	"database/sql/driver"

	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// TypeBucket represents a dedicated string type for a bucket of type groups
type TypeBucket string

// String implements the Stringer interface

func (t TypeBucket) String() string { return string(t) }

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098
// See also https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (t *TypeBucket) Scan(value interface{}) error { *t = TypeBucket(value.([]byte)); return nil }

// Value implements the https://golang.org/pkg/database/sql/driver/#Valuer interface
func (t TypeBucket) Value() (driver.Value, error) { return string(t), nil }

// Use following bucket constants while defining static groups.
// NOTE: Those buckets can later be used by reporting tools for example to gather
// information on a collective range of work item types.
const (
	BucketPortfolio   TypeBucket = "portfolio"
	BucketRequirement TypeBucket = "requirement"
	BucketIteration   TypeBucket = "iteration"
)

// WorkItemTypeGroup represents the node in the group of work item types
type WorkItemTypeGroup struct {
	gormsupport.Lifecycle
	ID          uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	Bucket      TypeBucket
	Name        string      // the name to be displayed to user
	TypeList    []uuid.UUID // TODO(kwk): We need to store this outside of this structure in the DB
	DefaultType uuid.UUID   // the work item type that is supposed to be used with the quick add for example.
	Icon        string
}

// TypeGroups returns the list of work item type groups
func TypeGroups() []WorkItemTypeGroup {
	return []WorkItemTypeGroup{
		// There can be more than one groups in the "Portfolio" bucket
		{
			ID:     uuid.FromStringOrNil("feb28a28-44a6-43f8-946a-bae987713891"),
			Bucket: BucketPortfolio,
			Name:   "Scenarios",
			Icon:   "fa fa-suitcase",
			TypeList: []uuid.UUID{
				SystemScenario,
				SystemFundamental,
				SystemPapercuts,
			},
			DefaultType: SystemScenario,
		},
		{
			ID:     uuid.FromStringOrNil("d4e2c859-f416-4e9a-a3e0-e7bb4e1b454b"),
			Bucket: BucketPortfolio,
			Name:   "Experiences",
			Icon:   "fa fa-suitcase",
			TypeList: []uuid.UUID{
				SystemExperience,
				SystemValueProposition,
			},
			DefaultType: SystemExperience,
		},
		// There's always only one group in the "Requirement" bucket
		{
			ID:     uuid.FromStringOrNil("bb1de8b6-3175-4821-abe9-50d0a64f19a2"),
			Bucket: BucketRequirement,
			Name:   "Requirements",
			Icon:   "fa fa-list-ul",
			TypeList: []uuid.UUID{
				SystemFeature,
				SystemBug,
			},
			DefaultType: SystemFeature,
		},
		// There's always only one group in the "Iteration" bucket
		{
			ID:     uuid.FromStringOrNil("7fdfde54-9cf2-4098-b33b-30cd505dcfc3"),
			Bucket: BucketIteration,
			Name:   "Execution",
			Icon:   "fa fa-repeat",
			TypeList: []uuid.UUID{
				SystemTask,
			},
			DefaultType: SystemTask,
		},
	}
}

// TypeGroupsByBucket returns all type groups which fall into the given bucket
func TypeGroupsByBucket(bucket TypeBucket) []WorkItemTypeGroup {
	res := []WorkItemTypeGroup{}
	for _, t := range TypeGroups() {
		if t.Bucket == bucket {
			res = append(res, t)
		}
	}
	return res
}
