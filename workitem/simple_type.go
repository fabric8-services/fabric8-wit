package workitem

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/almighty/almighty-core/convert"
	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
)

// SimpleType is an unstructured FieldType
type SimpleType struct {
	Kind Kind
}

// Ensure SimpleType implements the Equaler interface
var _ convert.Equaler = SimpleType{}
var _ convert.Equaler = (*SimpleType)(nil)

// Equal returns true if two SimpleType objects are equal; otherwise false is returned.
func (self SimpleType) Equal(u convert.Equaler) bool {
	other, ok := u.(SimpleType)
	if !ok {
		return false
	}
	return self.Kind == other.Kind
}

// GetKind implements FieldType
func (self SimpleType) GetKind() Kind {
	return self.Kind
}

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

// ConvertToModel implements the FieldType interface
func (fieldType SimpleType) ConvertToModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch fieldType.GetKind() {
	case KindString, KindUser, KindIteration:
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
	case KindEnum:
		// to be done yet | not sure what to write here as of now.
		return value, nil
	case KindMarkup:
		// 'markup' is just a string in the API layer for now:
		// it corresponds to the MarkupContent.Content field. The MarkupContent.Markup is set to the default value
		switch value.(type) {
		case MarkupContent:
			markupContent := value.(MarkupContent)
			return markupContent.toMap(), nil
		default:
			return nil, errors.Errorf("value %v should be %s, but is %s", value, "MarkupContent", valueType)
		}
	default:
		return nil, errors.Errorf("unexpected type constant: '%s'", fieldType.GetKind())
	}
}

// ConvertFromModel implements the FieldType interface
func (fieldType SimpleType) ConvertFromModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch fieldType.GetKind() {
	case KindString, KindURL, KindUser, KindInteger, KindFloat, KindDuration, KindIteration:
		return value, nil
	case KindInstant:
		return time.Unix(0, value.(int64)), nil
	case KindWorkitemReference:
		if valueType.Kind() != reflect.String {
			return nil, fmt.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		return strconv.FormatUint(value.(uint64), 10), nil
	case KindMarkup:
		if valueType.Kind() != reflect.Map {
			return nil, errors.Errorf("value %v should be %s, but is %s", value, reflect.Map, valueType.Name())
		}
		return NewMarkupContentFromMap(value.(map[string]interface{})), nil
	default:
		return nil, errors.Errorf("unexpected field type: %s", fieldType.GetKind())
	}
}
