package criteria

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// test left-to-right, depth first iteration
	visited := []Expression{}
	l := Field("a")
	r := Literal(5)
	expr := Equals(l, r)
	IteratePostOrder(expr, func(expr Expression) bool {
		visited = append(visited, expr)
		return true
	})
	expected := []Expression{l, r, expr}
	require.Equal(t, expected, visited, "visited should be %+v, but is %+v", expected, visited)

	// test early iteration cutoff with false return from iterator function
	visited = []Expression{}
	IteratePostOrder(expr, func(expr Expression) bool {
		visited = append(visited, expr)
		return expr != r
	})
	expected = []Expression{l, r}
	require.Equal(t, expected, visited, "visited should be %+v, but is %+v", expected, visited)

}
