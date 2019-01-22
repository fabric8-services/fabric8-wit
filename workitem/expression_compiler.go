package workitem

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fabric8-services/fabric8-wit/criteria"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	jsonAnnotation = "JSON"
)

// Compile takes an expression and compiles it to a where clause for use with
// gorm.DB.Where(). Returns the number of expected parameters for the query and a
// slice of errors if something goes wrong.
func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, joins []*TableJoin, err []error) {
	compiler := newExpressionCompiler()

	criteria.IteratePostOrder(where, bubbleUpJSONContext(&compiler))

	compiled := where.Accept(&compiler)

	c, ok := compiled.(string)
	if !ok {
		c = ""
	}

	// Make sure we don't return all possible joins but only the once that were
	// activated. Returning them as a slice preserves the correct order of
	// joins.
	joins, e := compiler.joins.GetOrderdActivatedJoins()
	if e != nil {
		compiler.err = append(compiler.err, e)
	}

	for _, j := range joins {
		if j.Where == "" {
			continue
		}
		if c != "" {
			c += " AND "
		}
		c += j.Where
	}

	return c, compiler.parameters, joins, compiler.err
}

// mark expression tree nodes that reference json fields
func bubbleUpJSONContext(c *expressionCompiler) func(exp criteria.Expression) bool {
	return func(exp criteria.Expression) bool {
		switch t := exp.(type) {
		case *criteria.FieldExpression:
			_, isJSONField := c.getFieldName(t.FieldName)
			if isJSONField {
				t.SetAnnotation(jsonAnnotation, true)
			}
		case *criteria.EqualsExpression:
			if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
				t.SetAnnotation(jsonAnnotation, true)
			}
		case *criteria.SubstringExpression:
			if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
				t.SetAnnotation(jsonAnnotation, true)
			}
		case *criteria.NotExpression:
			if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
				t.SetAnnotation(jsonAnnotation, true)
			}
		}
		return true
	}
}

// Column returns a proper column name from the given column name in the given
// table.
func Column(table, column string) string {
	return fmt.Sprintf(`"%s"."%s"`, table, column)
}

// fieldMap tells how to resolve struct fields as SQL fields in the work_items
// SQL table.
// NOTE: anything not listed here will be treated as if it is nested inside the
// jsonb "fields" column.
var fieldMap = map[string]string{
	"ID":      "id",
	"Type":    "type",
	"Version": "version",
	"Number":  "number",
	"SpaceID": "space_id",
}

// getFieldName applies any potentially necessary mapping to field names (e.g.
// SpaceID -> space_id) and tells if the field is stored inside the jsonb column
// (last result is true then) or as a normal column.
func (c *expressionCompiler) getFieldName(fieldName string) (mappedFieldName string, isJSONField bool) {
	// If this field name references a joinable table, we will not say that it
	// is a JSON field even though it might contain a dot.
	for _, j := range c.joins {
		if j.HandlesFieldName(fieldName) {
			return fieldName, false
		}
	}

	mappedFieldName, isColumnField := fieldMap[fieldName]
	if isColumnField {
		return Column(WorkItemStorage{}.TableName(), mappedFieldName), false
	}
	// Check if the field name contains an underscore
	if strings.Contains(fieldName, "_") {
		// leave field untouched
		return fieldName, true
	}
	return Column(WorkItemStorage{}.TableName(), fieldName), false
}

