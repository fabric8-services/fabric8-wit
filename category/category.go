package category

import (
	"context"
	"fmt"
	"strings"

	"github.com/almighty/almighty-core/convert"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Defines "type" string to be used while validating jsonapi spec based payload
const (
	APIStringType = "categories"
)

// String constants for system categories
const (
	PlannerRequirements = "planner.requirements"
	PlannerPortfolio    = "planner.portfolio"
)

// WARNING: Do not change these UUIDs!
// System defined categories
var (
	PlannerRequirementsID = uuid.FromStringOrNil("b51ccf69-d574-41c5-b738-4a69265129d1") // "planner.requirements"
	PlannerPortfolioID    = uuid.FromStringOrNil("0625e4bf-122a-4c1c-8ccd-e0995ef31974") // "planner.portfolio"
)

// Category describes a single category
type Category struct {
	gormsupport.Lifecycle
	ID   uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Name string
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (category *Category) TableName() string {
	return "categories"
}

// WorkItemTypeCategoryRelationship describes relationship between a category and a workitemtype.
type WorkItemTypeCategoryRelationship struct {
	gormsupport.Lifecycle
	ID             uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	CategoryID     uuid.UUID `sql:"type:uuid"`
	WorkItemTypeID uuid.UUID `sql:"type:uuid"`
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *WorkItemTypeCategoryRelationship) TableName() string {
	return "work_item_type_categories"
}

// Ensure fields implements the Equaler interface
var _ convert.Equaler = Category{}
var _ convert.Equaler = (*Category)(nil)

// Equal returns true if two Category objects are equal; otherwise false is returned.
func (category Category) Equal(u convert.Equaler) bool {
	other, ok := u.(Category)
	if !ok {
		return false
	}
	if !uuid.Equal(category.ID, other.ID) {
		return false
	}
	if !category.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if category.Name != other.Name {
		return false
	}
	return true
}

// Repository encapsulates storage and retrieval of categories
type Repository interface {
	Create(ctx context.Context, category *Category) (*Category, error)
	LoadCategory(ctx context.Context, id uuid.UUID) (*Category, error)
	List(ctx context.Context) ([]*Category, error)
	AssociateWIT(ctx context.Context, relationship *WorkItemTypeCategoryRelationship) error
	LoadAllRelationshipsOfCategory(ctx context.Context, categoryID uuid.UUID) ([]*WorkItemTypeCategoryRelationship, error)
	Save(ctx context.Context, category *Category) (*Category, error)
}

// NewRepository creates a new storage type.
func NewRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// GormRepository is the implementation of the storage interface for Categories.
type GormRepository struct {
	db *gorm.DB
}

// List all Categories.
func (m *GormRepository) List(ctx context.Context) ([]*Category, error) {
	var objs []*Category
	err := m.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to list categories")
		return nil, errors.NewInternalError(errs.Wrap(err, "failed to list categories"))
	}
	return objs, nil
}

// AssociateWIT creates relationship between workitemtype and category
func (m *GormRepository) AssociateWIT(ctx context.Context, relationship *WorkItemTypeCategoryRelationship) error {
	db := m.db.Create(relationship)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "work_item_type_categories_idx") {
			return errors.NewBadParameterError("category+workitemtype", relationship.CategoryID).Expected("unique")
		}
		log.Error(ctx, map[string]interface{}{
			"category_id": relationship.CategoryID,
			"wit_id":      relationship.WorkItemTypeID,
			"err":         db.Error.Error(),
		}, "unable to create workitemtype category relationship")
		return errors.NewInternalError(errs.Wrap(db.Error, "unable to create workitemtype category relationship"))
	}
	return nil
}

// Create creates category. This function is used to populate categories table during migration -> PopulateCategories()
func (m *GormRepository) Create(ctx context.Context, category *Category) (*Category, error) {
	if strings.TrimSpace(category.Name) == "" {
		return nil, errors.NewBadParameterError("Name", category.Name)

	}
	db := m.db.Create(category)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "categories_name_idx") {
			return nil, errors.NewBadParameterError("Name", category.Name).Expected("unique")
		}
		log.Error(ctx, map[string]interface{}{
			"category_id": category.ID,
			"err":         db.Error.Error(),
		}, "unable to create category")
		return nil, errors.NewInternalError(errs.Wrap(db.Error, "unable to create category"))
	}
	log.Info(ctx, map[string]interface{}{
		"category_id": category.ID,
	}, "category created successfully")
	return category, nil
}

// LoadAllRelationshipsOfCategory loads all the relationships of a category. This is required for workitemtype filtering.
func (m *GormRepository) LoadAllRelationshipsOfCategory(ctx context.Context, categoryID uuid.UUID) ([]*WorkItemTypeCategoryRelationship, error) {
	// Check if category is present
	_, err := m.LoadCategory(ctx, categoryID)
	if err != nil {
		return nil, errs.Wrap(err, fmt.Sprintf("failed to load category with id %s", categoryID))
	}

	relationship := []*WorkItemTypeCategoryRelationship{}
	db := m.db.Model(&relationship).Where("category_id=?", categoryID).Find(&relationship)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"category_id": categoryID,
		}, "workitemtype category relationship not found")
		return nil, errors.NewNotFoundError("work item type category", categoryID.String())
	}
	if err := db.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": categoryID,
			"err":         err,
		}, "unable to list workitemtype category relationships")
		return nil, errors.NewInternalError(errs.Wrap(db.Error, "unable to list workitemtype category relationships"))
	}
	return relationship, nil
}

// LoadCategory returns category for the given id
// This is needed to check if a category is present in db or not.
func (m *GormRepository) LoadCategory(ctx context.Context, id uuid.UUID) (*Category, error) {
	res := Category{}
	db := m.db.Model(&res).Where("id=?", id).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"category_id": id,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", id.String())
	}
	if err := db.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": id,
			"err":         err,
		}, "unable to load category", err.Error())
		return nil, errors.NewInternalError(errs.Wrap(db.Error, "unable to load category"))
	}
	return &res, nil
}

// Save updates a category in the database based on the ID by the given category object
func (m *GormRepository) Save(ctx context.Context, category *Category) (*Category, error) {
	res := Category{}
	log.Info(ctx, map[string]interface{}{
		"category_id": category.ID,
	}, "Looking for category")
	tx := m.db.Model(&res).Where("id=?", category.ID).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"category_id": category.ID,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", category.ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": category.ID,
			"err":         tx.Error,
		}, "unable to load category")
		return nil, errors.NewInternalError(errs.Wrap(tx.Error, "unable to save category"))
	}

	res.Name = category.Name
	tx = tx.Save(&res)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": category.ID,
			"err":         err,
		}, "unable to save category")
		return nil, errors.NewInternalError(errs.Wrap(tx.Error, "unable to save category"))
	}
	log.Info(ctx, map[string]interface{}{
		"category_id": category.ID,
	}, "Updated category")
	return &res, nil
}
