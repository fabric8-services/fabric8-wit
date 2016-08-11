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

// should validate SimpleType completely
// returns nil if value is as expected else returns error.
func (fieldType SimpleType) Validate(value interface{}) error {
	// && govalidator.IsURL(value.(string)) == false
	valueType := reflect.TypeOf(value)
	switch fieldType.GetKind() {
	case KindString:
		if valueType.Kind() != reflect.String {
			return fmt.Errorf("value %v should be a valid %s", value, KindString)
		}
		return nil
	case KindInteger, KindFloat, KindDuration:
		// have to group these together becasue valueType.Kind() gives `float` for above group
		if valueType.Kind() != reflect.Float64 {
			return fmt.Errorf("value %v should be a valid %s", value, KindInteger)
		}
		return nil
	case KindURL:
		if valueType.Kind() == reflect.String && govalidator.IsURL(value.(string)) == true {
			return nil
		} else {
			return fmt.Errorf("value %v should be a valid %s", value, KindURL)
		}
	case KindWorkitemReference:
		// need to add validation for this
		return nil
	case KindUser:
		// need to add validation for this
		return nil
	case KindEnum:
		// need to add validation for this
		return nil
	case KindList:
		if valueType.Kind() != reflect.Array {
			return fmt.Errorf("value %v should be a valid %s", value, KindList)
		}
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
