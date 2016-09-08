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

// MapFlattener defines the data structures and the implemetation of how to flatten a nested map.
type MapFlattener struct {
	nestedKeyStack stack
	flattenedMap   map[string]interface{}
}

// traverseMapAsDFS does a depth-first traversal
func (fm MapFlattener) traverseMapAsDFS(value map[string]interface{}) {

	for k, v := range value {
		fm.nestedKeyStack.push(k)

		if v != nil && reflect.TypeOf(v).Kind() == reflect.Map {
			fm.traverseMapAsDFS(v.(map[string]interface{}))
		} else {
			fm.flattenedMap[fm.nestedKeyStack.String()] = v
		}
		_ = fm.nestedKeyStack.pop()
	}
}

// Flatten Takes the nested map and returns a non nested one with dot delimited keys
func (fm MapFlattener) Flatten(nestedMap map[string]interface{}) map[string]interface{} {
	fm.nestedKeyStack = make(stack, 0)
	fm.flattenedMap = make(map[string]interface{})
	fm.traverseMapAsDFS(nestedMap)
	return fm.flattenedMap
}
