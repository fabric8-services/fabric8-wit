package workitem

import (
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Use following bucket constants while defining static groups.
// NOTE: Those buckets can later be used by reporting tools for example to gather
// information on a collective range of work item types.
const (
	BucketPortfolio   TypeBucket = "portfolio"
	BucketRequirement TypeBucket = "requirement"
	BucketIteration   TypeBucket = "iteration"
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

// WorkItemTypeGroup represents the node in the group of work item types
type WorkItemTypeGroup struct {
	gormsupport.Lifecycle `json:"lifecycle"`
	ID                    uuid.UUID   `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id"`
	SpaceTemplateID       uuid.UUID   `sql:"type:uuid" json:"space_template_id"`
	Bucket                TypeBucket  `json:"bucket,omitempty"`
	Name                  string      `json:"name,omitempty"` // the name to be displayed to user (is unique)
	Description           *string     `json:"description,omitempty"`
	Icon                  string      `json:"icon,omitempty"`
	Position              int         `json:"-"`
	TypeList              []uuid.UUID `gorm:"-" json:"type_list,omitempty"`
}

// TableName implements gorm.tabler
func (witg WorkItemTypeGroup) TableName() string {
	return "work_item_type_groups"
}

// Ensure WorkItemTypeGroup implements the Equaler interface
var _ convert.Equaler = WorkItemTypeGroup{}
var _ convert.Equaler = (*WorkItemTypeGroup)(nil)

// Equal returns true if two WorkItemTypeGroup objects are equal; otherwise false is returned.
func (witg WorkItemTypeGroup) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemTypeGroup)
	if !ok {
		return false
	}
	if witg.ID != other.ID {
		return false
	}
	if witg.SpaceTemplateID != other.SpaceTemplateID {
		return false
	}
	if !convert.CascadeEqual(witg.Lifecycle, other.Lifecycle) {
		return false
	}
	if witg.Name != other.Name {
		return false
	}
	if !reflect.DeepEqual(witg.Description, other.Description) {
		return false
	}
	if witg.Bucket != other.Bucket {
		return false
	}
	if witg.Icon != other.Icon {
		return false
	}
	if witg.Position != other.Position {
		return false
	}
	if len(witg.TypeList) != len(other.TypeList) {
		return false
	}
	for i := range witg.TypeList {
		if witg.TypeList[i] != other.TypeList[i] {
			return false
		}
	}
	return true
}

// EqualValue implements convert.Equaler interface
func (witg WorkItemTypeGroup) EqualValue(u convert.Equaler) bool {
	other, ok := u.(WorkItemTypeGroup)
	if !ok {
		return false
	}
	witg.Lifecycle = other.Lifecycle
	return witg.Equal(u)
}

// GetETagData returns the field values to use to generate the ETag
func (witg WorkItemTypeGroup) GetETagData() []interface{} {
	return []interface{}{witg.ID, witg.UpdatedAt}
}

// GetLastModified returns the last modification time
func (witg WorkItemTypeGroup) GetLastModified() time.Time {
	return witg.UpdatedAt
}

type typeGroupMember struct {
	gormsupport.Lifecycle
	ID             uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	TypeGroupID    uuid.UUID `sql:"type:uuid"`
	WorkItemTypeID uuid.UUID `sql:"type:uuid"`
	Position       int       // position in type list of type group
}

// TableName implements gorm.tabler
func (wit typeGroupMember) TableName() string {
	return "work_item_type_group_members"
}

// TypeGroups returns the list of work item type groups
func TypeGroups() []WorkItemTypeGroup {
	scenariosID := uuid.FromStringOrNil("feb28a28-44a6-43f8-946a-bae987713891")
	experiencesID := uuid.FromStringOrNil("d4e2c859-f416-4e9a-a3e0-e7bb4e1b454b")
	requirementsID := uuid.FromStringOrNil("bb1de8b6-3175-4821-abe9-50d0a64f19a2")
	executionID := uuid.FromStringOrNil("7fdfde54-9cf2-4098-b33b-30cd505dcfc3")

	return []WorkItemTypeGroup{
		// There can be more than one groups in the "Portfolio" bucket
		{
			ID:     scenariosID,
			Bucket: BucketPortfolio,
			Name:   "Scenarios",
			Icon:   "fa fa-bullseye",
			TypeList: []uuid.UUID{
				SystemScenario,
				SystemFundamental,
				SystemPapercuts,
			},
		},
		{
			ID:     experiencesID,
			Bucket: BucketPortfolio,
			Name:   "Experiences",
			Icon:   "pficon pficon-infrastructure",
			TypeList: []uuid.UUID{
				SystemExperience,
				SystemValueProposition,
			},
		},
		// There's always only one group in the "Requirement" bucket
		{
			ID:     requirementsID,
			Bucket: BucketRequirement,
			Name:   "Requirements",
			Icon:   "fa fa-list-ul",
			TypeList: []uuid.UUID{
				SystemFeature,
				SystemBug,
			},
		},
		// There's always only one group in the "Iteration" bucket
		{
			ID:     executionID,
			Bucket: BucketIteration,
			Name:   "Execution",
			Icon:   "fa fa-repeat",
			TypeList: []uuid.UUID{
				SystemTask,
				SystemBug,
				SystemFeature,
			},
		},
	}
}

// TypeGroupByName returns a type group based on its name if such a group
// exists; otherwise nil is returned.
func TypeGroupByName(name string) *WorkItemTypeGroup {
	for _, t := range TypeGroups() {
		if t.Name == name {
			return &t
		}
	}
	return nil
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
