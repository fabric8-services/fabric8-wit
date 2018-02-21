package criteria

// NotExpression represents the negation operator
type NotExpression struct {
	binaryExpression
}

// Ensure NotExpression implements the Expression interface
var _ Expression = &NotExpression{}
var _ Expression = (*NotExpression)(nil)

// Accept implements ExpressionVisitor
func (t *NotExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Not(t)
}

// Not constructs a NotExpression
func Not(left Expression, right Expression) Expression {
	return reparent(&NotExpression{binaryExpression{expression{}, left, right}})
}
