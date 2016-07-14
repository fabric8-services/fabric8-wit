package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Fields map[string]interface{}

func (j Fields) Value() (driver.Value, error) {
	return toBytes(j);
}

func (j *Fields) Scan(src interface{}) error {
	return fromBytes(src, j)
}

func (j FieldDefinition) Value() (driver.Value, error) {
	return toBytes(j);
}

func (j *FieldDefinition) Scan(src interface{}) error {
	return fromBytes(src, j)
}


func toBytes(j interface{}) (driver.Value, error) {
	if j == nil {
		// log.Trace("returning null")
		return nil, nil
	}

	res, error := json.Marshal(j)
	return res, error
}


func fromBytes(src interface{}, target interface{}) error {
	if src == nil {
		target = nil
		return nil
	}
	s, ok := src.([]byte)
	if !ok {
		return errors.New("Scan source was not string")
	}
	return json.Unmarshal(s, target)
}
