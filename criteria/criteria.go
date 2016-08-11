package criteria

// Expression is used to express conditions for selecting an entity
type Expression interface {
	Accept(visitor ExpressionVisitor) interface{}
}

// ExpressionVisitor is an implementation of the visitor pattern for expressions
type ExpressionVisitor interface {
	Field(t FieldExpression) interface{}
	And(a AndExpression) interface{}
	Or(a OrExpression) interface{}
	Equals(e EqualsExpression) interface{}
	Parameter(v ParameterExpression) interface{}
	Value(c LiteralExpression) interface{}
}

// access Field

// FieldExpression represents access to a field of the tested object
type FieldExpression struct {
	FieldName string
}

// Accept implements ExpressionVisitor
func (t FieldExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Field(t)
}

// Field constructs a FieldExpression
func Field(id string) Expression {
	return FieldExpression{id}
}

// Parameter (free variable of the expression)

// A ParameterExpression represents a parameter to be passed upon evaluation of the expression
type ParameterExpression struct {
}

// Accept implements ExpressionVisitor
func (t ParameterExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Parameter(t)
}

// Parameter constructs a value expression.
func Parameter() Expression {
	return ParameterExpression{}
}

// constant value

// A LiteralExpression represents a single constant value in the expression, think "5" or "asdf"
type LiteralExpression struct {
	Value interface{}
}

// Accept implements ExpressionVisitor
func (t LiteralExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Value(t)
}

// Literal constructs a literal expression
func Literal(value interface{}) Expression {
	return LiteralExpression{value}
}

// BinaryExpression is an "abstract" type for binary expressions.
type BinaryExpression struct {
	Left  Expression
	Right Expression
}

// And

// AndExpression represents the conjunction operation of two terms
type AndExpression struct {
	BinaryExpression
}

// Accept implements ExpressionVisitor
func (t AndExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.And(t)
}

// And constructs an AndExpression
func And(left Expression, right Expression) Expression {
	return AndExpression{BinaryExpression{left, right}}
}

// Or

// OrExpression represents the disjunction operation of two terms
type OrExpression struct {
	BinaryExpression
}

// Accept implements ExpressionVisitor
func (t OrExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Or(t)
}

// Or constructs an OrExpression
func Or(left Expression, right Expression) Expression {
	return OrExpression{BinaryExpression{left, right}}
}

// ==

// EqualsExpression represents the equality operator
type EqualsExpression struct {
	BinaryExpression
}

// Accept implements ExpressionVisitor
func (t EqualsExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Equals(t)
}

// Equals constructs an EqualsExpression
func Equals(left Expression, right Expression) Expression {
	return EqualsExpression{BinaryExpression{left, right}}
}
