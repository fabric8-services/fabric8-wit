package criteria

func IteratePostOrder(exp Expression, visitorFunction func(exp Expression) bool) {
	exp.Accept(&postOrderIterator{visitorFunction})
}

type postOrderIterator struct {
	visit func(exp Expression) bool
}

func (i *postOrderIterator) Field(exp *FieldExpression) interface{} {
	return i.visit(exp)
}

func (i *postOrderIterator) And(exp *AndExpression) interface{} {
	return i.binary(exp)
}

func (i *postOrderIterator) Or(exp *OrExpression) interface{} {
	return i.binary(exp)
}

func (i *postOrderIterator) Equals(exp *EqualsExpression) interface{} {
	return i.binary(exp)
}

func (i *postOrderIterator) Parameter(exp *ParameterExpression) interface{} {
	return i.visit(exp)
}

func (i *postOrderIterator) Literal(exp *LiteralExpression) interface{} {
	return i.visit(exp)
}

func (i *postOrderIterator) binary(exp BinaryExpression) bool {
	if exp.Left().Accept(i) == false {
		return false
	}
	if exp.Right().Accept(i) == false {
		return false
	}
	return i.visit(exp)
}
