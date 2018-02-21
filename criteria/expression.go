package criteria

// Expression is used to express conditions for selecting an entity
type Expression interface {
	// Accept calls the visitor callback of the appropriate type
	Accept(visitor ExpressionVisitor) interface{}
	// SetAnnotation puts the given annotation on the expression
	SetAnnotation(key string, value interface{})
	// Annotation reads back values set with SetAnnotation
	Annotation(key string) interface{}
	// Returns the parent expression or nil
	Parent() Expression
	setParent(parent Expression)
}

// NOTE: expression itself doesn't implement the Expression interface because
// the Accept method is "missing".
type expression struct {
	parent      Expression
	annotations map[string]interface{}
}

func (exp *expression) SetAnnotation(key string, value interface{}) {
	if exp.annotations == nil {
		exp.annotations = map[string]interface{}{}
	}
	exp.annotations[key] = value
}

func (exp *expression) Annotation(key string) interface{} {
	return exp.annotations[key]
}

func (exp *expression) Parent() Expression {
	result := exp.parent
	return result
}

func (exp *expression) setParent(parent Expression) {
	exp.parent = parent
}
