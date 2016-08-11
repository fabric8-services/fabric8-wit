package models

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/criteria"
)

// Compile takes a an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and an array of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameterCount uint16, err []error) {
	compiler := expressionCompiler{}
	compiled := where.Accept(&compiler)

	return compiled.(string), compiler.parameterCount, compiler.err
}

type expressionCompiler struct {
	parameterCount uint16
	err            []error
}

// Field implements criteria.ExpressionVisitor
func (c *expressionCompiler) Field(f criteria.FieldExpression) interface{} {
	switch f.FieldName {
	case "ID", "Name", "Type", "Version":
		return f.FieldName
	default:
		return "Fields->'" + f.FieldName + "'"
	}
}

// And implements criteria.ExpressionVisitor
func (c *expressionCompiler) And(a criteria.AndExpression) interface{} {
	return c.binary(a.BinaryExpression, "and")
}

func (c *expressionCompiler) binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left.Accept(c)
	right := a.Right.Accept(c)
	if left != nil && right != nil {
		return "(" + left.(string) + " " + op + " " + right.(string) + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return ""
}

// Or implements criteria.ExpressionVisitor
func (c *expressionCompiler) Or(a criteria.OrExpression) interface{} {
	return c.binary(a.BinaryExpression, "or")
}

// Equals implements criteria.ExpressionVisitor
func (c *expressionCompiler) Equals(e criteria.EqualsExpression) interface{} {
	return c.binary(e.BinaryExpression, "=")
}

// Field implements criteria.ExpressionVisitor
func (c *expressionCompiler) Parameter(v criteria.ParameterExpression) interface{} {
	c.parameterCount++
	return "?"
}

func (c *expressionCompiler) Value(v criteria.LiteralExpression) interface{} {
	switch t := v.Value.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case string:
		return "'\"" + t + "\"'"
	case bool:
		return strconv.FormatBool(t)
	}
	c.err = append(c.err, fmt.Errorf("unknown value type of %v: %T", v.Value, v.Value))
	return ""
}
