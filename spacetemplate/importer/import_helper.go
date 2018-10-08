package importer

import (
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/spacetemplate"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/ghodss/yaml"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// ImportHelper is a type to allow creation of space templates from a YAML file.
type ImportHelper struct {
	Template spacetemplate.SpaceTemplate   `json:"space_template"`
	WITs     []*workitem.WorkItemType      `gorm:"-" json:"work_item_types,omitempty"`
	WILTs    []*link.WorkItemLinkType      `gorm:"-" json:"work_item_link_types,omitempty"`
	WITGs    []*workitem.WorkItemTypeGroup `gorm:"-" json:"work_item_type_groups,omitempty"`
	WIBs     []*workitem.Board             `gorm:"-" json:"work_item_boards,omitempty"`
}

// Validate ensures that all inner-document references of the given space
// template are fine.
func (s *ImportHelper) Validate() error {
	// validate nested space template
	if err := s.Template.Validate(); err != nil {
		return errs.Wrap(err, "failed to validate space template")
	}

	// Ensure all artifacts have the correct space template ID set and are
	// valid
	for _, wit := range s.WITs {
		if wit.SpaceTemplateID != s.Template.ID {
			return errors.NewBadParameterError("work item types's space template ID", wit.SpaceTemplateID.String()).Expected(s.Template.ID.String())
		}
		if err := wit.Validate(); err != nil {
			return errs.Wrapf(err, `failed to validate work item type "%s" (ID=%s)`, wit.Name, wit.ID)
		}
	}
	for _, wilt := range s.WILTs {
		if wilt.SpaceTemplateID != s.Template.ID {
			return errors.NewBadParameterError("work item link type's space template ID", wilt.SpaceTemplateID.String()).Expected(s.Template.ID.String())
		}
	}
	for _, witg := range s.WITGs {
		if witg.SpaceTemplateID != s.Template.ID {
			return errors.NewBadParameterError("work item type group's space template ID", witg.SpaceTemplateID.String()).Expected(s.Template.ID.String())
		}
	}
	for _, wibs := range s.WIBs {
		if wibs.SpaceTemplateID != s.Template.ID {
			return errors.NewBadParameterError("work item board's space template ID", wibs.SpaceTemplateID.String()).Expected(s.Template.ID.String())
		}
	}

	return nil
}

// String convert a parsed template into a string in YAML format
func (s ImportHelper) String() string {
	copy := s
	bytes, err := yaml.Marshal(copy)
	if err != nil {
		log.Info(nil, map[string]interface{}{
			"err": err,
		}, "failed to marshal space template to YAML")
		return ""
	}
	return string(bytes)
}

// FromString parses a given string into a parsed template object and validates
// it.
func FromString(templ string) (*ImportHelper, error) {
	var s ImportHelper
	if err := yaml.Unmarshal([]byte(templ), &s); err != nil {
		log.Info(nil, map[string]interface{}{
			"template": templ,
			"err":      err,
		}, "failed to unmarshal YAML space template")
		return nil, errs.Wrapf(err, "failed to parse YAML space template: \n%s", templ)
	}
	// If the space template has no ID, create one on the fly
	if uuid.Equal(s.Template.ID, uuid.Nil) {
		s.Template.ID = uuid.NewV4()
	}
	// update all refs to this ID
	s.SetID(s.Template.ID)
	if err := s.Validate(); err != nil {
		return nil, errs.Wrap(err, "failed to validate space template")
	}
	return &s, nil
}

// SetID updates the space templates IDs and updates the references to that ID.
func (s *ImportHelper) SetID(id uuid.UUID) {
	s.Template.ID = id
	for _, wit := range s.WITs {
		wit.SpaceTemplateID = s.Template.ID
	}
	for _, wilt := range s.WILTs {
		wilt.SpaceTemplateID = s.Template.ID
	}
	for _, witg := range s.WITGs {
		witg.SpaceTemplateID = s.Template.ID
	}
	for _, wib := range s.WIBs {
		wib.SpaceTemplateID = s.Template.ID
	}
}

// Ensure ImportHelper implements the Equaler interface
var _ convert.Equaler = ImportHelper{}
var _ convert.Equaler = (*ImportHelper)(nil)

// Equal returns true if two ImportHelper objects are equal; otherwise false is
// returned.
func (s ImportHelper) Equal(u convert.Equaler) bool {
	other, ok := u.(ImportHelper)
	if !ok {
		return false
	}
	// test nested space template on equality
	if !convert.CascadeEqual(s.Template, other.Template) {
		return false
	}
	if len(s.WITs) != len(other.WITs) {
		return false
	}
	for k := range s.WITs {
		if other.WITs[k] == nil {
			return false
		}
		if !convert.CascadeEqual(s.WITs[k], *other.WITs[k]) {
			return false
		}
	}
	if len(s.WILTs) != len(other.WILTs) {
		return false
	}
	for k := range s.WILTs {
		if other.WILTs[k] == nil {
			return false
		}
		if !convert.CascadeEqual(s.WILTs[k], *other.WILTs[k]) {
			return false
		}
	}
	if len(s.WITGs) != len(other.WITGs) {
		return false
	}
	for k := range s.WITGs {
		if other.WITGs[k] == nil {
			return false
		}
		if !convert.CascadeEqual(s.WITGs[k], *other.WITGs[k]) {
			return false
		}
	}
	if len(s.WIBs) != len(other.WIBs) {
		return false
	}
	for k := range s.WIBs {
		if other.WIBs[k] == nil {
			return false
		}
		if !convert.CascadeEqual(s.WIBs[k], *other.WIBs[k]) {
			return false
		}
	}
	return true
}

// EqualValue implements convert.Equaler
func (s ImportHelper) EqualValue(u convert.Equaler) bool {
	return s.Equal(u)
}

// BaseTemplate returns the base template
func BaseTemplate() (*ImportHelper, error) {
	bs, err := spacetemplate.Asset("base.yaml")
	if err != nil {
		return nil, errs.Wrap(err, "failed to load base template")
	}
	s, err := FromString(string(bs))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	s.SetID(spacetemplate.SystemBaseTemplateID)
	return s, nil
}

// LegacyTemplate returns the legacy template as it is known to the system
func LegacyTemplate() (*ImportHelper, error) {
	bs, err := spacetemplate.Asset("legacy.yaml")
	if err != nil {
		return nil, errs.Wrap(err, "failed to load legacy template")
	}
	s, err := FromString(string(bs))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	s.SetID(spacetemplate.SystemLegacyTemplateID)
	return s, nil
}

// ScrumTemplate returns the scrum template as it is known to the system
func ScrumTemplate() (*ImportHelper, error) {
	bs, err := spacetemplate.Asset("scrum.yaml")
	if err != nil {
		return nil, errs.Wrap(err, "failed to load scrum template")
	}
	s, err := FromString(string(bs))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	s.SetID(spacetemplate.SystemScrumTemplateID)
	return s, nil
}

// AgileTemplate returns the agile template as it is known to the system
func AgileTemplate() (*ImportHelper, error) {
	bs, err := spacetemplate.Asset("agile.yaml")
	if err != nil {
		return nil, errs.Wrap(err, "failed to load agile template")
	}
	s, err := FromString(string(bs))
	if err != nil {
		return nil, errs.WithStack(err)
	}
	s.SetID(spacetemplate.SystemAgileTemplateID)
	return s, nil
}
