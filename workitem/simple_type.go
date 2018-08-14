package workitem

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/asaskevich/govalidator"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/rendering"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
	if t.DefaultValue != nil {
		_, err := t.ConvertToModel(t.DefaultValue)
		if err != nil {
			return errs.Wrapf(err, "failed to convert value to kind %s: %+v", t.Kind, t.DefaultValue)
		}
	}
	return nil
}

// GetDefaultValue implements FieldType
func (t SimpleType) GetDefaultValue(value interface{}) (interface{}, error) {
	if err := t.Validate(); err != nil {
		return nil, errs.Wrapf(err, "failed to validate simple type")
	}
	if value != nil {
		v, err := t.ConvertToModel(value)
		if err != nil {
			return nil, errs.Wrapf(err, `value "%+v" is not a valid simple value`)
		}
		return v, nil
	}
	if t.DefaultValue != nil {
		return t.DefaultValue, nil
	}
	return value, nil
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

// GetKind implements FieldType
func (t SimpleType) GetKind() Kind {
	return t.Kind
}

var timeType = reflect.TypeOf((*time.Time)(nil)).Elem()

// convertNumberToInt32 can take any integer value and it convert it to an int32
func convertNumberToInt32(value interface{}) (int32, error) {
	if value == nil {
		return 0, nil
	}

	valueType := reflect.TypeOf(value)

	// Check if the value even though it is recognized as a float can also
	// be represented as an integer.
	switch valueType.Kind() {
	case reflect.String:
		// try to parse as string or json.Number
		var str string
		str, ok := value.(string)
		if !ok {
			jsonNumber, ok := value.(json.Number)
			if !ok {
				return 0, errs.Errorf("failed to convert to value %+v (%[1]T) to string or json.Number", value)
			}
			str = string(jsonNumber)
		}
		ival, errInt := strconv.ParseInt(string(str), 10, 64)
		if errInt == nil {
			return convertNumberToInt32(ival)
		}
		// try parsing as float
		fval, errFloat := strconv.ParseFloat(string(str), 64)
		if errFloat == nil {
			return convertNumberToInt32(fval)
		}
		return 0, errs.Errorf("failed to parse value %+v (%[1]T) as int64 or float64: %s, %s", errInt, errFloat)
	case reflect.Int:
		ival := value.(int)
		if ival > math.MaxInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be lesser or equal to %d", value, math.MaxInt32)
		}
		if ival < math.MinInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be greater or equal to %d", value, math.MinInt32)
		}
		return int32(ival), nil
	case reflect.Int8:
		return int32(value.(int8)), nil
	case reflect.Int16:
		return int32(value.(int16)), nil
	case reflect.Int32:
		return int32(value.(int32)), nil
	case reflect.Int64:
		ival := value.(int64)
		if ival > math.MaxInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be lesser or equal to %d", value, math.MaxInt32)
		}
		if ival < math.MinInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be greater or equal to %d", value, math.MinInt32)
		}
		return int32(ival), nil
	case reflect.Uint:
		uival := value.(uint)
		if uival > math.MaxInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be lesser or equal to %d", value, math.MaxInt32)
		}
		return int32(uival), nil
	case reflect.Uint8:
		return int32(value.(uint8)), nil
	case reflect.Uint16:
		return int32(value.(uint16)), nil
	case reflect.Uint32:
		return int32(value.(uint32)), nil
	case reflect.Uint64:
		uival := value.(uint64)
		if uival > math.MaxInt32 {
			return 0, errs.Errorf("integer value %+v (%[1]T) must be lesser or equal to %d", value, math.MaxInt32)
		}
		return int32(uival), nil
	case reflect.Float64, reflect.Float32:
		fval, err := convertNumberToFloat64(value)
		if err != nil {
			return 0, errs.WithStack(err)
		}
		if math.Trunc(fval) != fval {
			return 0, errs.Errorf("floating point value %+v (%[1]T) has decimal points and therefore cannot be represented as an int", value)
		}
		if fval > float64(math.MaxInt64) {
			return 0, errs.Errorf("value %+v (%[1]T) must be lesser or equal to %d", fval, math.MaxInt32)
		}
		if fval < float64(math.MinInt64) {
			return 0, errs.Errorf("value %+v (%[1]T) must be greater or equal to %d", fval, math.MinInt32)
		}
		return int32(fval), nil
	}
	return 0, errs.Errorf("value %+v (%[1]T) is not representable by an integer type: %s", value, valueType.Name())
}

