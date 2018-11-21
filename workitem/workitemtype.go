package workitem

import (
	"reflect"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// String constants for the local work item types.
const (
	// pathSep specifies the symbol used to concatenate WIT names to form a so
	// called "path"
	pathSep = "."

	SystemVersion = "version"

	SystemRemoteItemID        = "system_remote_item_id"
	SystemNumber              = "system_number"
	SystemTitle               = "system_title"
	SystemDescription         = "system_description"
	SystemDescriptionMarkup   = "system_description_markup"
	SystemDescriptionRendered = "system_description_rendered"
	SystemState               = "system_state"
	SystemAssignees           = "system_assignees"
	SystemCreator             = "system_creator"
	SystemCreatedAt           = "system_created_at"
	SystemUpdatedAt           = "system_updated_at"
	SystemOrder               = "system_order"
	SystemIteration           = "system_iteration"
	SystemArea                = "system_area"
	SystemCodebase            = "system_codebase"
	SystemLabels              = "system_labels"
	SystemBoardcolumns        = "system_boardcolumns"
	SystemMetaState           = "system_metastate"

	SystemBoard = "Board"

	SystemStateOpen       = "open"
	SystemStateNew        = "new"
	SystemStateInProgress = "in progress"
	SystemStateResolved   = "resolved"
	SystemStateClosed     = "closed"
)

// NewToOldFieldNameMap is a temporary map which stores old field names.
// TODO(ibrahim): Remove this when field name migration is completed.
var NewToOldFieldNameMap = map[string]string{
	SystemRemoteItemID:        "system.remote_item_id",
	SystemNumber:              "system.number",
	SystemTitle:               "system.title",
	SystemDescription:         "system.description",
	SystemDescriptionMarkup:   "system.description.markup",
	SystemDescriptionRendered: "system.description.rendered",
	SystemState:               "system.state",
	SystemAssignees:           "system.assignees",
	SystemCreator:             "system.creator",
	SystemCreatedAt:           "system.created_at",
	SystemUpdatedAt:           "system.updated_at",
	SystemOrder:               "system.order",
	SystemIteration:           "system.iteration",
	SystemArea:                "system.area",
	SystemCodebase:            "system.codebase",
	SystemLabels:              "system.labels",
	SystemBoardcolumns:        "system.boardcolumns",
	SystemMetaState:           "system.metastate",
}

// OldToNewFieldNameMap is a temporary map which stores old field names.
// TODO(ibrahim): Remove this when field name migration is completed.
var OldToNewFieldNameMap = map[string]string{
	"system.remote_item_id":       SystemRemoteItemID,
	"system.number":               SystemNumber,
	"system.title":                SystemTitle,
	"system.description":          SystemDescription,
	"system.description.markup":   SystemDescriptionMarkup,
	"system.description.rendered": SystemDescriptionRendered,
	"system.state":                SystemState,
	"system.assignees":            SystemAssignees,
	"system.creator":              SystemCreator,
	"system.created_at":           SystemCreatedAt,
	"system.updated_at":           SystemUpdatedAt,
	"system.order":                SystemOrder,
	"system.iteration":            SystemIteration,
	"system.area":                 SystemArea,
	"system.codebase":             SystemCodebase,
	"system.labels":               SystemLabels,
	"system.boardcolumns":         SystemBoardcolumns,
	"system.metastate":            SystemMetaState,
}

// Never ever change these UUIDs!!!
var (
	// base item type with common fields for planner item types like userstory,
	// experience, bug, feature, etc.
	SystemPlannerItem      = uuid.FromStringOrNil("86af5178-9b41-469b-9096-57e5155c3f31") // "planneritem"
	SystemTask             = uuid.FromStringOrNil("bbf35418-04b6-426c-a60b-7f80beb0b624") // "task"
	SystemValueProposition = uuid.FromStringOrNil("3194ab60-855b-4155-9005-9dce4a05f1eb") // "valueproposition"
	SystemFundamental      = uuid.FromStringOrNil("ee7ca005-f81d-4eea-9b9b-1965df0988d0") // "fundamental"
	SystemExperience       = uuid.FromStringOrNil("b9a71831-c803-4f66-8774-4193fffd1311") // "experience"
	SystemFeature          = uuid.FromStringOrNil("0a24d3c2-e0a6-4686-8051-ec0ea1915a28") // "feature"
	SystemScenario         = uuid.FromStringOrNil("71171e90-6d35-498f-a6a7-2083b5267c18") // "scenario"
	SystemBug              = uuid.FromStringOrNil("26787039-b68f-4e28-8814-c2f93be1ef4e") // "bug"
	SystemPapercuts        = uuid.FromStringOrNil("6d603ab4-7c5e-4c5f-bba8-a3ba9d370985") // "papercuts"
)

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	gormsupport.Lifecycle `json:"lifecycle,omitempty"`

	// ID is the primary key of a work item type.
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id,omitempty"`

	// Name is a human readable name of this work item type.
	Name string `json:"name,omitempty"`

	// Description is an optional description of the work item type.
	Description *string `json:"description,omitempty"`

	// Icon contains the CSS icon class(es) to render an icon for the work item
	// type.
	Icon string `json:"icon,omitempty"`

	// Version contains the revision number of this work item type and is used
	// for optimistic concurrency control.
	Version int `json:"version,omitempty"`

	// Path contains the IDs of the parents, separated with a dot (".")
	// separator.
	// TODO(kwk): Think about changing this to the dedicated path type also used
	// by iterations.
	Path string `json:"path,omitempty"`

	// Fields contains the definitions of the fields this work item type
	// supports.
	Fields FieldDefinitions `sql:"type:jsonb" json:"fields,omitempty"`

	// SpaceTemplateID refers to the space template to which this work item type
	// belongs.
	SpaceTemplateID uuid.UUID `sql:"type:uuid" json:"space_template_id,omitempty"`

	// Extends is a helper ID to support "extends" attribute of WIT in a space
	// template. This field is not filled when you load a work item type from
	// the DB. Instead the Path member contains the information.
	Extends uuid.UUID `gorm:"-" json:"extends,omitempty"`

	// CanConstruct is true when you can create work items from this work item
	// type.
	CanConstruct bool `gorm:"can_construct" json:"can_construct,omitempty"`

	// ChildTypeIDs is a list of work item type IDs that can be used as child
	// type of this work item. This field is filled upon loading the work item
	// type from the DB.
	ChildTypeIDs []uuid.UUID `gorm:"-" json:"child_types,omitempty"`
}

