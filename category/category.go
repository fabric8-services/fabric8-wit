package category

import (
	"context"
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
	PlannerIssues       = "planner.issues"
)

// Do not change these UUIDs!!!
// System defined categories
var (
	PlannerRequirementsID = uuid.FromStringOrNil("04aef834-1505-44cf-80e4-ab0d857d9f56") // "planner.requirements"
	PlannerIssuesID       = uuid.FromStringOrNil("27d92fe4-b2ee-45c2-b9bb-01f355ad616f") // "planner.issues"
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
	WorkitemtypeID uuid.UUID `sql:"type:uuid"`
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *WorkItemTypeCategoryRelationship) TableName() string {
	return "workitemtype_categories"
}

// Ensure fields implements the Equaler interface
var _ convert.Equaler = Category{}
var _ convert.Equaler = (*Category)(nil)

// Equal returns true if two Category objects are equal; otherwise false is returned.
func (category Category) Equal(u convert.Equaler) bool {
	other, ok := u.(Category)
	if !ok || !uuid.Equal(category.ID, other.ID) || !category.Lifecycle.Equal(other.Lifecycle) || category.Name != other.Name {
		return false
	}
	return true
}

// Repository encapsulates storage and retrieval of categories
type Repository interface {
	Create(ctx context.Context, category *Category) (*Category, error)
	LoadCategoryFromDB(ctx context.Context, id uuid.UUID) (*Category, error)
	List(ctx context.Context) ([]*Category, error)
	CreateRelationship(ctx context.Context, relationship *WorkItemTypeCategoryRelationship) error
	LoadWorkItemTypeCategoryRelationship(ctx context.Context, workitemtypeID uuid.UUID, categoryID uuid.UUID) (*WorkItemTypeCategoryRelationship, error)
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
		return nil, errs.WithStack(err)
	}
	return objs, nil
}

// CreateRelationship creates relationship between workitemtype and category
func (m *GormRepository) CreateRelationship(ctx context.Context, relationship *WorkItemTypeCategoryRelationship) error {
	db := m.db.Create(relationship)
	if db.Error != nil {
		if gormsupport.IsUniqueViolation(db.Error, "workitemtype_categories_idx") {
			return errors.NewBadParameterError("category+workitemtype", relationship.CategoryID).Expected("unique")
		}
		return errors.NewInternalError(db.Error.Error())
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
		return nil, errors.NewInternalError(db.Error.Error())
	}
	log.Info(ctx, map[string]interface{}{
		"category_id": category.ID,
	}, "Category created successfully")
	return category, nil
}

// LoadAllRelationshipsOfCategory loads all the relationships of a category. This is required for workitemtype filtering.
func (m *GormRepository) LoadAllRelationshipsOfCategory(ctx context.Context, categoryID uuid.UUID) ([]*WorkItemTypeCategoryRelationship, error) {
	// Check if category is present
	getCategory := Category{}
	db := m.db.Model(&getCategory).Where("id=?", categoryID).Find(&getCategory)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	relationship := []*WorkItemTypeCategoryRelationship{}
	db = m.db.Model(&relationship).Where("category_id=?", categoryID).Find(&relationship)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "workitemtypes of category not found")
		return nil, errors.NewNotFoundError("work item type category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return relationship, nil
}

// LoadCategoryFromDB returns category for the given id
// This is needed to check if a category is present in db or not.
func (m *GormRepository) LoadCategoryFromDB(ctx context.Context, id uuid.UUID) (*Category, error) {
	log.Logger().Infoln("Loading category", id)
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
			"categoryID": id,
		}, "category retrieval error", err.Error())
		return nil, errors.NewInternalError(err.Error())
	}
	return &res, nil
}

// LoadWorkItemTypeCategoryRelationship loads all the relationships of a category. This is required for testing.
func (m *GormRepository) LoadWorkItemTypeCategoryRelationship(ctx context.Context, workitemtypeID uuid.UUID, categoryID uuid.UUID) (*WorkItemTypeCategoryRelationship, error) {
	// Check if category is present
	getCategory := Category{}
	db := m.db.Model(&getCategory).Where("id=?", categoryID).Find(&getCategory)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	relationship := WorkItemTypeCategoryRelationship{}
	db = m.db.Model(&relationship).Where("category_id=? AND workitemtype_id=?", categoryID, workitemtypeID).Find(&relationship)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "workitemtypes of category not found")
		return nil, errors.NewNotFoundError("work item type category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(db.Error.Error())
	}
	return &relationship, nil
}

// Save saves a category object. This function is used to update category properly during migration -> createOrUpdateSingleCategory()
func (m *GormRepository) Save(ctx context.Context, category *Category) (*Category, error) {
	res := Category{}
	log.Info(ctx, map[string]interface{}{
		"id": category.ID,
	}, "Looking for category")
	tx := m.db.Model(&res).Where("id=?", category.ID).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"id": category.ID,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", category.ID.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(tx.Error.Error())
	}

	res.Name = category.Name
	tx = tx.Save(&res)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"id":  category.ID,
			"err": err,
		}, "unable to save category")
		return nil, errors.NewInternalError(err.Error())
	}
	log.Info(ctx, map[string]interface{}{
		"id": category.ID,
	}, "Updated category")
	return &res, nil
}