// DefaultTableJoins returns the default list of joinable tables used when
// creating a new expression compiler.
var DefaultTableJoins = func() TableJoinMap {
	res := TableJoinMap{
		"iteration": {
			TableName:        "iterations",
			TableAlias:       "iter",
			On:               JoinOnJSONField(SystemIteration, "iter.id") + " AND " + Column("iter", "space_id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id"),
			PrefixActivators: []string{"iteration."},
			AllowedColumns:   []string{"name", "created_at", "number"},
		},
		"area": {
			TableName:        "areas",
			TableAlias:       "ar",
			On:               JoinOnJSONField(SystemArea, "ar.id") + " AND " + Column("ar", "space_id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id"),
			PrefixActivators: []string{"area."},
			AllowedColumns:   []string{"name", "number"},
		},
		"codebase": {
			TableName:        "codebases",
			TableAlias:       "cb",
			On:               JoinOnJSONField(SystemCodebase, "cb.id") + " AND " + Column("cb", "space_id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id"),
			PrefixActivators: []string{"codebase."},
			AllowedColumns:   []string{"url"},
		},
		"work_item_type": {
			TableName:        "work_item_types",
			TableAlias:       "wit",
			On:               "wit.id = " + WorkItemStorage{}.TableName() + ".type",
			PrefixActivators: []string{"wit.", "workitemtype.", "work_item_type.", "type."},
			AllowedColumns:   []string{"name"},
		},
		"creator": {
			TableName:        "users",
			TableAlias:       "creator",
			On:               JoinOnJSONField(SystemCreator, "creator.id"),
			PrefixActivators: []string{"creator.", "author."},
			AllowedColumns:   []string{"full_name"},
		},
		"space": {
			TableName:        "spaces",
			TableAlias:       "space",
			On:               Column("space", "id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id"),
			PrefixActivators: []string{"space."},
			AllowedColumns:   []string{"name"},
		},
		"boardcolumns": {
			TableName:  `(SELECT id colid, board_id id, jsonb_agg(id::text) AS colids FROM ` + BoardColumn{}.TableName() + ` GROUP BY 1,2)`,
			TableAlias: "boardcolumns",
			On: fmt.Sprintf(Column("boardcolumns", "colid")+`::text IN (
				SELECT jsonb_array_elements_text(jsonb_strip_nulls(`+Column(WorkItemStorage{}.TableName(), "fields")+`)->'%s')
			)`, SystemBoardcolumns),
			PrefixActivators: []string{"board."},
			AllowedColumns:   []string{"id"},
		},
		"typegroup": {
			TableName:  "work_item_type_groups",
			TableAlias: "witg",
			On:         fmt.Sprintf(`%s=%s`, Column("witg", "space_template_id"), Column("space", "space_template_id")),
			// In this WHERE clause we access information from the
			// `work_item_type_group_members` table which is why we must
			// delegate from that table to this one.
			Where: fmt.Sprintf(`%s=%s AND %s=%s`,
				Column("witg_members", "type_group_id"), Column("witg", "id"),
				Column(WorkItemStorage{}.TableName(), "type"), Column("witg_members", "work_item_type_id"),
			),
			// This join doesn't specify it's own prefix because we delegate to
			// this join from another table in order to get the correct order of
			// joins.
			// PrefixActivators: []string{"typegroup."},
			AllowedColumns:     []string{"name"},
			ActivateOtherJoins: []string{"space"},
		},
		"label": {
			TableName:  "labels",
			TableAlias: "lbl",
			On: Column("lbl", "space_id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id") + `
		                    AND lbl.id::text IN (
		                        SELECT
						jsonb_array_elements_text(` + Column(WorkItemStorage{}.TableName(), "fields") + `->'system_labels')
					FROM labels)`,
			PrefixActivators: []string{"label."},
			AllowedColumns:   []string{"name"},
		},
		"trackerquery": {
			TableName:        "tracker_queries",
			TableAlias:       "tq",
			On:               JoinOnJSONField(SystemRemoteTrackerID, "tq.id") + " AND " + Column("tq", "space_id") + "=" + Column(WorkItemStorage{}.TableName(), "space_id"),
			PrefixActivators: []string{"trackerquery."},
			AllowedColumns:   []string{"id"},
		},
	}

	res["typegroup_members"] = &TableJoin{
		TableName:          "work_item_type_group_members",
		TableAlias:         "witg_members",
		On:                 Column("witg_members", "type_group_id") + "=" + Column("witg", "id"),
		PrefixActivators:   []string{"typegroup_members.", "typegroup."},
		AllowedColumns:     []string{"work_item_type_id"},
		ActivateOtherJoins: []string{"typegroup"},
		// every field prefixed with .typegroup will be handled by the
		// "typegroup" join
		DelegateTo: map[string]*TableJoin{
			"typegroup.": res["typegroup"],
		},
	}

	// Filter by parent's ID or human-readable Number
	res["parent_link"] = &TableJoin{
		TableName:  "work_item_links",
		TableAlias: "parent_link",
		// importing the link package here to get the link type is currently not
		// possible because of an import cycle
		On: Column("parent_link", "link_type_id") + "= '25C326A7-6D03-4F5A-B23B-86A9EE4171E9' AND " + Column("parent_link", "target_id") + "=" + Column(WorkItemStorage{}.TableName(), "id"),
	}
	res["parent"] = &TableJoin{
		TableName:          WorkItemStorage{}.TableName(),
		TableAlias:         "parent",
		On:                 Column("parent_link", "source_id") + "=" + Column("parent", "id"),
		AllowedColumns:     []string{"id", "number"},
		PrefixActivators:   []string{"parent."},
		ActivateOtherJoins: []string{"parent_link"},
	}

	return res
}

