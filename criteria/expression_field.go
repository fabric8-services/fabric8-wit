package criteria

// FieldExpression represents access to a field of the tested object
type FieldExpression struct {
	expression
	FieldName string
}

// Ensure FieldExpression implements the Expression interface
var _ Expression = &FieldExpression{}
var _ Expression = (*FieldExpression)(nil)

// Accept implements ExpressionVisitor
func (t *FieldExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Field(t)
}

// Field constructs a FieldExpression
func Field(id string) Expression {
	return &FieldExpression{expression{}, id}
}