// Validate runs some checks on the work item type to ensure the field
// definitions make sense.
func (wit WorkItemType) Validate() error {
	if strings.TrimSpace(wit.Name) == "" {
		return errs.Errorf(`work item type name "%s" when trimmed has a zero-length`, wit.Name)
	}
	if err := wit.Fields.Validate(); err != nil {
		return errs.Wrapf(err, "failed to validate work item type's fields")
	}
	return nil
}

// GetTypePathSeparator returns the work item type's path separator "."
func GetTypePathSeparator() string {
	return pathSep
}

// LtreeSafeID returns the ID of the work item type in an postgres ltree safe manner.
// The returned string can be used as an ltree node.
func (wit WorkItemType) LtreeSafeID() string {
	return LtreeSafeID(wit.ID)
}

// LtreeSafeID returns the ID of the work item type in an postgres ltree safe manner
// The returned string can be used as an ltree node.
func LtreeSafeID(witID uuid.UUID) string {
	return strings.Replace(witID.String(), "-", "_", -1)
}

// TableName implements gorm.tabler
func (wit WorkItemType) TableName() string {
	return "work_item_types"
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemType{}
var _ convert.Equaler = (*WorkItemType)(nil)

// returns true if the left hand and right hand side string
// pointers either both point to nil or reference the same
// content; otherwise false is returned.
func strPtrIsNilOrContentIsEqual(l, r *string) bool {
	if l == nil && r != nil {
		return false
	}
	if l != nil && r == nil {
		return false
	}
	if l == nil && r == nil {
		return true
	}
	return *l == *r
}

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (wit WorkItemType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	if wit.ID != other.ID {
		return false
	}
	if !convert.CascadeEqual(wit.Lifecycle, other.Lifecycle) {
		return false
	}
	if wit.Version != other.Version {
		return false
	}
	if wit.Name != other.Name {
		return false
	}
	if wit.Extends != other.Extends {
		return false
	}
	if wit.CanConstruct != other.CanConstruct {
		return false
	}
	if !reflect.DeepEqual(wit.Description, other.Description) {
		return false
	}
	if wit.Icon != other.Icon {
		return false
	}
	if wit.Path != other.Path {
		return false
	}
	if len(wit.ChildTypeIDs) != len(other.ChildTypeIDs) {
		return false
	}
	for i := range wit.ChildTypeIDs {
		if wit.ChildTypeIDs[i] != other.ChildTypeIDs[i] {
			return false
		}
	}
	if len(wit.Fields) != len(other.Fields) {
		return false
	}
	for witKey, witVal := range wit.Fields {
		otherVal, keyFound := other.Fields[witKey]
		if !keyFound {
			return false
		}
		if !convert.CascadeEqual(witVal, otherVal) {
			return false
		}
	}
	if wit.SpaceTemplateID != other.SpaceTemplateID {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler interface
func (wit WorkItemType) EqualValue(u convert.Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	wit.Version = other.Version
	wit.Lifecycle = other.Lifecycle
	return wit.Equal(u)
}

// ConvertWorkItemStorageToModel converts a workItem from the storage/persistence layer into a workItem of the model domain layer
func (wit WorkItemType) ConvertWorkItemStorageToModel(workItem WorkItemStorage) (*WorkItem, error) {
	result := WorkItem{
		ID:                     workItem.ID,
		Number:                 workItem.Number,
		Type:                   workItem.Type,
		Version:                workItem.Version,
		Fields:                 map[string]interface{}{},
		SpaceID:                workItem.SpaceID,
		relationShipsChangedAt: workItem.RelationShipsChangedAt,
	}

	for name, field := range wit.Fields {
		var err error
		if name == SystemCreatedAt {
			continue
		}
		result.Fields[name], err = field.ConvertFromModel(name, workItem.Fields[name])
		if err != nil {
			return nil, errs.WithStack(err)
		}
		result.Fields[SystemOrder] = workItem.ExecutionOrder
	}

	return &result, nil
}

// IsTypeOrSubtypeOf returns true if the work item type with the given type ID,
// is of the same type as the current WIT or of it is a subtype; otherwise false
// is returned.
func (wit WorkItemType) IsTypeOrSubtypeOf(typeID uuid.UUID) bool {
	// Check for complete inclusion (e.g. "bar" is contained in "foo.bar.cake")
	// and for suffix (e.g. ".cake" is the suffix of "foo.bar.cake").
	return uuid.Equal(wit.ID, typeID) || strings.Contains(wit.Path, LtreeSafeID(typeID)+pathSep)
}

// GetETagData returns the field values to use to generate the ETag
func (wit WorkItemType) GetETagData() []interface{} {
	return []interface{}{wit.ID, wit.Version}
}

// GetLastModified returns the last modification time
func (wit WorkItemType) GetLastModified() time.Time {
	return wit.UpdatedAt
}
