package remoteworkitem

import (
	"fmt"
	"log"
	"math"
	"reflect"
)

// Flatten Takes the nested map and returns a non nested one with dot delimited keys
func Flatten(source map[string]interface{}) map[string]interface{} {
	target := make(map[string]interface{})
	flatten(target, source, nil)
	log.Println("Target: ")
	for k, v := range target {
		switch v.(type) {
		case string:
			value := v.(string)
			l := int(math.Min(float64(60), float64(len(value))))
			log.Printf("\t%s=%v\n", k, value[0:l])
		}
	}
	return target
}

func flatten(target map[string]interface{}, source map[string]interface{}, parent *string) {
	for k, v := range source {
		var key string
		if parent == nil {
			key = k
		} else {
			key = *parent + "." + k
		}

		if v != nil && reflect.TypeOf(v).Kind() == reflect.Map {
			flatten(target, v.(map[string]interface{}), &key)
		} else if v != nil && reflect.TypeOf(v).Kind() == reflect.Slice {
			arrayAsMap := convertArrayToMap(v.([]interface{}))
			flatten(target, arrayAsMap, &key)
		} else {
			target[key] = v
		}
	}
}

func convertArrayToMap(arrayOfObjects []interface{}) map[string]interface{} {
	arrayAsMap := make(map[string]interface{})
	for k, v := range arrayOfObjects {
		arrayAsMap[fmt.Sprintf("%d", k)] = v
	}
	return arrayAsMap
}
