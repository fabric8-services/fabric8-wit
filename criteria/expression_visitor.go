package criteria

// ExpressionVisitor is an implementation of the visitor pattern for expressions
type ExpressionVisitor interface {
	Field(t *FieldExpression) interface{}
	And(a *AndExpression) interface{}
	Or(a *OrExpression) interface{}
	Equals(e *EqualsExpression) interface{}
	Substring(e *SubstringExpression) interface{}
	Parameter(v *ParameterExpression) interface{}
	Literal(c *LiteralExpression) interface{}
	Not(e *NotExpression) interface{}
	Child(e *ChildExpression) interface{}
	IsNull(e *IsNullExpression) interface{}
}
