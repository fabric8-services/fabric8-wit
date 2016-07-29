// Package query This package implements a super basic parser for search expressions. To be replaced by something more complete when we get to that bridge
package query

import (
	"encoding/json"

	. "github.com/almighty/almighty-core/models/criteria"
)

// Parse parses strings of the form "attribute1:value1,attribute2:value2" into an expression of the form "true and attribute1=value1 and attribute2=value2"
func Parse(exp *string) (Expression, error) {
	if exp == nil || len(*exp) == 0 {
		return Constant(true), nil
	}
	var unmarshalled map[string]interface{}
	err := json.Unmarshal([]byte(*exp), &unmarshalled)
	if err != nil {
		return nil, err
	}
	var result *Expression
	if len(unmarshalled) > 0 {
		for key, value := range unmarshalled {
			current := Equals(Field(key), Value(value))
			if result == nil {
				result = &current
			} else {
				current = And(*result, current)
				result = &current
			}
		}
		return *result, nil
	}
	return Constant(true), nil
}
