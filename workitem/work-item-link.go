package workitem

import (
	"strconv"

	"github.com/almighty/almighty-core/app"
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

// WorkItemLink represents the connection of two work items as it is stored in the db
type WorkItemLink struct {
	gormsupport.Lifecycle
	// ID
	ID satoriuuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version    int
	SourceID   uint64
	TargetID   uint64
	LinkTypeID satoriuuid.UUID `sql:"type:uuid default uuid_generate_v4()"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLink{}
var _ convert.Equaler = (*WorkItemLink)(nil)

// Equal returns true if two WorkItemLink objects are equal; otherwise false is returned.
func (self WorkItemLink) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLink)
	if !ok {
		return false
	}
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !satoriuuid.Equal(self.ID, other.ID) {
		return false
	}
	if self.Version != other.Version {
		return false
	}
	if self.SourceID != other.SourceID {
		return false
	}
	if self.TargetID != other.TargetID {
		return false
	}
	if self.LinkTypeID != other.LinkTypeID {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link
// cannot be used for the creation of a new work item link.
func (t *WorkItemLink) CheckValidForCreation() error {
	if satoriuuid.Equal(t.LinkTypeID, satoriuuid.Nil) {
		return errors.NewBadParameterError("link_type_id", t.LinkTypeID)
	}
	return nil
}

// ConvertLinkFromModel converts a work item from model to REST representation
func ConvertLinkFromModel(t WorkItemLink) app.WorkItemLink {
	id := t.ID.String()
	var converted = app.WorkItemLink{
		Data: &app.WorkItemLinkData{
			Type: EndpointWorkItemLinks,
			ID:   &id,
			Attributes: &app.WorkItemLinkAttributes{
				Version: &t.Version,
			},
			Relationships: &app.WorkItemLinkRelationships{
				LinkType: &app.RelationWorkItemLinkType{
					Data: &app.RelationWorkItemLinkTypeData{
						Type: EndpointWorkItemLinkTypes,
						ID:   t.LinkTypeID.String(),
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
func ConvertLinkToModel(in app.WorkItemLink, out *WorkItemLink) error {
	attrs := in.Data.Attributes
	rel := in.Data.Relationships
	var err error

	if in.Data.ID != nil {
		id, err := satoriuuid.FromString(*in.Data.ID)
		if err != nil {
			//log.Printf("Error when converting %s to UUID: %s", *in.Data.ID, err.Error())
			// treat as not found: clients don't know it must be a UUID
			return errors.NewNotFoundError("work item link", id.String())
		}
		out.ID = id
	}

	if in.Data.Type != EndpointWorkItemLinks {
		return errors.NewBadParameterError("data.type", in.Data.Type).Expected(EndpointWorkItemLinks)
	}

	if attrs != nil {
		if attrs.Version != nil {
			out.Version = *attrs.Version
		}
	}

	if rel != nil && rel.LinkType != nil && rel.LinkType.Data != nil {
		d := rel.LinkType.Data
		// If the the link category is not nil, it MUST be "workitemlinktypes"
		if d.Type != EndpointWorkItemLinkTypes {
			return errors.NewBadParameterError("data.relationships.link_type.data.type", d.Type).Expected(EndpointWorkItemLinkTypes)
		}
		// The the link type id MUST NOT be empty
		if d.ID == "" {
			return errors.NewBadParameterError("data.relationships.link_type.data.id", d.ID)
		}
		if out.LinkTypeID, err = satoriuuid.FromString(d.ID); err != nil {
			//log.Printf("Error when converting %s to UUID: %s", in.Data.ID, err.Error())
			// treat as not found: clients don't know it must be a UUID
			return errors.NewNotFoundError("work item link type", d.ID)
		}
	}

	if rel != nil && rel.Source != nil && rel.Source.Data != nil {
		d := rel.Source.Data
		// If the the source type is not nil, it MUST be "workitems"
		if d.Type != EndpointWorkItems {
			return errors.NewBadParameterError("data.relationships.source.data.type", d.Type).Expected(EndpointWorkItems)
		}
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
		if d.Type != EndpointWorkItems {
			return errors.NewBadParameterError("data.relationships.target.data.type", d.Type).Expected(EndpointWorkItems)
		}
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
