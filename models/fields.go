package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Fields map[string]interface{}

func (j Fields) Value() (driver.Value, error) {
	if j == nil {
		//      log.Trace("returning null")
		return nil, nil
	}

	res, error := json.Marshal(j)
	return res, error
}

func (j *Fields) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}
	s, ok := src.([]byte)
	if !ok {
		return errors.New("Scan source was not string")
	}
	res:= Fields{}
	err := json.Unmarshal(s, &res)
	*j=res;

	return err
}
