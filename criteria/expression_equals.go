package criteria

// EqualsExpression represents the equality operator
type EqualsExpression struct {
	binaryExpression
}

// Ensure EqualsExpression implements the Expression interface
var _ Expression = &EqualsExpression{}
var _ Expression = (*EqualsExpression)(nil)

// Accept implements ExpressionVisitor
func (t *EqualsExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Equals(t)
}

// Equals constructs an EqualsExpression
func Equals(left Expression, right Expression) Expression {
	return reparent(&EqualsExpression{binaryExpression{expression{}, left, right}})
}
