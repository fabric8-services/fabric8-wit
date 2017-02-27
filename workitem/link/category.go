package link

import (
	"github.com/almighty/almighty-core/app"
	convert "github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

const (
	SystemWorkItemLinkCategorySystem = "system"
	SystemWorkItemLinkCategoryUser   = "user"
)

// WorkItemLinkCategory represents the category of a work item link as it is stored in the db
type WorkItemLinkCategory struct {
	gormsupport.Lifecycle
	// ID
	ID satoriuuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Name is the unique name of this work item link category.
	Name string
	// Description is an optional description of the work item link category
	Description *string
	// Version for optimistic concurrency control
	Version int
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkCategory{}
var _ convert.Equaler = (*WorkItemLinkCategory)(nil)

// Equal returns true if two WorkItemLinkCategory objects are equal; otherwise false is returned.
func (c WorkItemLinkCategory) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkCategory)
	if !ok {
		return false
	}
	if !c.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if c.ID != other.ID {
		return false
	}
	if c.Name != other.Name {
		return false
	}
	if c.Version != other.Version {
		return false
	}
	if !strPtrIsNilOrContentIsEqual(c.Description, other.Description) {
		return false
	}
	return true
}

// TableName implements gorm.tabler
func (c WorkItemLinkCategory) TableName() string {
	return "work_item_link_categories"
}

// ConvertLinkCategoryFromModel converts work item link category from model to app representation
func ConvertLinkCategoryFromModel(t WorkItemLinkCategory) app.WorkItemLinkCategorySingle {
	var converted = app.WorkItemLinkCategorySingle{
		Data: &app.WorkItemLinkCategoryData{
			Type: EndpointWorkItemLinkCategories,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
			},
		},
	}
	return converted
}
