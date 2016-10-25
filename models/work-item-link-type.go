package models

import (
	"github.com/almighty/almighty-core/app"
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

const (
	TopologyNetwork         = "network"
	TopologyDirectedNetwork = "directed_network"
	TopologyDependency      = "dependency"
	TopologyTree            = "tree"
)

// WorkItemLinkType represents the type of a work item link as it is stored in the db
type WorkItemLinkType struct {
	gormsupport.Lifecycle
	// ID
	ID satoriuuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Name is the unique name of this work item link category.
	Name string
	// Description is an optional description of the work item link category
	Description *string
	// Version for optimistic concurrency control
	Version  int
	Topology string // Valid values: network, directed_network, dependency, tree

	SourceTypeName string
	TargetTypeName string

	ForwardName string
	ReverseName string

	LinkCategoryID satoriuuid.UUID
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkType{}
var _ convert.Equaler = (*WorkItemLinkType)(nil)

// Equal returns true if two WorkItemLinkType objects are equal; otherwise false is returned.
func (self WorkItemLinkType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkType)
	if !ok {
		return false
	}
	if !self.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !satoriuuid.Equal(self.ID, other.ID) {
		return false
	}
	if self.Name != other.Name {
		return false
	}
	if self.Version != other.Version {
		return false
	}
	if self.Description != nil && other.Description != nil {
		if *self.Description != *other.Description {
			return false
		}
	} else {
		if self.Description != other.Description {
			return false
		}
	}
	if self.Topology != other.Topology {
		return false
	}
	if self.SourceTypeName != other.SourceTypeName {
		return false
	}
	if self.TargetTypeName != other.TargetTypeName {
		return false
	}
	if self.ForwardName != other.ForwardName {
		return false
	}
	if self.ReverseName != other.ReverseName {
		return false
	}
	if !satoriuuid.Equal(self.LinkCategoryID, other.LinkCategoryID) {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link type
// cannot be used for the creation of a new work item link type.
func (t *WorkItemLinkType) CheckValidForCreation() error {
	if t.Name == "" {
		return NewBadParameterError("name", t.Name)
	}
	if t.SourceTypeName == "" {
		return NewBadParameterError("source_type_name", t.SourceTypeName)
	}
	if t.TargetTypeName == "" {
		return NewBadParameterError("target_type_name", t.TargetTypeName)
	}
	if t.ForwardName == "" {
		return NewBadParameterError("forward_name", t.ForwardName)
	}
	if t.ReverseName == "" {
		return NewBadParameterError("reverse_name", t.ReverseName)
	}
	if err := CheckValidTopology(t.Topology); err != nil {
		return err
	}
	if t.LinkCategoryID == satoriuuid.Nil {
		return NewBadParameterError("link_category_id", t.LinkCategoryID)
	}
	return nil
}

// CheckValidTopology returns nil if the given topology is valid;
// otherwise a BadParameterError is returned.
func CheckValidTopology(t string) error {
	if t != TopologyNetwork && t != TopologyDirectedNetwork && t != TopologyDependency && t != TopologyTree {
		return NewBadParameterError("topolgy", t).Expected(TopologyNetwork + "|" + TopologyDirectedNetwork + "|" + TopologyDependency + "|" + TopologyTree)
	}
	return nil
}

// ConvertLinkTypeFromModel converts a work item link type from model to REST representation
func ConvertLinkTypeFromModel(t *WorkItemLinkType) app.WorkItemLinkType {
	id := t.ID.String()
	var converted = app.WorkItemLinkType{
		Data: &app.WorkItemLinkTypeData{
			Type: EndpointWorkItemLinkTypes,
			ID:   &id,
			Attributes: &app.WorkItemLinkTypeAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
				ForwardName: &t.ForwardName,
				ReverseName: &t.ReverseName,
				Topology:    &t.Topology,
			},
			Relationships: &app.WorkItemLinkTypeRelationships{
				LinkCategory: &app.RelationWorkItemLinkCategory{
					Data: &app.RelationWorkItemLinkCategoryData{
						Type: EndpointWorkItemLinkCategories,
						ID:   t.LinkCategoryID.String(),
					},
				},
				SourceType: &app.RelationWorkItemType{
					Data: &app.RelationWorkItemTypeData{
						Type: EndpointWorkItemTypes,
						ID:   t.SourceTypeName,
					},
				},
				TargetType: &app.RelationWorkItemType{
					Data: &app.RelationWorkItemTypeData{
						Type: EndpointWorkItemTypes,
						ID:   t.TargetTypeName,
					},
				},
			},
		},
	}
	return converted
}

// ConvertLinkTypeToModel converts the incoming app representation of a work item link type to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
func ConvertLinkTypeToModel(in *app.WorkItemLinkType, out *WorkItemLinkType) error {
	attrs := in.Data.Attributes
	rel := in.Data.Relationships
	var err error

	if in.Data.ID != nil {
		id, err := satoriuuid.FromString(*in.Data.ID)
		if err != nil {
			//log.Printf("Error when converting %s to UUID: %s", *in.Data.ID, err.Error())
			// treat as not found: clients don't know it must be a UUID
			return NewNotFoundError("work item link type", id.String())
		}
		out.ID = id
	}

	if in.Data.Type != EndpointWorkItemLinkTypes {
		return NewBadParameterError("data.type", in.Data.Type).Expected(EndpointWorkItemLinkTypes)
	}

	if attrs != nil {
		// If the name is not nil, it MUST NOT be empty
		if attrs.Name != nil {
			if *attrs.Name == "" {
				return NewBadParameterError("data.attributes.name", *attrs.Name)
			}
			out.Name = *attrs.Name
		}

		if attrs.Description != nil {
			out.Description = attrs.Description
		}

		if attrs.Version != nil {
			out.Version = *attrs.Version
		}

		// If the forwardName is not nil, it MUST NOT be empty
		if attrs.ForwardName != nil {
			if *attrs.ForwardName == "" {
				return NewBadParameterError("data.attributes.forward_name", *attrs.ForwardName)
			}
			out.ForwardName = *attrs.ForwardName
		}

		// If the ReverseName is not nil, it MUST NOT be empty
		if attrs.ReverseName != nil {
			if *attrs.ReverseName == "" {
				return NewBadParameterError("data.attributes.reverse_name", *attrs.ReverseName)
			}
			out.ReverseName = *attrs.ReverseName
		}

		if attrs.Topology != nil {
			if err := CheckValidTopology(*attrs.Topology); err != nil {
				return err
			}
			out.Topology = *attrs.Topology
		}
	}

	if rel != nil && rel.LinkCategory != nil && rel.LinkCategory.Data != nil {
		d := rel.LinkCategory.Data
		// If the the link category is not nil, it MUST be "workitemlinkcategories"
		if d.Type != EndpointWorkItemLinkCategories {
			return NewBadParameterError("data.relationships.link_category.data.type", d.Type).Expected(EndpointWorkItemLinkCategories)
		}
		// The the link category MUST NOT be empty
		if d.ID == "" {
			return NewBadParameterError("data.relationships.link_category.data.id", d.ID)
		}
		out.LinkCategoryID, err = satoriuuid.FromString(d.ID)
		if err != nil {
			//log.Printf("Error when converting %s to UUID: %s", in.Data.ID, err.Error())
			// treat as not found: clients don't know it must be a UUID
			return NotFoundError{entity: "work item link category", ID: d.ID}
		}
	}

	if rel != nil && rel.SourceType != nil && rel.SourceType.Data != nil {
		d := rel.SourceType.Data
		// If the the link type is not nil, it MUST be "workitemlinktypes"
		if d.Type != EndpointWorkItemTypes {
			return NewBadParameterError("data.relationships.source_type.data.type", d.Type).Expected(EndpointWorkItemTypes)
		}
		// The the link type MUST NOT be empty
		if d.ID == "" {
			return NewBadParameterError("data.relationships.source_type.data.id", d.ID)
		}
		out.SourceTypeName = d.ID
	}

	if rel != nil && rel.TargetType != nil && rel.TargetType.Data != nil {
		d := rel.TargetType.Data
		// If the the link type is not nil, it MUST be "workitemlinktypes"
		if d.Type != EndpointWorkItemTypes {
			return NewBadParameterError("data.relationships.target_type.data.type", d.Type).Expected(EndpointWorkItemTypes)
		}
		// The the link type MUST NOT be empty
		if d.ID == "" {
			return NewBadParameterError("data.relationships.target_type.data.id", d.ID)
		}
		out.TargetTypeName = d.ID
	}

	return nil
}
