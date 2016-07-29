package criteria

type Expression interface {
	Accept(visitor ExpressionVisitor) interface{}
}

type ExpressionVisitor interface {
	Field(t FieldExpression) interface{}
	And(a AndExpression) interface{}
	Or(a OrExpression) interface{}
	Equals(e EqualsExpression) interface{}
	Value(v ValueExpression) interface{}
	Constant(c ConstantExpression) interface{}
}

// access Field

type FieldExpression struct {
	FieldName string
}

func (t FieldExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Field(t)
}

func Field(id string) Expression {
	return FieldExpression{id}
}

// simple value

type ValueExpression struct {
	Value interface{}
}

func (t ValueExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Value(t)
}

func Value(value interface{}) Expression {
	return ValueExpression{value}
}

// constant

type ConstantExpression struct {
	Value interface{}
}

func (t ConstantExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Constant(t)
}

func Constant(value interface{}) Expression {
	return ConstantExpression{value}
}

type BinaryExpression struct {
	Left  Expression
	Right Expression
}

// And

type AndExpression struct {
	BinaryExpression
}

func (t AndExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.And(t)
}

func And(left Expression, right Expression) Expression {
	return AndExpression{BinaryExpression{left, right}}
}

// Or

type OrExpression struct {
	BinaryExpression
}

func (t OrExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Or(t)
}

func Or(left Expression, right Expression) Expression {
	return OrExpression{BinaryExpression{left, right}}
}

// ==

type EqualsExpression struct {
	BinaryExpression
}

func (t EqualsExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Equals(t)
}

func Equals(left Expression, right Expression) Expression {
	return EqualsExpression{BinaryExpression{left, right}}
}
