package comment

import (
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
