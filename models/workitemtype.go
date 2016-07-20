package models

const (
	String Kind = iota
	Integer
	Float
	Instant
	Duration
	Url
	WorkitemReference
	User
	Enum
	List
)

type Kind byte

type FieldType interface {
	GetKind() Kind
	ConvertToModel(value interface{}) (interface{}, error)
	ConvertFromModel(value interface{}) (interface{}, error)
}

type WorkItemType struct {
	Id      uint64
	Version int
	Name    string
	Fields  map[string]FieldDefinition `sql:"type:jsonb"`
}