const (
	maxAcurateInt64InFloat64 = int64((1 << 54) - 2)
	minAcurateInt64InFloat64 = -1 * int64(1<<54-2)
)

func convertNumberToFloat64(value interface{}) (float64, error) {
	if value == nil {
		return 0, nil
	}

	var fval float64
	valueType := reflect.TypeOf(value)
	switch valueType.Kind() {
	case reflect.String:
		var str string
		str, ok := value.(string)
		if !ok {
			jsonNumber, ok := value.(json.Number)
			if !ok {
				return 0, errs.Errorf("failed to convert to value %+v (%[1]T) to string or json.Number", value)
			}
			str = string(jsonNumber)
		}
		fval, errFloat := strconv.ParseFloat(string(str), 64)
		if errFloat == nil {
			return convertNumberToFloat64(fval)
		}
		// try parsing as int
		ival, errInt := strconv.ParseInt(string(str), 10, 64)
		if errInt == nil {
			return convertNumberToFloat64(ival)
		}
		return 0, errs.Errorf("failed to parse value %+v (%[1]T) as float64 or int64: %s, %s", errFloat, errInt)
	case reflect.Float64:
		fval = float64(value.(float64))
	case reflect.Float32:
		fval = float64(value.(float32))
	case reflect.Int:
		fval = float64(value.(int))
	case reflect.Int8:
		fval = float64(value.(int8))
	case reflect.Int16:
		fval = float64(value.(int16))
	case reflect.Int32:
		fval = float64(value.(int32))
	case reflect.Int64:
		ival := value.(int64)
		if ival > maxAcurateInt64InFloat64 {
			return 0, errs.Errorf("64-bit integer value must be smaller than %d in order to be acurately represented by a float64", maxAcurateInt64InFloat64, ival)
		}
		if ival < minAcurateInt64InFloat64 {
			return 0, errs.Errorf("64-bit integer value must be larger than %d in order to be acurately represented by a float64", minAcurateInt64InFloat64, ival)
		}
		fval = float64(ival)
	case reflect.Uint:
		fval = float64(value.(uint))
	case reflect.Uint8:
		fval = float64(value.(uint8))
	case reflect.Uint16:
		fval = float64(value.(uint16))
	case reflect.Uint32:
		fval = float64(value.(uint32))
	case reflect.Uint64:
		uival := value.(uint64)
		if uival > uint64(maxAcurateInt64InFloat64) {
			return 0, errs.Errorf("unsigned 64-bit integer value must be smaller than %d in order to be acurately represented by a float64", maxAcurateInt64InFloat64, uival)
		}
		fval = float64(uival)
	default:
		return 0, errs.Errorf("value %+v (%[1]T) should be a floating point number, but is %s", value, reflect.TypeOf(value).Name())
	}
	if math.IsInf(fval, 0) {
		return 0, errs.Errorf("value %v (%T) is infinity", fval)
	}
	if math.IsNaN(fval) {
		return 0, errs.Errorf("value %v (%[1]T) is not a number", fval)
	}
	return fval, nil
}

// toUnix converts a unix timestamp with our without nano second precision to
// just seconds.
func toUnix(v int64) int64 {
	if v > 1000000000000 {
		return int64(v / 1000000000.0)
	}
	return v
}

