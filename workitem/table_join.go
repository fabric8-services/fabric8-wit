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
//     LEFT JOIN iterations iter ON fields@> concat('{"system_iteration": "', iter.ID, '"}')::jsonb
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

	// TableName specifies the foreign table upon which to join. This can also
	// be subselect on its own.
	TableName string // e.g. "iterations"

	// TableAlias allows us to specify an alias for the table to be used in the
	// WHERE clause.
	TableAlias string // e.g. "iter"

	// On is the ON part of the JOIN.
	On string // e.g. `fields@> concat('{"system_iteration": "', iter.ID, '"}')::jsonb`

	// Where defines what condition to place in the main WHERE clause of the
	// final query.
	Where string

	// PrefixActivators can hold a number of prefix strings that cause this join
	// object to be activated.
	PrefixActivators []string // e.g. []string{"iteration."}

	// AllowedColumns specified all fields that are allowed to be queried from
	// the foreign table. When empty, all columns are allowed.
	AllowedColumns []string // e.g. ["name"].

	// DisallowedColumns specified all fields that are not allowed to be queried
	// from the foreign table. When empty, all columns are allowed.
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

	// All prefixes in the map keys that would normally be handled by this join
	// will be handled by the join specified here (if any).
	DelegateTo map[string]*TableJoin

	// TODO(kwk): Maybe introduce a column mapping table here: ColumnMapping map[string]string
}

// Validate returns nil if the join is active and all the fields handled by this
// join do exist in the joined table; otherwise an error is returned.
func (j TableJoin) Validate(db *gorm.DB) error {
	dialect := db.Dialect()
	dialect.SetDB(db.CommonDB())
	if j.Active {
		for _, f := range j.HandledFields {
			var allowed bool
			for _, allowedColumn := range j.AllowedColumns {
				if allowedColumn == f {
					allowed = true
				}
			}
			if !allowed {
				if !dialect.HasColumn(j.TableName, f) {
					return errs.Errorf(`table "%s" has no column "%s"`, j.TableName, f)
				}
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
	return fmt.Sprintf(`LEFT JOIN %s "%s" ON %s`, j.TableName, j.TableAlias, j.On)
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
			break
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

	// Check if this field should be handled by another one table join
	delegator := j
	for prefix, dele := range j.DelegateTo {
		if strings.HasPrefix(fieldName, prefix) {
			if dele == nil {
				return "", errs.Errorf(`delegated join "%s" for field "%s" must not point to nil`, prefix, fieldName)
			}
			delegator = dele
			// ensure the other join is active (just to be safe) if it
			// wasn't activated yet.
			delegator.Active = true
			break
		}
	}

	// if no columns are explicitly allowed, then this column is allowed by
	// default.
	columnIsAllowed := (delegator.AllowedColumns == nil || len(delegator.AllowedColumns) == 0)
	for _, c := range delegator.AllowedColumns {
		if c == col {
			columnIsAllowed = true
			break
		}
	}
	// check if a column is explicitly disallowed
	for _, c := range delegator.DisallowedColumns {
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
	delegator.HandledFields = append(delegator.HandledFields, col)

	return Column(delegator.TableAlias, col), nil
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

// GetOrderdActivatedJoins returns a slice of activated joins in a proper order,
// beginning with the join that activates no other join, and ending with the
// join that activates another join but isn't activated by another join itself.
func (joins *TableJoinMap) GetOrderdActivatedJoins() ([]*TableJoin, error) {
	if err := joins.ActivateRequiredJoins(); err != nil {
		return nil, errs.Wrap(err, "failed to get activate required joins")
	}

	orderer := activationOrderer{
		m:              *joins,
		alreadyVisited: map[*TableJoin]struct{}{},
	}

	for name := range *joins {
		if err := orderer.visitDepthFirst(name); err != nil {
			return nil, errs.Wrapf(err, `failed to visit "%s" join`, name)
		}
	}
	return orderer.orderedActivatedJoins, nil
}

type activationOrderer struct {
	m                     TableJoinMap
	alreadyVisited        map[*TableJoin]struct{}
	orderedActivatedJoins []*TableJoin
}

func (o *activationOrderer) visitDepthFirst(name string) error {
	j, ok := o.m[name]
	if !ok {
		return errs.Errorf(`join "%s" not found`, name)
	}

	_, alreadyVisited := o.alreadyVisited[j]
	if alreadyVisited {
		return nil
	}

	o.alreadyVisited[j] = struct{}{}

	if !j.Active {
		return nil
	}

	for _, subJoinName := range j.ActivateOtherJoins {
		if err := o.visitDepthFirst(subJoinName); err != nil {
			return errs.Errorf(`failed to visit "%s" join`, subJoinName)
		}
	}

	if o.orderedActivatedJoins == nil {
		o.orderedActivatedJoins = []*TableJoin{}
	}
	o.orderedActivatedJoins = append(o.orderedActivatedJoins, j)

	return nil
}
