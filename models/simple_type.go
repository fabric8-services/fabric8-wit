package models

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
)

// SimpleType is an unstructured FieldType
type SimpleType struct {
	Kind Kind
}

// GetKind implements FieldType
func (self SimpleType) GetKind() Kind {
	return self.Kind
}

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

// ConvertToModel implements the FieldType interface
func (fieldType SimpleType) ConvertToModel(value interface{}) (interface{}, error) {
	valueType := reflect.TypeOf(value)
	switch fieldType.GetKind() {
	case KindString, KindUser:
		if valueType.Kind() != reflect.String {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		return value, nil
	case KindURL:
		if valueType.Kind() == reflect.String && govalidator.IsURL(value.(string)) {
			return value, nil
		}
		return nil, fmt.Errorf("value %v should be %s, but is %s", value, "URL", valueType.Name())
	case KindFloat:
		if valueType.Kind() != reflect.Float64 {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "float64", valueType.Name())
		}
		return value, nil
	case KindInteger, KindDuration:
		if valueType.Kind() != reflect.Int {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "int", valueType.Name())
		}
		return value, nil
	case KindInstant:
		// instant == milliseconds
		if !valueType.Implements(timeType) {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "time.Time", valueType.Name())
		}
		return value.(time.Time).UnixNano(), nil
	case KindWorkitemReference:
		if valueType.Kind() != reflect.String {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		idValue, err := strconv.Atoi(value.(string))
		return idValue, err
	case KindList:
		if (valueType.Kind() != reflect.Array) && (valueType.Kind() != reflect.Slice) {
			return nil, fmt.Errorf("value %v should be %s, but is %s,", value, "array/slice", valueType.Kind())
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected type constant: %d", fieldType.GetKind())
	}
}

// ConvertFromModel implements the FieldType interface
func (fieldType SimpleType) ConvertFromModel(value interface{}) (interface{}, error) {
	valueType := reflect.TypeOf(value)
	switch fieldType.GetKind() {
	case KindString, KindURL, KindUser, KindInteger, KindFloat, KindDuration:
		return value, nil
	case KindInstant:
		return time.Unix(0, value.(int64)), nil
	case KindWorkitemReference:
		if valueType.Kind() != reflect.String {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		return strconv.FormatUint(value.(uint64), 10), nil
	default:
		return nil, fmt.Errorf("unexpected type constant: %d", fieldType.GetKind())
	}
}
