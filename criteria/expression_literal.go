package criteria

// A LiteralExpression represents a single constant value in the expression, think "5" or "asdf"
// the type of literals is not restricted at this level, but compilers or interpreters will have limitations on what they handle
type LiteralExpression struct {
	expression
	Value interface{}
}

// Ensure LiteralExpression implements the Expression interface
var _ Expression = &LiteralExpression{}
var _ Expression = (*LiteralExpression)(nil)

// Accept implements ExpressionVisitor
func (t *LiteralExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Literal(t)
}

// Literal constructs a literal expression
func Literal(value interface{}) Expression {
	return &LiteralExpression{expression{}, value}
}
