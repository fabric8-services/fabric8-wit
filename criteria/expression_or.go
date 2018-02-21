package criteria

// OrExpression represents the disjunction operation of two terms
type OrExpression struct {
	binaryExpression
}

// Ensure OrExpression implements the Expression interface
var _ Expression = &OrExpression{}
var _ Expression = (*OrExpression)(nil)

// Accept implements ExpressionVisitor
func (t *OrExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Or(t)
}

// Or constructs an OrExpression
func Or(left Expression, right Expression) Expression {
	return reparent(&OrExpression{binaryExpression{expression{}, left, right}})
}
