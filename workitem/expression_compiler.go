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

// Compile takes an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and a slice of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, joins map[string]TableJoin, err []error) {
	criteria.IteratePostOrder(where, bubbleUpJSONContext)

	compiler := newExpressionCompiler()
	compiled := where.Accept(&compiler)

	c, ok := compiled.(string)
	if !ok {
		c = ""
	}
	return c, compiler.parameters, compiler.joins, compiler.err
}

// mark expression tree nodes that reference json fields
func bubbleUpJSONContext(exp criteria.Expression) bool {
	switch t := exp.(type) {
	case *criteria.FieldExpression:
		_, isJSONField := getFieldName(t.FieldName)
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
func getFieldName(fieldName string) (mappedFieldName string, isJSONField bool) {
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
		// joins:      map[string]TableJoin{},
	}
}

// A TableJoin helps to construct a query like this:
//
//   SELECT *
//     FROM workitems
//     JOIN iterations iter ON iter.ID = "a1801a16-0f09-4536-8c49-894be664488f"
//     WHERE iter.name = "foo"
//
// With the prefix trigger we can identify if a certain field expression points
// at data from a joined table. By default there are no restrictions on what can
// be queried in joined table but if you fill the allowed/disallowed columns
// arrays you can explicitly disallow columns to be queried.
type TableJoin struct {
	TableName         string // e.g. "iterations"
	TableNameShortcut string // e.g. "iter"
	JoinOnLeftColumn  string // e.g. "iter.ID"
	JoinOnRightColumn string // e.g. "Field->>system.iteration"

	PrefixTrigger     string   // e.g. "iteration."
	AllowedColumns    []string // e.g. ["name"]. when empty all columns are allowed
	DisallowedColumns []string // e.g. ["created_at"]. when empty all columns are allowed
}

// TranslateFieldName returns the name of the linked
func (j TableJoin) TranslateFieldName(fieldName string) string {
	if !strings.HasPrefix(fieldName, j.PrefixTrigger) {
		return ""
	}
	col := strings.TrimPrefix(fieldName, j.PrefixTrigger)
	// if no columns are explicitly allowed, then this column is allowed by
	// default.
	columnIsAllowed := (j.AllowedColumns == nil || len(j.AllowedColumns) == 0)
	for _, allowedColumn := range j.AllowedColumns {
		if allowedColumn == col {
			columnIsAllowed = true
			break
		}
	}
	// if a columns is explictly disallowed we must check for it.
	for _, disallowedColumn := range j.DisallowedColumns {
		if disallowedColumn == col {
			columnIsAllowed = false
			break
		}
	}
	if !columnIsAllowed {
		return ""
	}
	return col
}

// String implements Stringer interface
func (j TableJoin) String() string {
	return "JOIN " + j.TableName + " ON " + j.JoinOnLeftColumn + " = " + j.JoinOnRightColumn
}

// expressionCompiler takes an expression and compiles it to a where clause for our gorm models
// implements criteria.ExpressionVisitor
type expressionCompiler struct {
	parameters []interface{}        // records the number of parameter expressions encountered
	err        []error              // record any errors found in the expression
	joins      map[string]TableJoin // map of table joins keyed by table name
}

// Ensure expressionCompiler implements the ExpressionVisitor interface
var _ criteria.ExpressionVisitor = &expressionCompiler{}
var _ criteria.ExpressionVisitor = (*expressionCompiler)(nil)

// visitor implementation
// the convention is to return nil when the expression cannot be compiled and to append an error to the err field

func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	mappedFieldName, isJSONField := getFieldName(f.FieldName)
	if !isJSONField {
		return mappedFieldName
	}
	if strings.Contains(mappedFieldName, "'") {
		// beware of injection, it's a reasonable restriction for field names,
		// make sure it's not allowed when creating wi types
		c.err = append(c.err, errs.Errorf("single quote not allowed in field name: %s", mappedFieldName))
		return nil
	}

	if strings.HasPrefix(mappedFieldName, "iteration.") {
		if c.joins == nil {
			c.joins = map[string]TableJoin{}
		}
		c.joins["iterations"] = TableJoin{
			TableName:         "iterations",
			TableNameShortcut: "iter",
			JoinOnLeftColumn:  "iter.ID",
			JoinOnRightColumn: fmt.Sprintf("Fields->>'%s'", SystemIteration),
			PrefixTrigger:     "iteration.",
			AllowedColumns:    []string{"name"},
		}
		return "iter." + strings.TrimPrefix(mappedFieldName, "iteration.")
	}

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

func (c *expressionCompiler) lookupJoinedData(fieldExpression criteria.Expression) (TableJoin, bool) {
	switch t := fieldExpression.(type) {
	case *criteria.FieldExpression:
		for _, j := range c.joins {
			if j.TranslateFieldName(t.FieldName) != "" {
				return j, true
			}
		}
	}
	return TableJoin{}, false
}

func (c *expressionCompiler) Equals(e *criteria.EqualsExpression) interface{} {
	op := "="
	if isInJSONContext(e.Left()) {
		op = ":"
	}
	_, ok := c.lookupJoinedData(e.Left())
	if ok {
		op = "="
	}
	return c.binary(e, op)
}

func (c *expressionCompiler) Substring(e *criteria.SubstringExpression) interface{} {
	inJSONContext := isInJSONContext(e.Left())
	_, isJoinedRef := c.lookupJoinedData(e.Left())
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
		if true || inJSONContext {
			r = "%" + r + "%"
			c.parameters = append(c.parameters, r)
			return "Fields->>'" + left.FieldName + "' ILIKE ?"
		}
	}
	return c.binary(e, "ILIKE")
}

func (c *expressionCompiler) IsNull(e *criteria.IsNullExpression) interface{} {
	mappedFieldName, isJSONField := getFieldName(e.FieldName)
	if isJSONField {
		return "(Fields->>'" + mappedFieldName + "' IS NULL)"
	}
	return "(" + mappedFieldName + " IS NULL)"
}

func (c *expressionCompiler) Not(e *criteria.NotExpression) interface{} {
	// TODO(kwk): Handle operator switching here as well if left is a field
	// expression and has an "iteration." prefix.
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

// literal values need to be converted differently depending on whether they are used in a JSON context or a regular SQL expression.
// JSON values are always strings (delimited with "'"), but operators can be used depending on the dynamic type. For example,
// you can write "a->'foo' < '5'" and it will return true for the json object { "a": 40 }.
func (c *expressionCompiler) Literal(v *criteria.LiteralExpression) interface{} {
	json := isInJSONContext(v)
	if json {
		stringVal, err := c.convertToString(v.Value)
		if err == nil {
			return stringVal + "}'"
		}
		if stringArr, ok := v.Value.([]string); ok {
			return "[" + c.wrapStrings(stringArr) + "]}'"
		}
		c.err = append(c.err, err)
		return nil
	}
	c.parameters = append(c.parameters, v.Value)
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
		return "", errs.Errorf("unknown value type of %v: %T", value, value)
	}
	return result, nil
}
