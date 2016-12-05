package project

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/gormsupport"
	satoriuuid "github.com/satori/go.uuid"
)

// Project represents a project on the domain and db layer
type Project struct {
	gormsupport.Lifecycle
	ID      satoriuuid.UUID
	Version int
	Name    string
}

// Converts a project to the API layer representation
func (p *Project) ConvertFromModel() *app.ProjectData {
	return &app.ProjectData{
		ID:   p.ID.String(),
		Type: "projects",
		Attributes: &app.ProjectAttributes{
			Name: &p.Name,
		},
	}
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = Project{}
var _ convert.Equaler = (*Project)(nil)

// Equal returns true if two Project objects are equal; otherwise false is returned.
func (p Project) Equal(u convert.Equaler) bool {
	other, ok := u.(Project)
	if !ok {
		return false
	}
	lfEqual := p.Lifecycle.Equal(other.Lifecycle)
	if !lfEqual {
		return false
	}
	if p.Version != other.Version {
		return false
	}
	if p.Name != other.Name {
		return false
	}
	return true
}