func newExpressionCompiler() expressionCompiler {
	return expressionCompiler{
		parameters: []interface{}{},
		// Define all possible join scenarios here
		joins: DefaultTableJoins(),
	}
}

// expressionCompiler takes an expression and compiles it to a where clause for our gorm models
// implements criteria.ExpressionVisitor
type expressionCompiler struct {
	parameters []interface{} // records the number of parameter expressions encountered
	err        []error       // record any errors found in the expression
	joins      TableJoinMap  // map of table joins keyed by table name
}

// Ensure expressionCompiler implements the ExpressionVisitor interface
var _ criteria.ExpressionVisitor = &expressionCompiler{}
var _ criteria.ExpressionVisitor = (*expressionCompiler)(nil)

// expressionRefersToJoinedData returns true if the given field expression is a
// field expression and  refers to joined data; otherwise false is returned.
func (c *expressionCompiler) expressionRefersToJoinedData(e criteria.Expression) (*TableJoin, bool) {
	switch t := e.(type) {
	case *criteria.FieldExpression:
		for _, j := range c.joins {
			if j.HandlesFieldName(t.FieldName) {
				j.Active = true
				return j, true
			}
		}
	}
	return nil, false
}

// Ensure expressionCompiler implements the ExpressionVisitor interface
var _ criteria.ExpressionVisitor = &expressionCompiler{}
var _ criteria.ExpressionVisitor = (*expressionCompiler)(nil)

// visitor implementation
// the convention is to return nil when the expression cannot be compiled and to append an error to the err field

func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	if strings.Contains(f.FieldName, `"`) {
		c.err = append(c.err, errs.Errorf("field name must not contain double quotes: %s", f.FieldName))
		return nil
	}
	if strings.Contains(f.FieldName, `'`) {
		c.err = append(c.err, errs.Errorf("field name must not contain single quotes: %s", f.FieldName))
		return nil
	}

	mappedFieldName, isJSONField := c.getFieldName(f.FieldName)

	// Check if this field is referencing joinable data
	for _, j := range c.joins {
		if j.HandlesFieldName(mappedFieldName) {
			j.Active = true
			col, err := j.TranslateFieldName(mappedFieldName)
			if err != nil {
				c.err = append(c.err, errs.Wrapf(err, `failed to translate field "%s"`, mappedFieldName))
			}
			return col // e.g. "iter.name"
		}
	}

	if !isJSONField {
		return mappedFieldName
	}
	if strings.Contains(mappedFieldName, "'") {
		// beware of injection, it's a reasonable restriction for field names,
		// make sure it's not allowed when creating wi types
		c.err = append(c.err, errs.Errorf("single quote not allowed in field name: %s", mappedFieldName))
		return nil
	}

	// default to plain json field (e.g. for ID comparisons)
	return Column(WorkItemStorage{}.TableName(), "fields") + ` @> '{"` + mappedFieldName + `"`
}

