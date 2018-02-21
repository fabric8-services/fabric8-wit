package criteria

// IterateParents calls f for every member of the parent chain
// Stops iterating if f returns false
func IterateParents(exp Expression, f func(Expression) bool) {
	if exp != nil {
		exp = exp.Parent()
	}
	for exp != nil {
		if !f(exp) {
			return
		}
		exp = exp.Parent()
	}
}
