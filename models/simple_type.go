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
	case KindString, KindURL, KindUser:
		if valueType.Kind() != reflect.String {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		return value, nil
	case KindInteger, KindFloat, KindDuration:
		// instant == milliseconds
		if valueType.Kind() != reflect.Float64 {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "float64", valueType.Name())
		}
		return value, nil
	case KindInstant:
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

	default:
		return nil, fmt.Errorf("unexpected type constant: %d", fieldType.GetKind())
	}
}

func (fieldType SimpleType) Validate(value interface{}) error {
	switch fieldType.GetKind() {
	case KindURL:
		if govalidator.IsURL(value.(string)) == false {
			return fmt.Errorf("value %v should be a valid %s", value, "URL")
		}
		return nil
	case KindUser:
		return nil
	default:
		return fmt.Errorf("Type %s not supported", fieldType.GetKind())
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
