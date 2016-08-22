package models

import (
	"encoding/json"
	"fmt"
)

// constants for describing possible field types
const (
	KindString            Kind = "string"
	KindInteger           Kind = "integer"
	KindFloat             Kind = "float"
	KindInstant           Kind = "instant"
	KindDuration          Kind = "duration"
	KindURL               Kind = "url"
	KindWorkitemReference Kind = "workitem"
	KindUser              Kind = "user"
	KindEnum              Kind = "enum"
	KindList              Kind = "list"
)

// Kind is the kind of field type
type Kind string

/*
FieldType describes the possible values of a FieldDefinition
*/
type FieldType interface {
	GetKind() Kind
	/*
	   ConvertToModel converts a field value for use in the REST API
	*/
	ConvertToModel(value interface{}) (interface{}, error)
	/*
		ConvertToModel converts a field value for storage in the db
	*/
	ConvertFromModel(value interface{}) (interface{}, error)
}

/*
FieldDefintion describes type & other restrictions of a field
*/
type FieldDefinition struct {
	Required bool
	Type     FieldType
}

/*
 ConvertToModel converts a field value for storage as json. As the system matures, add more checks (for example whether a user is in the system, etc.)
*/
func (f FieldDefinition) ConvertToModel(name string, value interface{}) (interface{}, error) {
	if f.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return f.Type.ConvertToModel(value)
}

/*
ConvertFromModel converts from json storage to API form.
*/
func (f FieldDefinition) ConvertFromModel(name string, value interface{}) (interface{}, error) {
	if f.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return f.Type.ConvertFromModel(value)
}

type rawFieldDef struct {
	Required bool
	Type     *json.RawMessage
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (f *FieldDefinition) UnmarshalJSON(bytes []byte) error {
	temp := rawFieldDef{}

	fmt.Printf(string(bytes))
	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return err
	}
	rawType := map[string]interface{}{}
	json.Unmarshal(*temp.Type, &rawType)

	kind, ok := rawType["Kind"].(string)
	if !ok {
		return fmt.Errorf("Kind is not a kind value")
	}

	switch {
	case kind == string(KindList):
		theType := ListType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return err
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required}
	case kind == string(KindEnum):
		theType := EnumType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return err
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required}
	default:
		theType := SimpleType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return err
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required}
	}
	return nil
}
