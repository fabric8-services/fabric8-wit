package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
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
	Values []interface{}
}

type FieldTypes map[string]interface{}

type WorkItemType struct {
	Id      uint64
	Version int
	Name    string
	Fields  FieldTypes `sql:"type:jsonb"`
}

func (self *FieldTypes) UnmarshalJSON(bytes []byte) error {
	type fieldType struct {
		Kind  Kind
		Extra *json.RawMessage
	}
	temp := map[string]fieldType{}

	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return err
	}

	*self = FieldTypes{}

	for n, t := range temp {
		switch t.Kind {
		case List:
			var baseType SimpleType
			err = json.Unmarshal(*t.Extra, &baseType)
			if err != nil {
				return err
			}
			(*self)[n] = ListType{SimpleType: SimpleType{Kind: t.Kind}, ComponentType: baseType}
		case Enum:
			var values []interface{}
			json.Unmarshal(*t.Extra, &values)
			if err != nil {
				return err
			}
			(*self)[n] = EnumType{SimpleType: SimpleType{Kind: t.Kind}, Values: values}
		default:
			(*self)[n] = SimpleType{Kind: t.Kind}
		}
	}
	return nil
}

func (self FieldTypes) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("{")
	first := true
	for key, value := range self {
		t, ok := value.(FieldType)
		if !ok {
			panic(fmt.Sprintf("not a type: %v", reflect.TypeOf(value)))
		}
		if !first {
			first = false
			buf.WriteString(", ")
		}
		buf.WriteString("\"")
		buf.WriteString(key)
		buf.WriteString("\": {")
		buf.WriteString(fmt.Sprintf("\"kind\": %d", t.GetKind()))
		switch complexType := value.(type) {
		case ListType:
			buf.WriteString(", \"extra\": ")
			v, err := json.Marshal(complexType.ComponentType)
			if err != nil {
				return nil, err
			}
			buf.Write(v)
		case EnumType:
			buf.WriteString(", \"extra\": ")
			v, err := json.Marshal(complexType.Values)
			if err != nil {
				return nil, err
			}
			buf.Write(v)
		}
		buf.WriteString("}")
	}

	buf.WriteString(" }")
	return buf.Bytes(), nil
}
