package link

import (
	"reflect"
	"time"

	convert "github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	errs "github.com/pkg/errors"

	uuid "github.com/satori/go.uuid"
)

// Never ever change these UUIDs!!!
var (
	SystemWorkItemLinkTypeBugBlockerID     = uuid.FromStringOrNil("2CEA3C79-3B79-423B-90F4-1E59174C8F43")
	SystemWorkItemLinkPlannerItemRelatedID = uuid.FromStringOrNil("9B631885-83B1-4ABB-A340-3A9EDE8493FA")
	SystemWorkItemLinkTypeParentChildID    = uuid.FromStringOrNil("25C326A7-6D03-4F5A-B23B-86A9EE4171E9")
)

// WorkItemLinkType represents the type of a work item link as it is stored in
// the db
type WorkItemLinkType struct {
	gormsupport.Lifecycle `json:"lifecycle,inline"`
	ID                    uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id"`
	Name                  string    `json:"name"`                  // Name is the unique name of this work item link type.
	Description           *string   `json:"description,omitempty"` // Description is an optional description of the work item link type
	Version               int       `json:"version"`               // Version for optimistic concurrency control
	Topology              Topology  `json:"topology"`              // Valid values: network, directed_network, dependency, tree
	ForwardName           string    `json:"forward_name"`
	ForwardDescription    *string   `json:"forward_description,omitempty"`
	ReverseName           string    `json:"reverse_name"`
	ReverseDescription    *string   `json:"reverse_description,omitempty"`
	SpaceTemplateID       uuid.UUID `sql:"type:uuid" json:"space_template_id"` // Reference to a space template
}

// Ensure WorkItemLinkType implements the Equaler interface
var _ convert.Equaler = WorkItemLinkType{}
var _ convert.Equaler = (*WorkItemLinkType)(nil)

// Equal returns true if two WorkItemLinkType objects are equal; otherwise false is returned.
func (t WorkItemLinkType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkType)
	if !ok {
		return false
	}
	if t.ID != other.ID {
		return false
	}
	if t.Name != other.Name {
		return false
	}
	if t.Version != other.Version {
		return false
	}
	if !convert.CascadeEqual(t.Lifecycle, other.Lifecycle) {
		return false
	}
	if !reflect.DeepEqual(t.Description, other.Description) {
		return false
	}
	if !reflect.DeepEqual(t.ForwardDescription, other.ForwardDescription) {
		return false
	}
	if !reflect.DeepEqual(t.ReverseDescription, other.ReverseDescription) {
		return false
	}
	if t.Topology != other.Topology {
		return false
	}
	if t.ForwardName != other.ForwardName {
		return false
	}
	if t.ReverseName != other.ReverseName {
		return false
	}
	if t.SpaceTemplateID != other.SpaceTemplateID {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler interface
func (t WorkItemLinkType) EqualValue(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkType)
	if !ok {
		return false
	}
	t.Version = other.Version
	t.Lifecycle = other.Lifecycle
	return t.Equal(u)
}

// CheckValidForCreation returns an error if the work item link type cannot be
// used for the creation of a new work item link type.
func (t *WorkItemLinkType) CheckValidForCreation() error {
	if t.Name == "" {
		return errors.NewBadParameterError("name", t.Name)
	}
	if t.ForwardName == "" {
		return errors.NewBadParameterError("forward_name", t.ForwardName)
	}
	if t.ReverseName == "" {
		return errors.NewBadParameterError("reverse_name", t.ReverseName)
	}
	if err := t.Topology.CheckValid(); err != nil {
		return errs.WithStack(err)
	}
	if t.SpaceTemplateID == uuid.Nil {
		return errors.NewBadParameterError("space_template_id", t.SpaceTemplateID)
	}
	return nil
}

// TableName implements gorm.tabler
func (t WorkItemLinkType) TableName() string {
	return "work_item_link_types"
}

// GetETagData returns the field values to use to generate the ETag
func (t WorkItemLinkType) GetETagData() []interface{} {
	return []interface{}{t.ID, t.Version}
}

// GetLastModified returns the last modification time
func (t WorkItemLinkType) GetLastModified() time.Time {
	return t.UpdatedAt
}
