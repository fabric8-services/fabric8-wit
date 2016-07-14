package models

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	String Kind = iota
	Integer
	Float
	Instant
	Interval
	Url
	WorkitemReference
	User
	Enum
	List
)

type Kind byte

type FieldType interface {
	GetKind() Kind
}

func (self SimpleType) GetKind() Kind {
	return self.Kind
}

// simple types
type SimpleType struct {
	Kind Kind
}

type ListType struct {
	SimpleType
	ComponentType SimpleType
}

type EnumType struct {
	SimpleType
	BaseType SimpleType
	Values   []interface{}
}

type FieldDefinition struct {
	Type     FieldType
	Required bool
}

type FieldDefinitions map[string]FieldDefinition

type WorkItemType struct {
	Id      uint64
	Version int
	Name    string
	Fields  FieldDefinitions `sql:"type:jsonb"`
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
