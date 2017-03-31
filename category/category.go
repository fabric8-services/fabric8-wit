package category

import (
	"context"
	"time"

	"github.com/almighty/almighty-core/gormsupport"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Defines "type" string to be used while validating jsonapi spec based payload
const (
	APIStringTypeCategory = "categories"
)

// Category describes a single category
type Category struct {
	gormsupport.Lifecycle
	ID   uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Name string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *Category) TableName() string {
	return "categories"
}

// Repository describes interactions with Categories
type Repository interface {
	List(ctx context.Context) ([]*Category, error)
}

// NewCategoryRepository creates a new storage type.
func NewCategoryRepository(db *gorm.DB) Repository {
	return &GormCategoryRepository{db: db}
}

// GormCategoryRepository is the implementation of the storage interface for Categories.
type GormCategoryRepository struct {
	db *gorm.DB
}

// List all Categories related to a single item
func (m *GormCategoryRepository) List(ctx context.Context) ([]*Category, error) {
	defer goa.MeasureSince([]string{"goa", "db", "Category", "query"}, time.Now())
	var objs []*Category
	err := m.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}
