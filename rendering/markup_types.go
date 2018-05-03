package rendering

import (
	"database/sql"
	"database/sql/driver"

	"github.com/fabric8-services/fabric8-wit/errors"
)

// Markup determines the language in which a markup field is written.
type Markup string

const (
	// SystemMarkupDefault Default value
	SystemMarkupDefault Markup = SystemMarkupPlainText
	// SystemMarkupPlainText plain text
	SystemMarkupPlainText Markup = "PlainText"
	// SystemMarkupMarkdown Markdown
	SystemMarkupMarkdown Markup = "Markdown"
	// SystemMarkupJiraWiki JIRA Wiki
	SystemMarkupJiraWiki Markup = "JiraWiki"
)

// String implements the Stringer interface
func (t Markup) String() string { return string(t) }

// Scan implements the https://golang.org/pkg/database/sql/#Scanner interface
// See also https://stackoverflow.com/a/25374979/835098
// See also https://github.com/jinzhu/gorm/issues/302#issuecomment-80566841
func (t *Markup) Scan(value interface{}) error { *t = Markup(value.([]byte)); return nil }

// Value implements the https://golang.org/pkg/database/sql/driver/#Valuer interface
func (t Markup) Value() (driver.Value, error) { return string(t), nil }

// Ensure Markup implements the Scanner and Valuer interfaces
var _ sql.Scanner = (*Markup)(nil)
var _ driver.Valuer = (*Markup)(nil)

// CheckValid returns nil if the given markup is valid; otherwise a
// BadParameterError is returned.
func (t Markup) CheckValid() error {
	switch t {
	case SystemMarkupPlainText, SystemMarkupMarkdown /*, SystemMarkupJiraWiki*/ :
		return nil
	default:
		return errors.NewBadParameterError("markup", t).Expected(SystemMarkupPlainText + "|" + SystemMarkupMarkdown /*+ "|" + SystemMarkupJiraWiki*/)
	}
}

// NilSafeGetMarkup returns the given markup itself but only if it is not nil
// and valid; otherwise the default markup is returned (SystemMarkupDefault).
func (t *Markup) NilSafeGetMarkup() Markup {
	res := SystemMarkupDefault
	if t == nil {
		return res
	}
	if t.CheckValid() != nil {
		return res
	}
	return *t
}
