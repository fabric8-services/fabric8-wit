package criteria

// SubstringExpression represents the substring operator
type SubstringExpression struct {
	binaryExpression
}

// Ensure SubstringExpression implements the Expression interface
var _ Expression = &SubstringExpression{}
var _ Expression = (*SubstringExpression)(nil)

// Accept implements ExpressionVisitor
func (t *SubstringExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Substring(t)
}

// Substring constructs an SubstringExpression
func Substring(left Expression, right Expression) Expression {
	return reparent(&SubstringExpression{binaryExpression{expression{}, left, right}})
}