func convertAnyToTime(value interface{}) (time.Time, error) {
	switch t := value.(type) {
	case float64:
		return time.Unix(toUnix(int64(t)), 0).UTC(), nil
	case int64:
		return time.Unix(toUnix(int64(t)), 0).UTC(), nil
	case string:
		timeVal, err := dateparse.ParseAny(t)
		if err != nil {
			return time.Time{}, errs.Wrapf(err, "failed to parse string to time.Time: %+v", t)
		}
		return timeVal.UTC(), nil
	case time.Time:
		return t.UTC(), nil
	}
	return time.Time{}, errs.Errorf(`wrong input format for kind "%s": %T`, KindInstant, value)
}

func ConvertAnyToUUID(value interface{}) (uuid.UUID, error) {
	switch t := value.(type) {
	case string:
		u, err := uuid.FromString(t)
		if err != nil {
			return uuid.Nil, errs.Wrapf(err, "failed to parse string as UUID: %v", t)
		}
		return u, nil
	case uuid.UUID:
		return t, nil
	}
	return uuid.Nil, errs.Errorf("value %v should be of type string or uuid.UUID, but is %s", value, reflect.TypeOf(value).Name())
}

// ConvertToModel implements the FieldType interface
// This function is called when a client sends values.
func (t SimpleType) ConvertToModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindUser, KindIteration, KindArea, KindLabel, KindBoardColumn:
		return ConvertAnyToUUID(value)
	case KindString:
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
		return convertNumberToFloat64(value)
	case KindInteger: // NOTE: Duration is a typedef of int64
		return convertNumberToInt32(value)
	case KindDuration:
		dur, ok := value.(time.Duration)
		if !ok {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, "time.Duration", valueType.Name())
		}
		return dur, nil
	case KindInstant:
		timeVal, err := convertAnyToTime(value)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		return timeVal.Unix(), nil
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
	}
	return nil, errs.Errorf("unexpected type constant: '%s'", t.GetKind())
}

// ConvertFromModel implements the t interface ConvertFromModel converts the
// value from the storage representation to client representation
func (t SimpleType) ConvertFromModel(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	valueType := reflect.TypeOf(value)
	switch t.GetKind() {
	case KindUser, KindIteration, KindArea, KindLabel, KindBoardColumn:
		return ConvertAnyToUUID(value)
	case KindBoolean:
		if valueType.Kind() != reflect.Bool {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, "boolean", valueType.Name())
		}
		return value, nil
	case KindURL:
		if valueType.Kind() == reflect.String && govalidator.IsURL(value.(string)) {
			return value, nil
		}
		return nil, errs.Errorf("value %v should be %s, but is %s", value, "URL", valueType.Name())
	case KindString:
		if valueType.Kind() != reflect.String {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, "string", valueType.Name())
		}
		return value, nil
	case KindFloat:
		return convertNumberToFloat64(value)
	case KindInteger:
		return convertNumberToInt32(value)
	case KindDuration: // NOTE: Duration is a typedef of int64
		dur, ok := value.(time.Duration)
		if !ok {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, "time.Duration", valueType.Name())
		}
		return dur, nil
	case KindInstant:
		return convertAnyToTime(value)
	case KindMarkup:
		if valueType.Kind() != reflect.Map {
			return nil, errs.Errorf("value %v should be %s, but is %s", value, reflect.Map, valueType.Name())
		}
		markupContent := rendering.NewMarkupContentFromMap(value.(map[string]interface{}))
		return markupContent, nil

	// case KindMarkup:
	// 	// 'markup' is just a string in the API layer for now:
	// 	// it corresponds to the MarkupContent.Content field. The MarkupContent.Markup is set to the default value
	// 	switch value.(type) {
	// 	case rendering.MarkupContent:
	// 		markupContent := value.(rendering.MarkupContent)
	// 		if !rendering.IsMarkupSupported(markupContent.Markup) {
	// 			return nil, errs.Errorf("value %v (type %s) has no valid markup type %s", value, "MarkupContent", markupContent.Markup)
	// 		}
	// 		return markupContent.ToMap(), nil
	// 	default:
	// 		return nil, errs.Errorf("value %v should be %s, but is %s", value, "MarkupContent", valueType)
	// 	}

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
