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

// LtreePath type helps in storing and retrieving ltree fields to and from databse.
// It uses ltree knowledge while converting values. https://www.postgresql.org/docs/9.1/static/ltree.html
// It is just slice of UUIDs which will be saved to database in following format
// 4dd8f038_3fc4_48ab_ad4d_197ccc7b44a2.62ea5454_f8d5_4b35_8589_8d646d612250.c9b24b8f_8b33_4c22_82f0_8eb0a5b9837e
// Above content will be read from database as a slice of UUIDs
type LtreePath []uuid.UUID

// Following constants are used while saving and displaying LtreePath type to and from database
const (
	SepInService  = "/"
	SepInDatabase = "."
)

// IsEmpty checks count of items in LtreePath
func (p LtreePath) IsEmpty() bool {
	return len(p) == 0
}

// This returns last UUID of slice
func (p LtreePath) This() uuid.UUID {
	if len(p) > 0 {
		return p[len(p)-1]
	}
	return uuid.Nil
}

// Convert returns ltree compatible string of UUID slice
// 39fa8c5b_5732_436f_a084_0f2a247f3435.87762c8b_17f9_4cbb_b355_251b6a524f2f
func (p LtreePath) Convert() string {
	if len(p) == 0 {
		return ""
	}
	var op []string
	for _, id := range p {
		op = append(op, id.String())
	}
	str := strings.Join(op, SepInDatabase)
	return strings.Replace(str, "-", "_", -1)
}

// String converts the LtreePath to representable format in string
// Currently separator is '/'
// /87762c8b-17f9-4cbb-b355-251b6a524f2f/39fa8c5b-5732-436f-a084-0f2a247f3435/be54f2c4-cfa4-47af-ad06-280fba540872
func (p LtreePath) String() string {
	var op []string
	if len(p) == 0 {
		return SepInService
	}
	for _, id := range p {
		op = append(op, id.String())
	}
	str := strings.Join(op, SepInService)
	return SepInService + str
}

// Root retunrs a LtreePath instance with first element in the UUID slice
func (p LtreePath) Root() LtreePath {
	if len(p) > 0 {
		return LtreePath{p[0]}
	}
	return LtreePath{uuid.Nil}
}

// Parent returns a LtreePath instance with last element in the UUID slice
// Similar to This but following funtion returns LtreePath instance and not just UUID
func (p LtreePath) Parent() LtreePath {
	if len(p) > 0 {
		return LtreePath{p[len(p)-1]}
	}
	return LtreePath{uuid.Nil}
}

func (p LtreePath) convertToLtree(id uuid.UUID) string {
	converted := strings.Replace(id.String(), "-", "_", -1)
	return converted
}

func (p LtreePath) convertFromLtree(uuidStr string) ([]uuid.UUID, error) {
	// Ltree allows only "_" as a special character.
	converted := strings.Replace(uuidStr, "_", "-", -1)
	parts := strings.Split(converted, SepInDatabase)
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

// Value helps in implementing Valuer interfae on LtreePath
func (p LtreePath) Value() (driver.Value, error) {
	op := []string{}
	for _, x := range p {
		op = append(op, p.convertToLtree(x))
	}
	s := strings.Join(op, SepInDatabase)
	return s, nil
}

// Scan helps in implementing Scanner interface on LtreePath
func (p *LtreePath) Scan(value interface{}) error {
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

// MarshalJSON allows LtreePath to be serialized
func (p LtreePath) MarshalJSON() ([]byte, error) {
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

// UnmarshalJSON allows LtreePath to be deserialized
func (p *LtreePath) UnmarshalJSON(b []byte) error {
	var stringMap map[string]string
	err := json.Unmarshal(b, &stringMap)
	if err != nil {
		return err
	}
	for _, value := range stringMap {
		id, err := uuid.FromString(value)
		if err != nil {
			return err
		}
		*p = append(*p, id)
	}
	return nil
}

// ToExpression returns a string in ltree format.
// Joins UUIDs in the first argument using `.`
// Second argument is converted and appended if needed
func ToExpression(p LtreePath, this uuid.UUID) string {
	converted := strings.Replace(this.String(), "-", "_", -1)
	existingPath := p.Convert()
	if existingPath == "" {
		return converted
	}
	return fmt.Sprintf("%s.%s", p.Convert(), converted)
}
