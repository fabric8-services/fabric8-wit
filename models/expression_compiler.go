package models

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/criteria"
)

const (
	jsonAnnotation = "JSON"
)

// Compile takes a an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and an array of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameterCount uint16, err []error) {
	criteria.IteratePostOrder(where, bubbleUpJSONContext)

	compiler := expressionCompiler{}
	compiled := where.Accept(&compiler)

	return compiled.(string), compiler.parameterCount, compiler.err
}

func bubbleUpJSONContext(exp criteria.Expression) bool {
	switch t := exp.(type) {
	case *criteria.FieldExpression:
		if isJSONField(t.FieldName) {
			t.SetAnnotation(jsonAnnotation, true)
		}
	case *criteria.EqualsExpression:
		if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
			t.SetAnnotation(jsonAnnotation, true)
		}
	}
	return true
}

func isJSONField(fieldName string) bool {
	switch fieldName {
	case "ID", "Name", "Type", "Version":
		return false
	}
	return true
}

type expressionCompiler struct {
	parameterCount uint16
	err            []error
}

// Field implements criteria.ExpressionVisitor
func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	if !isJSONField(f.FieldName) {
		return f.FieldName
	}
	return "Fields->'" + f.FieldName + "'"
}

// And implements criteria.ExpressionVisitor
func (c *expressionCompiler) And(a *criteria.AndExpression) interface{} {
	return c.binary(a, "and")
}

func (c *expressionCompiler) binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left().Accept(c)
	right := a.Right().Accept(c)
	if left != nil && right != nil {
		return "(" + left.(string) + " " + op + " " + right.(string) + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return ""
}

// Or implements criteria.ExpressionVisitor
func (c *expressionCompiler) Or(a *criteria.OrExpression) interface{} {
	return c.binary(a, "or")
}

// Equals implements criteria.ExpressionVisitor
func (c *expressionCompiler) Equals(e *criteria.EqualsExpression) interface{} {
	return c.binary(e, "=")
}

// Field implements criteria.ExpressionVisitor
func (c *expressionCompiler) Parameter(v *criteria.ParameterExpression) interface{} {
	c.parameterCount++
	return "?"
}

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

func (c *expressionCompiler) Literal(v *criteria.LiteralExpression) interface{} {
	json := isInJSONContext(v)
	switch t := v.Value.(type) {
	case float64:
		return wrapJson(json, strconv.FormatFloat(t, 'f', -1, 64))
	case int:
		return wrapJson(json, strconv.FormatInt(int64(t), 10))
	case int64:
		return wrapJson(json, strconv.FormatInt(t, 10))
	case uint:
		return wrapJson(json, strconv.FormatUint(uint64(t), 10))
	case uint64:
		return wrapJson(json, strconv.FormatUint(t, 10))
	case string:
		if json {
			return "'\"" + t + "\"'"
		}
		return "'" + t + "'"
	case bool:
		return wrapJson(json, strconv.FormatBool(t))
	}
	c.err = append(c.err, fmt.Errorf("unknown value type of %v: %T", v.Value, v.Value))
	return ""
}

func wrapJson(isJSON bool, value string) string {
	if isJSON {
		return "'" + value + "'"
	}
	return value
}