func (c *expressionCompiler) And(a *criteria.AndExpression) interface{} {
	return c.binary(a, "AND")
}

func (c *expressionCompiler) binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left().Accept(c)
	right := a.Right().Accept(c)
	if left != nil && right != nil {
		l, ok := left.(string)
		if !ok {
			c.err = append(c.err, errs.Errorf("failed to convert left expression to string: %+v", left))
			return nil
		}
		r, ok := right.(string)
		if !ok {
			c.err = append(c.err, errs.Errorf("failed to convert right expression to string: %+v", right))
			return nil
		}
		return "(" + l + " " + op + " " + r + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return nil
}

func (c *expressionCompiler) Or(a *criteria.OrExpression) interface{} {
	return c.binary(a, "OR")
}

func (c *expressionCompiler) Equals(e *criteria.EqualsExpression) interface{} {
	op := "="
	if isInJSONContext(e.Left()) {
		op = ":"
	}
	_, ok := c.expressionRefersToJoinedData(e.Left())
	if ok {
		op = "="
	}
	return c.binary(e, op)
}

func (c *expressionCompiler) Substring(e *criteria.SubstringExpression) interface{} {
	inJSONContext := isInJSONContext(e.Left())
	join, isJoinedRef := c.expressionRefersToJoinedData(e.Left())
	if inJSONContext || isJoinedRef {
		left, ok := e.Left().(*criteria.FieldExpression)
		if !ok {
			c.err = append(c.err, errs.Errorf("invalid left expression (not a field expression): %+v", e.Left()))
			return nil
		}
		if strings.Contains(left.FieldName, "'") {
			// beware of injection, it's a reasonable restriction for field names,
			// make sure it's not allowed when creating wi types
			c.err = append(c.err, errs.Errorf("single quote not allowed in field name: %s", left.FieldName))
			return nil
		}

		litExp, ok := e.Right().(*criteria.LiteralExpression)
		if !ok {
			c.err = append(c.err, errs.Errorf("failed to convert right expression to literal expression: %+v", e.Right()))
			return nil
		}
		r, ok := litExp.Value.(string)
		if !ok {
			c.err = append(c.err, errs.Errorf("failed to convert value of right literal expression to string: %+v", litExp.Value))
			return nil
		}
		// Handle normal JSON field
		if inJSONContext {
			r = "%" + r + "%"
			c.parameters = append(c.parameters, r)
			return Column(WorkItemStorage{}.TableName(), "fields") + `->>'` + left.FieldName + `' ILIKE ?`
		}
		// Handle more complex joined field
		col, err := join.TranslateFieldName(left.FieldName)
		if err != nil {
			c.err = append(c.err, errs.Wrapf(err, `failed to translate field name: "%s"`, left.FieldName))
			return nil
		}
		return col
	}
	return c.binary(e, "ILIKE")
}

func (c *expressionCompiler) IsNull(e *criteria.IsNullExpression) interface{} {
	mappedFieldName, isJSONField := c.getFieldName(e.FieldName)
	if isJSONField {
		return "(" + Column(WorkItemStorage{}.TableName(), "fields") + "->>'" + mappedFieldName + "' IS NULL)"
	}
	return "(" + mappedFieldName + " IS NULL)"
}

func (c *expressionCompiler) Not(e *criteria.NotExpression) interface{} {
	if isInJSONContext(e.Left()) {
		condition := c.binary(e, ":")
		if condition != nil {
			cond, ok := condition.(string)
			if !ok {
				c.err = append(c.err, errs.Errorf("failed to convert condition to string: %+v", condition))
				return nil
			}
			return "NOT " + cond
		}
		return nil
	}
	return c.binary(e, "!=")
}

