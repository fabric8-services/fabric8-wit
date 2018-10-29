package criteria

// ChildExpression represents the child operator
type ChildExpression struct {
	binaryExpression
}

// Ensure ChildExpression implements the Expression interface
var _ Expression = &ChildExpression{}
var _ Expression = (*ChildExpression)(nil)

// Accept implements ExpressionVisitor
func (t *ChildExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Child(t)
}

// Child constructs a ChildExpression
func Child(left Expression, right Expression) Expression {
	return reparent(&ChildExpression{binaryExpression{expression{}, left, right}})
}
