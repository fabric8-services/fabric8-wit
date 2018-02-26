package workitem

import (
	"fmt"
	"strings"

	errs "github.com/pkg/errors"
)

// A TableJoin helps to construct a query like this:
//
//   SELECT *
//     FROM workitems
//     JOIN iterations iter ON fields@> concat('{"system.iteration": "', iter.ID, '"}')::jsonb
//     WHERE iter.name = "foo"
//
// With the prefix triggers we can identify if a certain field expression points
// at data from a joined table. By default there are no restrictions on what can
// be queried in the joined table but if you fill the allowed/disallowed columns
// arrays you can explicitly allow or disallow columns to be queried. The names
// in the allowed/disalowed columns are those of the table.
type TableJoin struct {
	Active            bool     // true if this table join is used
	TableName         string   // e.g. "iterations"
	TableAlias        string   // e.g. "iter"
	On                string   // e.g. `fields@> concat('{"system.iteration": "', iter.ID, '"}')::jsonb`
	PrefixTriggers    []string // e.g. "iteration."
	AllowedColumns    []string // e.g. ["name"]. When empty all columns are allowed.
	DisallowedColumns []string // e.g. ["created_at"]. When empty all columns are allowed.

	// TODO(kwk): Maybe introduce a column mapping table here: ColumnMapping map[string]string
}

// Activate tells the search engine to actually use this join information;
// otherwise it won't be used.
func (j *TableJoin) Activate() {
	j.Active = true
}

// IsActive returns true if this table join was activated; otherwise false is
// returned.
func (j TableJoin) IsActive() bool {
	return j.Active
}

// JoinOnJSONField returns the ON part of an SQL JOIN for the given fields
func JoinOnJSONField(jsonField, foreignCol string) string {
	return fmt.Sprintf(`fields@> concat('{"%[1]s": "', %[2]s, '"}')::jsonb`, jsonField, foreignCol)
}

// String implements Stringer interface
func (j TableJoin) String() string {
	return "JOIN " + j.TableName + " " + j.TableAlias + " ON " + j.On
}

// HandlesFieldName returns true if the given field name should be handled by
// this table join.
func (j *TableJoin) HandlesFieldName(fieldName string) bool {
	for _, t := range j.PrefixTriggers {
		if strings.HasPrefix(fieldName, t) {
			return true
		}
	}
	return false
}

// TranslateFieldName returns a non-empty string if the given field name has the
// prefix specified by the table join and if the field is allowed to be queried;
// otherwise it returns an empty string.
func (j *TableJoin) TranslateFieldName(fieldName string) (string, error) {
	if !j.HandlesFieldName(fieldName) {
		return "", errs.Errorf(`field name "%s" not handled by this table join`, fieldName)
	}

	// Ensure this join is active
	j.Activate()

	var prefix string
	for _, t := range j.PrefixTriggers {
		if strings.HasPrefix(fieldName, t) {
			prefix = t
		}
	}
	col := strings.TrimPrefix(fieldName, prefix)
	col = strings.TrimSpace(col)
	if col == "" {
		return "", errs.Errorf(`field name "%s" contains an empty column name after prefix "%s"`, fieldName, prefix)
	}
	if strings.Contains(col, "'") {
		// beware of injection, it's a reasonable restriction for field names,
		// make sure it's not allowed when creating wi types
		return "", errs.Errorf(`single quote not allowed in field name: "%s"`, col)
	}

	// now we have the final column name

	// if no columns are explicitly allowed, then this column is allowed by
	// default.
	columnIsAllowed := (j.AllowedColumns == nil || len(j.AllowedColumns) == 0)
	for _, c := range j.AllowedColumns {
		if c == col {
			columnIsAllowed = true
			break
		}
	}
	// check if a column is explicitly disallowed
	for _, c := range j.DisallowedColumns {
		if c == col {
			columnIsAllowed = false
			break
		}
	}
	if !columnIsAllowed {
		return "", errs.Errorf("column is not allowed: %s", col)
	}
	return j.TableAlias + "." + col, nil
}
