package workitem

import (
	"strconv"
	"github.com/fabric8-services/fabric8-wit/ptr"
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
	Kind         Kind        `json:"kind"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// Ensure SimpleType implements the FieldType interface
var _ FieldType = SimpleType{}
var _ FieldType = (*SimpleType)(nil)

// Ensure SimpleType implements the Equaler interface
var _ convert.Equaler = SimpleType{}
var _ convert.Equaler = (*SimpleType)(nil)

// Validate checks that the default value matches the Kind
func (t SimpleType) Validate() error {
	if !t.Kind.IsSimpleType() {
		return errs.New("a simple type can only have a simple type (e.g. no list or enum)")
	}
	_, err := t.SetDefaultValue(t.DefaultValue)
	if err != nil {
		return errs.Wrapf(err, "failed to validate default value for kind %s: %+v (%[1]T)", t.Kind, t.DefaultValue)
	}
	return nil
}

// SetDefaultValue implements FieldType
func (t SimpleType) SetDefaultValue(v interface{}) (FieldType, error) {
	if v == nil {
		t.DefaultValue = nil
		return t, nil
	}
	defVal, err := t.ConvertToModel(v)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to set default value of simple type to %+v (%[1]T)", v)
	}
	t.DefaultValue = defVal
	return t, nil
}

// GetDefaultValue implements FieldType
func (t SimpleType) GetDefaultValue() interface{} {
	return t.DefaultValue
}

// Equal returns true if two SimpleType objects are equal; otherwise false is returned.
func (t SimpleType) Equal(u convert.Equaler) bool {
	other, ok := u.(SimpleType)
	if !ok {
		return false
	}
	if t.DefaultValue != other.DefaultValue {
		return false
	}
	return t.Kind == other.Kind
}

// EqualValue implements convert.Equaler interface
func (t SimpleType) EqualValue(u convert.Equaler) bool {
	return t.Equal(u)
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
		return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %q", value, "URL", valueType.Name())
	case KindFloat:
		if valueType.Kind() != reflect.Float64 {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %q", value, "float64", valueType.Name())
		}
		return value, nil
	case KindInteger:
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
				return nil, errs.Errorf("float64 value %+v (%[1]T) has digits after the decimal point and therefore cannot be represented by an integer", value)
			}
			return int(fval), nil
		default:
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s ", value, "int or float", valueType.Name())
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
		case map[string]interface{}:
			markupContent := rendering.NewMarkupContentFromValue(value)
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

// ConvertToString implements the FieldType interface
func (t SimpleType) ConvertToString(value interface{}) (*string, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindString, KindUser, KindIteration, KindArea, KindLabel, KindBoardColumn:
		if valueType.Kind() != reflect.String {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "string", valueType.Name())
		}
		return ptr.String(value.(string)), nil
	case KindURL:
		if valueType.Kind() == reflect.String && govalidator.IsURL(value.(string)) {
			return ptr.String(value.(string)), nil
		}
		return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %q", value, "URL", valueType.Name())
	case KindFloat:
		if valueType.Kind() != reflect.Float64 {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %q", value, "float64", valueType.Name())
		}
		fval, ok := value.(float64)
		if !ok {
			return nil, errs.Errorf("failed to cast value %+v (%[1]T) to float64", value)
		}
		return ptr.String(strconv.FormatFloat(fval, 'f', 6, 64)), nil
	case KindInteger:
		// NOTE(kwk): This will change soon to be more consistent.
		switch valueType.Kind() {
		case reflect.Int:
			ival, ok := value.(int)
			if !ok {
				return nil, errs.Errorf("failed to cast value %+v (%[1]T) to int", value)
			}
			return ptr.String(strconv.Itoa(ival)), nil
		case reflect.Int64:
			ival, ok := value.(int64)
			if !ok {
				return nil, errs.Errorf("failed to cast value %+v (%[1]T) to int64", value)
			}
			return ptr.String(strconv.FormatInt((ival), 10)), nil
		case reflect.Float64:
			fval, ok := value.(float64)
			if !ok {
				return nil, errs.Errorf("failed to cast value %+v (%[1]T) to float64", value)
			}
			if fval != math.Trunc(fval) {
				return nil, errs.Errorf("float64 value %+v (%[1]T) has digits after the decimal point and therefore cannot be represented by an integer", value)
			}
			return ptr.String(strconv.FormatFloat(fval, 'f', 6, 64)), nil
		default:
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s ", value, "int or float", valueType.Name())
		}
	case KindInstant:
		// instant == milliseconds
		// if !valueType.Implements(timeType) {
		if valueType.Kind() != timeType.Kind() {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "time.Time", valueType.Name())
		}
		return ptr.String(strconv.FormatInt((value.(time.Time).UnixNano()), 10)), nil
	case KindList:
		/*
		if (valueType.Kind() != reflect.Array) && (valueType.Kind() != reflect.Slice) {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s,", value, "array/slice", valueType.Kind())
		}
		listStr := strings.Join(value, ";")
		return listStr, nil
		*/
		// TODO(michaelkleinhenz): implement
		return ptr.String("not supported"), nil
	case KindEnum:
		// to be done yet | not sure what to write here as of now.
		// TODO(michaelkleinhenz): implement
		return ptr.String("not supported"), nil
	case KindMarkup:
		// to be done yet | not sure what to write here as of now.
		// TODO(michaelkleinhenz): implement
		return ptr.String("not supported"), nil
	case KindCodebase:
		// to be done yet | not sure what to write here as of now.
		// TODO(michaelkleinhenz): implement
		return ptr.String("not supported"), nil
	case KindBoolean:
		if valueType.Kind() != reflect.Bool {
			return nil, errs.Errorf("value %v (%[1]T) should be %s, but is %s", value, "boolean", valueType.Name())
		}
		if value.(bool) {
			return ptr.String("true"), nil
		}
		return ptr.String("false"), nil
	default:
		return nil, errs.Errorf("unexpected type constant: '%s'", t.GetKind())
	}
}

// ConvertFromModel implements the FieldType interface
func (t SimpleType) ConvertFromModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindString, KindURL, KindUser, KindInteger, KindFloat, KindIteration, KindArea, KindLabel, KindBoardColumn, KindBoolean:
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

// ConvertToModelWithType implements FieldType
func (t SimpleType) ConvertToModelWithType(newFieldType FieldType, v interface{}) (interface{}, error) {
	// Try to assign the old value to the new field
	newVal, err := newFieldType.ConvertToModel(v)
	if err == nil {
		return newVal, nil
	}

	// if the new type is a list, stuff the old value in a list and
	// try to assign it
	if newFieldType.GetKind() == KindList {
		newVal, err = newFieldType.ConvertToModel([]interface{}{v})
		if err == nil {
			return newVal, nil
		}
	}

	// if the old type is a list but the new one isn't check that
	// the list contains only one element and assign that
	if t.GetKind() == KindList && newFieldType.GetKind() != KindList {
		ifArr, ok := v.([]interface{})
		if !ok {
			return nil, errs.Errorf("failed to convert value to interface array: %+v", v)
		}
		if len(ifArr) == 1 {
			newVal, err = newFieldType.ConvertToModel(ifArr[0])
			if err == nil {
				return newVal, nil
			}
		}
	}
	return nil, errs.Errorf("failed to convert value %+v (%[1]T) to field type %+v (%[2]T)", v, newFieldType)
}
