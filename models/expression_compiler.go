package models

import (
	"fmt"
	"strconv"

	"github.com/almighty/almighty-core/models/criteria"
)

func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, err []error) {
	compiler := ExpressionCompiler{}
	compiled := where.Accept(&compiler)

	return compiled.(string), compiler.parameterValues, compiler.err
}

type ExpressionCompiler struct {
	parameterValues []interface{}
	err             []error
}

func (c *ExpressionCompiler) Field(f criteria.FieldExpression) interface{} {
	switch f.FieldName {
	case "ID", "Name", "Type", "Version":
		return f.FieldName
	default:
		return "Fields->'" + f.FieldName + "'"
	}
}
func (c *ExpressionCompiler) And(a criteria.AndExpression) interface{} {
	return c.Binary(a.BinaryExpression, "and")
}

func (c *ExpressionCompiler) Binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left.Accept(c)
	right := a.Right.Accept(c)
	if left != nil && right != nil {
		return "(" + left.(string) + " " + op + " " + right.(string) + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return nil
}

func (c *ExpressionCompiler) Or(a criteria.OrExpression) interface{} {
	return c.Binary(a.BinaryExpression, "or")
}
func (c *ExpressionCompiler) Equals(e criteria.EqualsExpression) interface{} {
	return c.Binary(e.BinaryExpression, "=")
}

func (c *ExpressionCompiler) Value(v criteria.ValueExpression) interface{} {
	c.parameterValues = append(c.parameterValues, v.Value)
	return "?"
}

func (c *ExpressionCompiler) Constant(v criteria.ConstantExpression) interface{} {
	switch t := v.Value.(type) {
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case string:
		return "'" + t + "'"
	case bool:
		return strconv.FormatBool(t)
	}
	c.err = append(c.err, fmt.Errorf("unknonw value type of %v: %T", v.Value, v.Value))
	return nil
}
