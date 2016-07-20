package models

import(
	"fmt"
	"encoding/json"
	"bytes"
)

type FieldDefinition struct {
	Type     FieldType
	Required bool
}

/*
 Convert for storage as json. As the system matures, add more checks (for example whether a user is in the system, etc.)
*/
func (field FieldDefinition) ConvertToModel(name string, value interface{}) (interface{}, error) {
	if field.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return field.Type.ConvertToModel(value)
}

func (field FieldDefinition) ConvertFromModel(name string, value interface{}) (interface{}, error) {
	if field.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return field.Type.ConvertFromModel(value)
}

type rawFieldDef struct {
	Type     rawFieldType
	Required bool
}

type rawEnumType struct {
	BaseType SimpleType
	Values   []interface{}
}

type rawFieldType struct {
	Kind  Kind
	Extra *json.RawMessage
}

func (self *FieldDefinition) UnmarshalJSON(bytes []byte) error {

	temp := rawFieldDef{}

	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return err
	}

	switch temp.Type.Kind {
	case List:
		var baseType SimpleType
		err = json.Unmarshal(*temp.Type.Extra, &baseType)
		if err != nil {
			return err
		}
		theType := ListType{SimpleType: SimpleType{Kind: temp.Type.Kind}, ComponentType: baseType}
		*self = FieldDefinition{Type: theType, Required: temp.Required}
	case Enum:
		var extraInfo rawEnumType
		err = json.Unmarshal(*temp.Type.Extra, &extraInfo)
		if err != nil {
			return err
		}
		theType := EnumType{SimpleType: SimpleType{Kind: temp.Type.Kind}, BaseType: extraInfo.BaseType, Values: extraInfo.Values}
		*self = FieldDefinition{Type: theType, Required: temp.Required}
	default:
		*self = FieldDefinition{Type: SimpleType{Kind: temp.Type.Kind}, Required: temp.Required}
	}
	return nil
}

func (self FieldDefinition) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{ \"type\": {")
	buf.WriteString(fmt.Sprintf("\"kind\": %d", self.Type.GetKind()))
	switch complexType := self.Type.(type) {
	case ListType:
		buf.WriteString(", \"extra\": ")
		v, err := json.Marshal(complexType.ComponentType)
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	case EnumType:
		buf.WriteString(", \"extra\": ")
		r := rawEnumType{
			BaseType: complexType.BaseType,
			Values:   complexType.Values,
		}
		v, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		buf.Write(v)
	}
	buf.WriteString("}")

	buf.WriteString(", \"required\": ")
	if self.Required {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}

	buf.WriteString(" }")
	return buf.Bytes(), nil
}
