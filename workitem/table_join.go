package workitem

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

// A TableJoin helps to construct a query like this:
//
//   SELECT *
//     FROM workitems
//     LEFT JOIN iterations iter ON fields@> concat('{"system.iteration": "', iter.ID, '"}')::jsonb
//     WHERE iter.name = "foo"
//
// With the prefix activators we can identify if a certain field expression
// points at data from a joined table. By default there are no restrictions on
// what can be queried in the joined table but if you fill the
// allowed/disallowed columns arrays you can explicitly allow or disallow
// columns to be queried. The names in the allowed/disalowed columns are those
// of the foreign (aka joined) table.
type TableJoin struct {
	Active bool // true if this table join is used

	// TableName specifies the foreign table upon which to join
	TableName string // e.g. "iterations"

	// TableAlias allows us to specify an alias for the table to be used in the
	// WHERE clause.
	TableAlias string // e.g. "iter"

	// On is the ON part of the JOIN.
	On string // e.g. `fields@> concat('{"system.iteration": "', iter.ID, '"}')::jsonb`

	// PrefixActivators can hold a number of prefix strings that cause this join
	// object to be activated.
	PrefixActivators []string // e.g. []string{"iteration."}

	// disallowedColumns specified all fields that are allowed to be queried
	// from the foreign table. When empty all columns are allowed.
	AllowedColumns []string // e.g. ["name"].

	// DisallowedColumns specified all fields that are not allowed to be queried
	// from the foreign table. When empty all columns are allowed.
	DisallowedColumns []string // e.g. ["created_at"].

	// HandledFields contains those fields that were found to reference this
	// table join. It is later used by Validate() to find out if a field name
	// exists in the database.
	HandledFields []string // e.g. []string{"name", "created_at", "foobar"}

	// ActivateOtherJoins is useful when you make complex joins over mutliple
	// tables. Just put in the names of the table join keys in here that you
	// would like to activate as well. See DefaultTypeGroups() for how the map
	// looks like. If you ask for "A" and that requires "B", then "B" is also
	// added automatically.
	ActivateOtherJoins []string

	// TODO(kwk): Maybe introduce a column mapping table here: ColumnMapping map[string]string
}

// Validate returns nil if the join is active and all the fields handled by this
// join do exist in the joined table; otherwise an error is returned.
func (j TableJoin) Validate(db *gorm.DB) error {
	dialect := db.Dialect()
	dialect.SetDB(db.CommonDB())
	if j.Active {
		for _, f := range j.HandledFields {
			if !dialect.HasColumn(j.TableName, f) {
				return errs.Errorf(`table "%s" has no column "%s"`, j.TableName, f)
			}
		}
	}
	return nil
}

// JoinOnJSONField returns the ON part of an SQL JOIN for the given fields
func JoinOnJSONField(jsonField, foreignCol string) string {
	return fmt.Sprintf(`%[1]s @> concat('{"%[2]s": "', %[3]s, '"}')::jsonb`, Column(WorkItemStorage{}.TableName(), "fields"), jsonField, foreignCol)
}

// GetJoinExpression returns the SQL JOIN expression for this table join.
func (j TableJoin) GetJoinExpression() string {
	return fmt.Sprintf(`LEFT JOIN "%s" "%s" ON %s`, j.TableName, j.TableAlias, j.On)
}

// HandlesFieldName returns true if the given field name should be handled by
// this table join.
func (j *TableJoin) HandlesFieldName(fieldName string) bool {
	for _, t := range j.PrefixActivators {
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
	j.Active = true

	var prefix string
	for _, t := range j.PrefixActivators {
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

	// Remember what foreign columns where queried for. Later we can use
	// Validate() to see if those columns do exist or not.
	j.HandledFields = append(j.HandledFields, col)

	return Column(j.TableAlias, col), nil
}

// TableJoinMap is used to store join in the expression compiler
type TableJoinMap map[string]*TableJoin

// ActivateRequiredJoins recursively walks over all given joins potentially
// multiple times and activates all other required joins.
func (joins *TableJoinMap) ActivateRequiredJoins() error {
	for k, join := range *joins {
		if !join.Active {
			continue
		}

		for _, name := range join.ActivateOtherJoins {
			other, exists := (*joins)[name]
			if !exists {
				return errs.Errorf(`join "%s" not found for "%s" join`, name, k)
			}

			// Check if dependend join is already active
			if other.Active {
				continue
			}

			other.Active = true
			if err := joins.ActivateRequiredJoins(); err != nil {
				return errs.Wrapf(err, `failed to activate required joins for "%s" join`, k)
			}
		}
	}
	return nil
}
