package criteria

// IsNullExpression represents the IS operator with NULL value
type IsNullExpression struct {
	expression
	FieldName string
}

// Ensure IsNullExpression implements the Expression interface
var _ Expression = &IsNullExpression{}
var _ Expression = (*IsNullExpression)(nil)

// IsNull constructs an NullExpression
func IsNull(name string) Expression {
	return &IsNullExpression{expression{}, name}
}

// Accept implements ExpressionVisitor
func (t *IsNullExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.IsNull(t)
}
