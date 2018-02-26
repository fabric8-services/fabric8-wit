package workitem

import (
	"strconv"
	"strings"

	"github.com/fabric8-services/fabric8-wit/criteria"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	jsonAnnotation = "JSON"
)

// Compile takes an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and a slice of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, joins map[string]TableJoin, err []error) {
	compiler := newExpressionCompiler()

	criteria.IteratePostOrder(where, bubbleUpJSONContext(&compiler))

	compiled := where.Accept(&compiler)

	c, ok := compiled.(string)
	if !ok {
		c = ""
	}

	// Make sure we don't return all possible joins but only the once that were activated
	joins = map[string]TableJoin{}
	for k, j := range compiler.joins {
		if j.IsActive() {
			joins[k] = *j
		}
	}
	if len(joins) <= 0 {
		joins = nil
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
		return mappedFieldName, false
	}

	if strings.Contains(fieldName, ".") {
		// leave field untouched
		return fieldName, true
	}
	return fieldName, false
}

func newExpressionCompiler() expressionCompiler {
	return expressionCompiler{
		parameters: []interface{}{},
		// Define all possible join scenarios here
		joins: map[string]*TableJoin{
			"iteration": {
				TableName:      "iterations",
				TableAlias:     "iter",
				On:             JoinOnJSONField(SystemIteration, "iter.id"),
				PrefixTriggers: []string{"iteration."},
				AllowedColumns: []string{"name", "created_at"},
			},
			"area": {
				TableName:      "areas",
				TableAlias:     "ar",
				On:             JoinOnJSONField(SystemArea, "ar.id"),
				PrefixTriggers: []string{"area."},
				AllowedColumns: []string{"name"},
			},
			"codebase": {
				TableName:      "codebases",
				TableAlias:     "cb",
				On:             JoinOnJSONField(SystemCodebase, "cb.id"),
				PrefixTriggers: []string{"codebase."},
				AllowedColumns: []string{"url"},
			},
			"work_item_type": {
				TableName:      "work_item_types",
				TableAlias:     "wit",
				On:             "wit.id = " + WorkItemStorage{}.TableName() + ".type",
				PrefixTriggers: []string{"wit.", "workitemtype.", "work_item_type.", "type."},
				AllowedColumns: []string{"name"},
			},
			"space": {
				TableName:      "spaces",
				TableAlias:     "space",
				On:             "space.id = " + WorkItemStorage{}.TableName() + ".space_id",
				PrefixTriggers: []string{"space."},
				AllowedColumns: []string{"name"},
			},
			"creator": {
				TableName:      "users",
				TableAlias:     "creator",
				On:             JoinOnJSONField(SystemCreator, "creator.id"),
				PrefixTriggers: []string{"creator.", "author."},
				AllowedColumns: []string{"full_name"},
			},
		},
	}
}

// expressionCompiler takes an expression and compiles it to a where clause for our gorm models
// implements criteria.ExpressionVisitor
type expressionCompiler struct {
	parameters []interface{}         // records the number of parameter expressions encountered
	err        []error               // record any errors found in the expression
	joins      map[string]*TableJoin // map of table joins keyed by table name
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
				j.Activate()
				return j, true
			}
		}
	}
	return nil, false
}

// visitor implementation
// the convention is to return nil when the expression cannot be compiled and to append an error to the err field

func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	mappedFieldName, isJSONField := c.getFieldName(f.FieldName)

	// Check if this field is referencing joinable data
	for _, j := range c.joins {
		if j.HandlesFieldName(mappedFieldName) {
			j.Activate()
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
	return "Fields@>'{\"" + mappedFieldName + "\""
}

func (c *expressionCompiler) And(a *criteria.AndExpression) interface{} {
	return c.binary(a, "and")
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
	return c.binary(a, "or")
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
			return "Fields->>'" + left.FieldName + "' ILIKE ?"
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
		return "(Fields->>'" + mappedFieldName + "' IS NULL)"
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
