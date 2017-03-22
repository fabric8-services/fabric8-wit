package link

import (
	"strconv"

	"github.com/almighty/almighty-core/app"
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLink represents the connection of two work items as it is stored in the db
type WorkItemLink struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version    int
	SourceID   uint64
	TargetID   uint64
	LinkTypeID uuid.UUID `sql:"type:uuid"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLink{}
var _ convert.Equaler = (*WorkItemLink)(nil)

// Equal returns true if two WorkItemLink objects are equal; otherwise false is returned.
func (l WorkItemLink) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLink)
	if !ok {
		return false
	}
	if !l.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !uuid.Equal(l.ID, other.ID) {
		return false
	}
	if l.Version != other.Version {
		return false
	}
	if l.SourceID != other.SourceID {
		return false
	}
	if l.TargetID != other.TargetID {
		return false
	}
	if l.LinkTypeID != other.LinkTypeID {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link
// cannot be used for the creation of a new work item link.
func (l *WorkItemLink) CheckValidForCreation() error {
	if uuid.Equal(l.LinkTypeID, uuid.Nil) {
		return errors.NewBadParameterError("link_type_id", l.LinkTypeID)
	}
	return nil
}

// TableName implements gorm.tabler
func (l WorkItemLink) TableName() string {
	return "work_item_links"
}

// ConvertLinkFromModel converts a work item from model to REST representation
func ConvertLinkFromModel(t WorkItemLink) app.WorkItemLinkSingle {
	var converted = app.WorkItemLinkSingle{
		Data: &app.WorkItemLinkData{
			Type: EndpointWorkItemLinks,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkAttributes{
				Version: &t.Version,
			},
			Relationships: &app.WorkItemLinkRelationships{
				LinkType: &app.RelationWorkItemLinkType{
					Data: &app.RelationWorkItemLinkTypeData{
						Type: EndpointWorkItemLinkTypes,
						ID:   t.LinkTypeID,
					},
				},
				Source: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: EndpointWorkItems,
						ID:   strconv.FormatUint(t.SourceID, 10),
					},
				},
				Target: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: EndpointWorkItems,
						ID:   strconv.FormatUint(t.TargetID, 10),
					},
				},
			},
		},
	}
	return converted
}

// ConvertLinkToModel converts the incoming app representation of a work item link to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
// NOTE: Only the LinkTypeID, SourceID, and TargetID fields will be set.
//       You need to preload the elements after calling this function.
func ConvertLinkToModel(in app.WorkItemLinkSingle, out *WorkItemLink) error {
	attrs := in.Data.Attributes
	rel := in.Data.Relationships
	var err error

	if in.Data.ID != nil {
		out.ID = *in.Data.ID
	}

	if attrs != nil && attrs.Version != nil {
		out.Version = *attrs.Version
	}

	if rel != nil && rel.LinkType != nil && rel.LinkType.Data != nil {
		out.LinkTypeID = rel.LinkType.Data.ID
	}

	if rel != nil && rel.Source != nil && rel.Source.Data != nil {
		d := rel.Source.Data
		// The the work item id MUST NOT be empty
		if d.ID == "" {
			return errors.NewBadParameterError("data.relationships.source.data.id", d.ID)
		}
		if out.SourceID, err = strconv.ParseUint(d.ID, 10, 64); err != nil {
			return errors.NewBadParameterError("data.relationships.source.data.id", d.ID)
		}
	}

	if rel != nil && rel.Target != nil && rel.Target.Data != nil {
		d := rel.Target.Data
		// If the the target type is not nil, it MUST be "workitems"
		// The the work item id MUST NOT be empty
		if d.ID == "" {
			return errors.NewBadParameterError("data.relationships.target.data.id", d.ID)
		}
		if out.TargetID, err = strconv.ParseUint(d.ID, 10, 64); err != nil {
			return errors.NewBadParameterError("data.relationships.target.data.id", d.ID)
		}
	}

	return nil
}
