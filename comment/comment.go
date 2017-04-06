package comment

import (
	"strconv"
	"time"

	"github.com/almighty/almighty-core/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Comment describes a single comment
type Comment struct {
	gormsupport.Lifecycle
	ID        uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	ParentID  string
	CreatedBy uuid.UUID `sql:"type:uuid"` // Belongs To Identity
	Body      string
	Markup    string
}

// GetETagData returns the field values to use to generate the ETag
func (m Comment) GetETagData() []interface{} {
	// using the 'ID' and 'UpdatedAt' (converted to number of seconds since epoch) fields
	return []interface{}{m.ID, strconv.FormatInt(m.UpdatedAt.Unix(), 10)}
}

// GetLastModified returns the last modification time
func (m Comment) GetLastModified() time.Time {
	return m.UpdatedAt.Truncate(time.Second)
}
