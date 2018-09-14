package spacetemplate

import (
	"reflect"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Never ever change these UUIDs!!!
var (
	// pre-defined space templates
	SystemLegacyTemplateID        = uuid.FromStringOrNil("929c963a-174c-4c37-b487-272067e88bd4")
	SystemBaseTemplateID          = uuid.FromStringOrNil("1f48b7bf-bc51-4823-8101-9f10039035ba")
	SystemScrumTemplateID         = uuid.FromStringOrNil("cfff59dc-007a-4fa5-acf7-376d5345aef2")
	SystemAgileTemplateID         = uuid.FromStringOrNil("f405fa41-a8bb-46db-8800-2dbe13da1418")
	SystemIssueTrackingTemplateID = uuid.FromStringOrNil("f4a24db4-9376-4777-832b-852e0ce02fd7")
)

// A SpaceTemplate defines is what is stored in the database. See the
// ImportHelper to learn more about how we import space templates using YAML.
type SpaceTemplate struct {
	gormsupport.Lifecycle `json:"lifecycle"`
	ID                    uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id"`
	Version               int       `json:"version"`
	Name                  string    `json:"name"`
	Description           *string   `json:"description,omitempty"`
	CanConstruct          bool      `gorm:"can_construct" json:"can_construct"`
}

// Validate ensures that all inner-document references of the given space
// template are fine.
func (s *SpaceTemplate) Validate() error {
	if s.Name == "" {
		return errors.NewBadParameterError("name", s.Name).Expected("not empty")
	}
	if uuid.Equal(s.ID, uuid.Nil) {
		return errors.NewBadParameterError("template id", s.ID).Expected("non-nil UUID")
	}

	return nil
}

// GetETagData returns the field values to use to generate the ETag
func (s SpaceTemplate) GetETagData() []interface{} {
	return []interface{}{s.ID, s.Version}
}

// GetLastModified returns the last modification time
func (s SpaceTemplate) GetLastModified() time.Time {
	return s.UpdatedAt.Truncate(time.Second)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (s SpaceTemplate) TableName() string {
	return "space_templates"
}

// Ensure SpaceTemplate implements the Equaler interface
var _ convert.Equaler = SpaceTemplate{}
var _ convert.Equaler = (*SpaceTemplate)(nil)

// Equal returns true if two SpaceTemplate objects are equal; otherwise false is
// returned.
func (s SpaceTemplate) Equal(u convert.Equaler) bool {
	other, ok := u.(SpaceTemplate)
	if !ok {
		return false
	}
	if !uuid.Equal(s.ID, other.ID) {
		return false
	}
	if s.Name != other.Name {
		return false
	}
	if s.Version != other.Version {
		return false
	}
	if !convert.CascadeEqual(s.Lifecycle, other.Lifecycle) {
		return false
	}
	if s.CanConstruct != other.CanConstruct {
		return false
	}
	if !reflect.DeepEqual(s.Description, other.Description) {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler
func (s SpaceTemplate) EqualValue(u convert.Equaler) bool {
	other, ok := u.(SpaceTemplate)
	if !ok {
		return false
	}
	s.Version = other.Version
	s.Lifecycle = other.Lifecycle
	return s.Equal(u)
}
