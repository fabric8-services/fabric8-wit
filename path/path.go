package path

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	uuid "github.com/satori/go.uuid"
)

// Path type helps in storing and retrieving ltree fields to and from databse.
// It uses ltree knowledge while converting values. https://www.postgresql.org/docs/9.1/static/ltree.html
// It is just slice of UUIDs which will be saved to database in following format
// 4dd8f038-3fc4-48ab-ad4d-197ccc7b44a2.62ea5454-f8d5-4b35-8589-8d646d612250.c9b24b8f-8b33-4c22-82f0-8eb0a5b9837e
// Above content will be read from database as a slice of UUIDs
type Path []uuid.UUID

const (
	sepInService  = "/"
	sepInDatabase = "."
)

func (p Path) IsEmpty() bool {
	return len(p) == 0
}

// This returns last element of the UUID slice
func (p Path) This() uuid.UUID {
	if len(p) > 0 {
		return p[len(p)-1]
	}
	return uuid.UUID{}
}

// This returns last element of the UUID slice
func (p Path) Convert() string {
	if len(p) == 0 {
		return ""
	}
	return p.ReprDB()
}

// ToDo :- add Resolved()

// String converts the Path to representable format in string
// Currently separator is '/'
func (p Path) String() string {
	var op []string
	if len(p) == 0 {
		return sepInService
	}
	for _, id := range p {
		op = append(op, id.String())
	}
	return strings.Join(op, sepInService)
}

// ReprDB returns value like stored in DB
func (p Path) ReprDB() string {
	var op []string
	for _, id := range p {
		op = append(op, id.String())
	}
	str := strings.Join(op, sepInDatabase)
	return strings.Replace(str, "-", "_", -1)
}

// Root retunrs a Path instance with first element in the UUID slice
func (p Path) Root() Path {
	if len(p) > 0 {
		return Path{p[0]}
	}
	return Path{uuid.UUID{}}
}

// Parent returns a Path instance with last element in the UUID slice
// Similar to This but following funtion returns Path instance and not just UUID
func (p Path) Parent() Path {
	if len(p) > 0 {
		return Path{p[len(p)-1]}
	}
	return Path{uuid.UUID{}}
}

func (p Path) convertToLtree(id uuid.UUID) string {
	converted := strings.Replace(id.String(), "-", "_", -1)
	return converted
}

func (p Path) convertFromLtree(uuidStr string) ([]uuid.UUID, error) {
	// Ltree allows only "_" as a special character.
	converted := strings.Replace(uuidStr, "_", "-", -1)
	parts := strings.Split(converted, sepInDatabase)
	op := []uuid.UUID{}
	for _, x := range parts {
		id, err := uuid.FromString(x)
		if err != nil {
			return nil, err
		}
		op = append(op, id)
	}
	return op, nil
}

// Value helps in implementing Valuer interfae on Path
func (p Path) Value() (driver.Value, error) {
	op := []string{}
	for _, x := range p {
		op = append(op, p.convertToLtree(x))
	}
	s := strings.Join(op, sepInDatabase)
	fmt.Println("Valuer -> ", s)
	return s, nil
}

// Scan helps in implementing Scanner interface on Path
func (p *Path) Scan(value interface{}) error {
	// if value is nil, false
	if value == nil {
		*p = []uuid.UUID{}
		return nil
	}
	if str, err := driver.String.ConvertValue(value); err == nil {
		// if this is a string type
		v := ""
		for _, m := range str.([]uint8) {
			v += string(m)
		}
		all, err := p.convertFromLtree(v)
		if err != nil {
			*p = []uuid.UUID{}
			return nil
		}
		*p = all
		return nil
	}
	// otherwise, return an error
	return errors.New("failed to scan MyPath")
}

// MarshalJSON allows Path to be serialized
func (p Path) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	length := len(p)
	count := 0
	for key, value := range p {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		buffer.WriteString(fmt.Sprintf("\"%d\":%s", key, string(jsonValue)))
		count++
		if count < length {
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

// UnmarshalJSON allows Path to be deserialized
func (p *Path) UnmarshalJSON(b []byte) error {
	var stringMap map[string]string
	err := json.Unmarshal(b, &stringMap)
	if err != nil {
		return err
	}
	for _, value := range stringMap {
		id, err := uuid.FromString(value)
		fmt.Println("v = ", id, err)
		if err != nil {
			return err
		}
		*p = append(*p, id)
	}
	return nil
}
