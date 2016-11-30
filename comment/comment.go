package comment

import (
	"time"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Comment describes a single comment
type Comment struct {
	gormsupport.Lifecycle
	ID        uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	ParentID  string
	CreatedBy uuid.UUID `sql:"type:uuid"` // Belongs To Identity
	Body      string
}

// Repository describes interactions with comments
type Repository interface {
	Create(ctx context.Context, u *Comment) error
	List(ctx context.Context, parent string) ([]*Comment, error)
}

// NewCommentRepository creates a new storage type.
func NewCommentRepository(db *gorm.DB) Repository {
	return &GormCommentRepository{db: db}
}

// GormCommentRepository is the implementation of the storage interface for Comments.
type GormCommentRepository struct {
	db *gorm.DB
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormCommentRepository) TableName() string {
	return "comments"
}

// Create creates a new record.
func (m *GormCommentRepository) Create(ctx context.Context, u *Comment) error {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "create"}, time.Now())

	u.ID = uuid.NewV4()

	err := m.db.Create(u).Error
	if err != nil {
		goa.LogError(ctx, "error adding Comment", "error", err.Error())
		return err
	}

	return nil
}

// List all comments related to a single item
func (m *GormCommentRepository) List(ctx context.Context, parent string) ([]*Comment, error) {
	defer goa.MeasureSince([]string{"goa", "db", "comment", "query"}, time.Now())
	var objs []*Comment

	err := m.db.Table(m.TableName()).Where("parent_id = ?", parent).Order("created_at").Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}
