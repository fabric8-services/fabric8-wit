package criteria

// A ParameterExpression represents a parameter to be passed upon evaluation of the expression
type ParameterExpression struct {
	expression
}

// Ensure ParameterExpression implements the Expression interface
var _ Expression = &ParameterExpression{}
var _ Expression = (*ParameterExpression)(nil)

// Accept implements ExpressionVisitor
func (t *ParameterExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Parameter(t)
}

// Parameter constructs a value expression.
// Parameter (free variable of the expression)
func Parameter() Expression {
	return &ParameterExpression{}
}
