package category

import (
	"context"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// Defines "type" string to be used while validating jsonapi spec based payload
const (
	APIStringTypeCategory = "categories"
)

const (
	PlannerRequirements = "planner.requirements"
	PlannerIssues       = "planner.issues"
)

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
func (m *Category) TableName() string {
	return "categories"
}

// Category_wit_relationship describes relationship between a category and a workitemtype.
type CategoryWitRelationship struct {
	gormsupport.Lifecycle
	ID             uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	CategoryID     uuid.UUID `sql:"type:uuid"`
	WorkitemtypeID uuid.UUID `sql:"type:uuid"`
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *CategoryWitRelationship) TableName() string {
	return "workitemtype_categories"
}

type CategoryRepository interface {
	Create(ctx context.Context, category *Category) (*Category, error)
	List(ctx context.Context) ([]*Category, error)
	CreateRelationship(ctx context.Context, relationship *CategoryWitRelationship) error
	LoadRelationships(ctx context.Context, categoryID uuid.UUID) ([]*CategoryWitRelationship, error)
	LoadCategoryFromDB(ctx context.Context, id uuid.UUID) (*Category, error)
}

// NewCategoryRepository creates a new storage type.
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &GormCategoryRepository{db: db}
}

// GormCategoryRepository is the implementation of the storage interface for Categories.
type GormCategoryRepository struct {
	db *gorm.DB
}

// List all Categories.
func (m *GormCategoryRepository) List(ctx context.Context) ([]*Category, error) {
	var objs []*Category
	err := m.db.Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return objs, nil
}

// CreateRelationship creates relationship between workitemtype and category
func (r *GormCategoryRepository) CreateRelationship(ctx context.Context, relationship *CategoryWitRelationship) error {
	if relationship.ID == uuid.Nil {
		relationship.ID = uuid.NewV4()
	}
	db := r.db.Create(relationship)
	if db.Error != nil {
		return errors.NewInternalError(db.Error.Error())
	}
	return nil
}

// Create creates category. This function is used to populate categories table through migration -> PopulateCategories()
func (r *GormCategoryRepository) Create(ctx context.Context, category *Category) (*Category, error) {
	if category.ID == uuid.Nil {
		category.ID = uuid.NewV4()
	}
	db := r.db.Create(category)
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

// LoadRelationships loads the relationships. This is required for workitemtype filtering.
func (r *GormCategoryRepository) LoadRelationships(ctx context.Context, categoryID uuid.UUID) ([]*CategoryWitRelationship, error) {

	// Check if category is present
	getCategory := Category{}
	db := r.db.Model(&getCategory).Where("id=?", categoryID).Find(&getCategory)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "category not found")
		return nil, errors.NewNotFoundError("category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	relationship := []*CategoryWitRelationship{}
	db = r.db.Model(&relationship).Where("category_id=?", categoryID).Find(&relationship)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"categories_id": categoryID,
		}, "workitemtypes of category not found")
		return nil, errors.NewNotFoundError("work item type category", categoryID.String())
	}
	if err := db.Error; err != nil {
		return nil, errors.NewInternalError(err.Error())
	}
	return relationship, nil
}

// LoadCategoryFromDB returns category for the given id
// This is needed to check if a particular category is present in db or not before creating.
func (r *GormCategoryRepository) LoadCategoryFromDB(ctx context.Context, id uuid.UUID) (*Category, error) {
	log.Logger().Infoln("Loading category", id)
	res := Category{}
	db := r.db.Model(&res).Where("id=?", id).First(&res)
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
