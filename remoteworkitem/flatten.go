package remoteworkitem

import (
	"reflect"
	"strings"
)

type stack []string

func (s stack) push(v string) stack {
	return append(s, v)
}

func (s stack) pop() (stack, string) {
	l := len(s)
	if l == 0 {
		return s, ""
	}
	return s[:l-1], s[l-1]
}

func (s stack) String() string {
	return strings.Join(s, ".")
}

// nestedKeyStack stores the stack of keys till a leaf node is found.
var nestedKeyStack = make(stack, 0)

// flattendedMap is where the flattened map with dot delimited keys are stored.
var flattendMap map[string]interface{} = make(map[string]interface{})

// traverseMapAsDFS does a depth-first traversal
func traverseMapAsDFS(value map[string]interface{}) {
	for k, v := range value {
		nestedKeyStack = nestedKeyStack.push(k)

		if v != nil && reflect.TypeOf(v).Kind().String() == "map" {
			traverseMapAsDFS(v.(map[string]interface{}))
		} else {
			flattendMap[nestedKeyStack.String()] = v
			nestedKeyStack, _ = nestedKeyStack.pop()
		}

	}
}

// Flatten Takes the nested map and returns a non nested one with dot delimited keys
func Flatten(nestedMap map[string]interface{}) map[string]interface{} {
	nestedKeyStack = make(stack, 0)
	flattendMap = make(map[string]interface{})
	traverseMapAsDFS(nestedMap)
	return flattendMap
}
