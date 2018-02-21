package criteria

// BinaryExpression represents expressions with 2 children
// This could be generalized to n-ary expressions, but that is not necessary right now
type BinaryExpression interface {
	Expression
	Left() Expression
	Right() Expression
}

// binaryExpression is an "abstract" type for binary expressions.
// NOTE: binaryExpression itself doesn't implement the Expression interface
// because the Accept method is "missing".
type binaryExpression struct {
	expression
	left  Expression
	right Expression
}

// Left implements BinaryExpression
func (exp *binaryExpression) Left() Expression {
	return exp.left
}

// Right implements BinaryExpression
func (exp *binaryExpression) Right() Expression {
	return exp.right
}

// make sure the children have the correct parent
func reparent(parent BinaryExpression) Expression {
	parent.Left().setParent(parent)
	parent.Right().setParent(parent)
	return parent
}
