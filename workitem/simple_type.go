package workitem

import (
	"math"
	"reflect"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/rendering"
	errs "github.com/pkg/errors"
)

// SimpleType is an unstructured FieldType
type SimpleType struct {
	Kind Kind `json:"kind"`
}

// Ensure SimpleType implements the FieldType interface
var _ FieldType = SimpleType{}
var _ FieldType = (*SimpleType)(nil)

// Ensure SimpleType implements the Equaler interface
var _ convert.Equaler = SimpleType{}
var _ convert.Equaler = (*SimpleType)(nil)

// DefaultValue implements FieldType
func (t SimpleType) DefaultValue(value interface{}) (interface{}, error) {
	return value, nil
}

// Equal returns true if two SimpleType objects are equal; otherwise false is returned.
func (t SimpleType) Equal(u convert.Equaler) bool {
	other, ok := u.(SimpleType)
	if !ok {
		return false
	}
	return t.Kind == other.Kind
}

// GetKind implements FieldType
func (t SimpleType) GetKind() Kind {
	return t.Kind
}

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

// ConvertToModel implements the FieldType interface
func (t SimpleType) ConvertToModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindString, KindUser, KindIteration, KindArea, KindLabel, KindBoardColumn:
		if valueType.Kind() != reflect.String {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "string", valueType.Name())
		}
		return value, nil
	case KindURL:
		if valueType.Kind() == reflect.String && govalidator.IsURL(value.(string)) {
			return value, nil
		}
		return nil, errs.Errorf("value %v (%[1]T) should be %s, but is \"%s\"", value, "URL", valueType.Name())
	case KindFloat:
		if valueType.Kind() != reflect.Float64 {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is \"%s\"", value, "float64", valueType.Name())
		}
		return value, nil
	case KindInteger, KindDuration: // NOTE: Duration is a typedef of int64
		// NOTE(kwk): This will change soon to be more consistent.
		switch valueType.Kind() {
		case reflect.Int,
			reflect.Int64:
			return value, nil
		case reflect.Float64:
			fval, ok := value.(float64)
			if !ok {
				return nil, errs.Errorf("failed to cast value %+v (%[1]T) to float64", value)
			}
			if fval != math.Trunc(fval) {
				return nil, errs.Errorf("float64 value %+v (%[1]T) has digits after the decimal point and therefore cannot be represented by an integer", value, valueType.Name())
			}
			return int(fval), nil
		default:
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s ", value, "int", valueType.Name())
		}
	case KindInstant:
		// instant == milliseconds
		// if !valueType.Implements(timeType) {
		if valueType.Kind() != timeType.Kind() {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "time.Time", valueType.Name())
		}
		return value.(time.Time).UnixNano(), nil
	case KindList:
		if (valueType.Kind() != reflect.Array) && (valueType.Kind() != reflect.Slice) {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s,", value, "array/slice", valueType.Kind())
		}
		return value, nil
	case KindEnum:
		// to be done yet | not sure what to write here as of now.
		return value, nil
	case KindMarkup:
		// 'markup' is just a string in the API layer for now:
		// it corresponds to the MarkupContent.Content field. The MarkupContent.Markup is set to the default value
		switch value.(type) {
		case rendering.MarkupContent:
			markupContent := value.(rendering.MarkupContent)
			if !rendering.IsMarkupSupported(markupContent.Markup) {
				return nil, errs.Errorf("value %v (%[1]T) has no valid markup type %s", value, markupContent.Markup)
			}
			return markupContent.ToMap(), nil
		default:
			return nil, errs.Errorf("value %v (%[1]T) should be rendering.MarkupContent, but is %s", value, valueType)
		}
	case KindCodebase:
		switch value.(type) {
		case codebase.Content:
			cb := value.(codebase.Content)
			if err := cb.IsValid(); err != nil {
				return nil, errs.Wrapf(err, "value %v (%[1]T) is invalid %s", value, cb)
			}
			return cb.ToMap(), nil
		default:
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "CodebaseContent", valueType)
		}
	case KindBoolean:
		if valueType.Kind() != reflect.Bool {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "boolean", valueType.Name())
		}
		return value, nil
	default:
		return nil, errs.Errorf("unexpected type constant: '%s'", t.GetKind())
	}
}

// ConvertFromModel implements the t interface
func (t SimpleType) ConvertFromModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindString, KindURL, KindUser, KindInteger, KindFloat, KindDuration, KindIteration, KindArea, KindLabel, KindBoardColumn, KindBoolean:
		return value, nil
	case KindInstant:
		switch valueType.Kind() {
		case reflect.Float64:
			v, ok := value.(float64)
			if !ok {
				return nil, errs.Errorf("value %v could not be converted into an %s", value, reflect.Float64)
			}
			if v != math.Trunc(v) {
				return nil, errs.Errorf("value %v is not a whole number", value)
			}
			return time.Unix(0, int64(v)), nil
		case reflect.Int64:
			v, ok := value.(int64)
			if !ok {
				return nil, errs.Errorf("value %v could not be converted into an %s", value, reflect.Int64)
			}
			return time.Unix(0, v), nil
		default:
			return nil, errs.Errorf("value %v must be either %s or %s but has an unknown type %s", value, reflect.Int64, reflect.Float64, valueType.Name())
		}
	case KindMarkup:
		if valueType.Kind() != reflect.Map {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, reflect.Map, valueType.Name())
		}
		markupContent := rendering.NewMarkupContentFromMap(value.(map[string]interface{}))
		return markupContent, nil
	case KindCodebase:
		if valueType.Kind() != reflect.Map {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, reflect.Map, valueType.Name())
		}
		cb, err := codebase.NewCodebaseContent(value.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		return cb, nil
	default:
		return nil, errs.Errorf("unexpected field type: %s", t.GetKind())
	}
}
