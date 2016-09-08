package remoteworkitem

import (
	"reflect"
	"strings"
)

type stack []string

func (s *stack) push(v string) {
	*s = append(*s, v)
}

func (s *stack) pop() string {
	l := len(*s)
	if l == 0 {
		return ""
	}
	stackReference := *s // because *s[:l-1] did not work.
	poppedData := stackReference[l-1]
	*s = stackReference[:l-1]
	return poppedData
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
		nestedKeyStack.push(k)

		if v != nil && reflect.TypeOf(v).Kind() == reflect.Map {
			traverseMapAsDFS(v.(map[string]interface{}))
		} else {
			flattendMap[nestedKeyStack.String()] = v
			_ = nestedKeyStack.pop()
		}

	}
	_ = nestedKeyStack.pop()
}

// Flatten Takes the nested map and returns a non nested one with dot delimited keys
func Flatten(nestedMap map[string]interface{}) map[string]interface{} {
	nestedKeyStack = make(stack, 0)
	flattendMap = make(map[string]interface{})
	traverseMapAsDFS(nestedMap)
	return flattendMap
}
