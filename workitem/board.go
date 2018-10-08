package workitem

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Board represents the board configuration.
type Board struct {
	gormsupport.Lifecycle `json:"lifecycle"`
	ID                    uuid.UUID     `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id"`
	SpaceTemplateID       uuid.UUID     `sql:"type:uuid" json:"space_template_id"`
	Name                  string        `json:"name"`
	Description           string        `json:"description"`
	Columns               []BoardColumn `gorm:"-" json:"columns,omitempty"`
	Context               string        `json:"context"`
	ContextType           string        `json:"context_type"`
}

// TableName implements gorm.tabler
func (wib Board) TableName() string {
	return "work_item_boards"
}

// Ensure Board implements the Equaler interface
var _ convert.Equaler = Board{}
var _ convert.Equaler = (*Board)(nil)

// Equal returns true if two Board objects are equal; otherwise false is returned.
func (wib Board) Equal(u convert.Equaler) bool {
	other, ok := u.(Board)
	if !ok {
		return false
	}
	if wib.ID != other.ID {
		return false
	}
	if wib.SpaceTemplateID != other.SpaceTemplateID {
		return false
	}
	if !convert.CascadeEqual(wib.Lifecycle, other.Lifecycle) {
		return false
	}
	if wib.Name != other.Name {
		return false
	}
	if wib.Description != other.Description {
		return false
	}
	if wib.Context != other.Context {
		return false
	}
	if wib.ContextType != other.ContextType {
		return false
	}
	if len(wib.Columns) != len(other.Columns) {
		return false
	}
	for i := range wib.Columns {
		if !convert.CascadeEqual(wib.Columns[i], other.Columns[i]) {
			return false
		}
	}
	return true
}

// EqualValue implements convert.Equaler
func (wib Board) EqualValue(u convert.Equaler) bool {
	other, ok := u.(Board)
	if !ok {
		return false
	}
	wib.Lifecycle = other.Lifecycle
	return wib.Equal(u)
}

// GetETagData returns the field values to use to generate the ETag
func (wib Board) GetETagData() []interface{} {
	return []interface{}{wib.ID, wib.UpdatedAt}
}

// GetLastModified returns the last modification time
func (wib Board) GetLastModified() time.Time {
	return wib.UpdatedAt
}

// BoardColumn represents a column in a board.
type BoardColumn struct {
	gormsupport.Lifecycle `json:"lifecycle"`
	ID                    uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key" json:"id"`
	BoardID               uuid.UUID `sql:"type:uuid" json:"board_id"`
	Name                  string    `json:"name"`
	Order                 int       `json:"order" gorm:"column:column_order"`
	TransRuleKey          string    `json:"trans_rule_key"`
	TransRuleArgument     string    `json:"trans_rule_argument"` // TODO: this is a JSON, not a string
}

// TableName implements gorm.tabler
func (wibc BoardColumn) TableName() string {
	return "work_item_board_columns"
}

// Ensure BoardColumn implements the Equaler interface
var _ convert.Equaler = BoardColumn{}
var _ convert.Equaler = (*BoardColumn)(nil)

// Equal returns true if two BoardColumn objects are equal; otherwise false is returned.
func (wibc BoardColumn) Equal(u convert.Equaler) bool {
	other, ok := u.(BoardColumn)
	if !ok {
		return false
	}
	if wibc.ID != other.ID {
		return false
	}
	if wibc.BoardID != other.BoardID {
		return false
	}
	if !convert.CascadeEqual(wibc.Lifecycle, other.Lifecycle) {
		return false
	}
	if wibc.Name != other.Name {
		return false
	}
	if wibc.Order != other.Order {
		return false
	}
	if wibc.TransRuleKey != other.TransRuleKey {
		return false
	}
	if wibc.TransRuleArgument != other.TransRuleArgument {
		return false
	}
	return true
}

// EqualValue implements convert.Equaler interface
func (wibc BoardColumn) EqualValue(u convert.Equaler) bool {
	other, ok := u.(BoardColumn)
	if !ok {
		return false
	}
	wibc.Lifecycle = other.Lifecycle
	return wibc.Equal(u)
}
