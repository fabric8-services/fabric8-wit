package models

import (
	"github.com/almighty/almighty-core/app"
	satoriuuid "github.com/satori/go.uuid"
)

type Project struct {
	ID      satoriuuid.UUID
	Version int
	Name    string
}

func (p *Project) ConvertFromModel() *app.ProjectData {
	return &app.ProjectData{
		ID:   p.ID.String(),
		Type: "projects",
		Attributes: &app.ProjectAttributes{
			Name: &p.Name,
		},
	}
}