func (c *expressionCompiler) Child(e *criteria.ChildExpression) interface{} {
	left, ok := e.Left().(*criteria.FieldExpression)
	if !ok {
		c.err = append(c.err, errs.Errorf("invalid left expression (not a field expression): %+v", e.Left()))
		return nil
	}

	if strings.Contains(left.FieldName, "'") {
		c.err = append(c.err, errs.Errorf("single quote not allowed in field name: %s", left.FieldName))
		return nil
	}
	litExp, ok := e.Right().(*criteria.LiteralExpression)
	if !ok {
		c.err = append(c.err, errs.Errorf("failed to convert right expression to literal expression: %+v", e.Right()))
		return nil
	}
	r, ok := litExp.Value.(string)
	if !ok {
		c.err = append(c.err, errs.Errorf("failed to convert value of right literal expression to string: %+v", litExp.Value))
		return nil
	}
	if strings.Contains(r, "'") {
		// beware of injection, it's a reasonable restriction for field names,
		// make sure it's not allowed when creating wi types
		c.err = append(c.err, errs.Errorf("single quote not allowed in field value: %s", r))
		return nil
	}

	var tblName string
	var tblAlias string
	var tblJoin string
	if left.FieldName == SystemIteration {
		tblName = "iterations"
		tblAlias = "iter"
		tblJoin = "iteration"
	} else if left.FieldName == SystemArea {
		tblName = "areas"
		tblAlias = "ar"
		tblJoin = "area"
	} else {
		c.err = append(c.err, errs.Errorf("invalid field name for child expression: %+v", left.FieldName))
		return nil
	}
	c.joins[tblJoin].Active = true
	c.parameters = append(c.parameters, r)

	// Find all iteration/area which is a child of the given iteration/area
	return fmt.Sprintf(`(uuid("`+WorkItemStorage{}.TableName()+`".fields->>'%[1]s') IN (
				SELECT %[2]s.id
					WHERE
						(SELECT j.path
							FROM %[3]s j
							WHERE j.space_id = "`+WorkItemStorage{}.TableName()+`"."space_id" AND j.id = ? 
						) @> %[2]s.path
							  ))`, left.FieldName, tblAlias, tblName)

}

func (c *expressionCompiler) Parameter(v *criteria.ParameterExpression) interface{} {
	c.err = append(c.err, errs.Errorf("parameter expression not supported"))
	return nil
}

// iterate the parent chain to see if this expression references json fields
func isInJSONContext(exp criteria.Expression) bool {
	result := false
	criteria.IterateParents(exp, func(exp criteria.Expression) bool {
		if exp.Annotation(jsonAnnotation) == true {
			result = true
			return false
		}
		return true
	})
	return result
}

// literal values need to be converted differently depending on whether they are
// used in a JSON context or a regular SQL expression. JSON values are always
// strings (delimited with "'"), but operators can be used depending on the
// dynamic type. For example, you can write "a->'foo' < '5'" and it will return
// true for the json object { "a": 40 }.
func (c *expressionCompiler) Literal(e *criteria.LiteralExpression) interface{} {
	json := isInJSONContext(e)
	if json {
		stringVal, err := c.convertToString(e.Value)
		if err == nil {
			return stringVal + "}'"
		}
		if stringArr, ok := e.Value.([]string); ok {
			return "[" + c.wrapStrings(stringArr) + "]}'"
		}
		c.err = append(c.err, err)
		return nil
	}
	c.parameters = append(c.parameters, e.Value)
	return "?"
}

func (c *expressionCompiler) wrapStrings(value []string) string {
	wrapped := []string{}
	for i := 0; i < len(value); i++ {
		wrapped = append(wrapped, "\""+value[i]+"\"")
	}
	return strings.Join(wrapped, ",")
}

func (c *expressionCompiler) convertToString(value interface{}) (string, error) {
	var result string
	switch t := value.(type) {
	case float64:
		result = strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		result = strconv.Itoa(t)
	case int64:
		result = strconv.FormatInt(t, 10)
	case uint:
		result = strconv.FormatUint(uint64(t), 10)
	case uint64:
		result = strconv.FormatUint(t, 10)
	case string:
		result = "\"" + t + "\""
	case bool:
		result = strconv.FormatBool(t)
	case uuid.UUID:
		result = t.String()
	default:
		return "", errs.Errorf(`unknown value type "%T": %+v`, value, value)
	}
	return result, nil
}
