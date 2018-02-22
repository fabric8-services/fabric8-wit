package criteria

// AndExpression represents the conjunction operation of two terms
type AndExpression struct {
	binaryExpression
}

// Ensure AndExpression implements the Expression interface
var _ Expression = &AndExpression{}
var _ Expression = (*AndExpression)(nil)

// Accept implements ExpressionVisitor
func (t *AndExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.And(t)
}

// And constructs an AndExpression
func And(left Expression, right Expression) Expression {
	return reparent(&AndExpression{binaryExpression{expression{}, left, right}})
}
