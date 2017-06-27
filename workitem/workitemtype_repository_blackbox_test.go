package workitem_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/almighty/almighty-core/category"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	clean        func()
	repo         workitem.WorkItemTypeRepository
	ctx          context.Context
	categoryRepo category.Repository
}

func TestRunWorkItemTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemTypeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	s.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (s *workItemTypeRepoBlackBoxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
	s.categoryRepo = category.NewRepository(s.DB)
	workitem.ClearGlobalWorkItemTypeCache()
}

func (s *workItemTypeRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWIT() {
	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID)

	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	// Test that we can create a WIT with the same name as before.
	wit3, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit3)
	require.NotNil(s.T(), wit3.ID)

	wit2, err := s.repo.Load(s.ctx, space.SystemSpace, wit.ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Fields)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), workitem.KindFloat, field.Type.GetKind())
	assert.Equal(s.T(), true, field.Required)
}

func (s *workItemTypeRepoBlackBoxTest) TestExistsWIT() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("wit exists", func(t *testing.T) {
		t.Parallel()
		// given
		wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
			"foo": {
				Required: true,
				Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
			},
		})
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wit)
		require.NotNil(s.T(), wit.ID)

		exists, err := s.repo.Exists(s.ctx, wit.ID.String())
		require.Nil(s.T(), err)
		require.True(s.T(), exists)
	})

	t.Run("wit doesn't exist", func(t *testing.T) {
		t.Parallel()
		exists, err := s.repo.Exists(s.ctx, uuid.NewV4().String())
		require.False(t, exists)
		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWITWithList() {
	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID)
	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	wit3, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit3)
	require.NotNil(s.T(), wit3.ID)

	wit2, err := s.repo.Load(s.ctx, space.SystemSpace, wit.ID)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Fields)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), workitem.KindList, field.Type.GetKind())
	assert.Equal(s.T(), true, field.Required)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateWITWithBaseType() {
	basetype := "foo.bar"
	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID)
	baseWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, basetype, nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
		},
	})

	require.Nil(s.T(), err)
	require.NotNil(s.T(), baseWit)
	require.NotNil(s.T(), baseWit.ID)
	extendedWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, &baseWit.ID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), extendedWit)
	require.NotNil(s.T(), extendedWit.Fields)
	// the Field 'foo' must exist since it is inherited from the base work item type
	assert.NotNil(s.T(), extendedWit.Fields["foo"])
}

func (s *workItemTypeRepoBlackBoxTest) TestDoNotCreateWITWithMissingBaseType() {
	baseTypeID := uuid.Nil
	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID)
	extendedWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, &baseTypeID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	// expect an error as the given base type does not exist
	require.NotNil(s.T(), err)
	require.Nil(s.T(), extendedWit)
}

// LoadWorkItemTypeCategoryRelationship loads all the relationships of a category. This is required for testing.
func (s *workItemTypeRepoBlackBoxTest) loadWorkItemTypeCategoryRelationship(ctx context.Context, workitemtypeID uuid.UUID, categoryID uuid.UUID) (*category.WorkItemTypeCategoryRelationship, error) {
	// Check if category is present
	_, err := s.categoryRepo.LoadCategory(ctx, categoryID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": categoryID,
		}, "category not found")
		return nil, errs.Wrap(err, fmt.Sprintf("failed to load category with id %s", categoryID))
	}
	relationship := category.WorkItemTypeCategoryRelationship{}
	db := s.DB.Model(&relationship).Where("category_id=? AND work_item_type_id=?", categoryID, workitemtypeID).Find(&relationship)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"category_id": categoryID,
			"wit_id":      workitemtypeID,
		}, "workitemtype category relationship not found")
		return nil, errors.NewNotFoundError("work item type category", categoryID.String())
	}
	if err := db.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"category_id": categoryID,
			"wit_id":      workitemtypeID,
			"err":         err,
		}, "unable to load workitemtype category relationship")
		return nil, errors.NewInternalError(ctx, errs.Wrap(db.Error, "unable to load workitemtype category relationship"))
	}
	return &relationship, nil
}

//----------------------------------------------------------------------------------------------
// Tests on WorkItemType and Category Relationship
//----------------------------------------------------------------------------------------------

// TestSingleWorkItemTypeToSingleCategoryRelationship tests creation of relationship between a single workitemtype and a single category i.e. one-to-one relationship
func (s *workItemTypeRepoBlackBoxTest) TestSingleWorkItemTypeToSingleCategoryRelationship() {

	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID) // single category
	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create work item type
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit.ID, categoryID) // associate workitemtype with category
	require.Nil(s.T(), err)
	relationship, err := s.loadWorkItemTypeCategoryRelationship(s.ctx, wit.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)
}

// TestSingleWorkItemTypeToMultipleCategoryRelationship tests creation of relationship between a single workitemtype and multiple categories i.e. one-to-many relationship
func (s *workItemTypeRepoBlackBoxTest) TestSingleWorkItemTypeToMultipleCategoryRelationship() {

	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID, &category.PlannerPortfolioID) // multiple categories
	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create workitemtype
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit.ID, categoryID) // associate workitemtype with categories
	require.Nil(s.T(), err)

	relationship, err := s.loadWorkItemTypeCategoryRelationship(s.ctx, wit.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)

	relationship, err = s.loadWorkItemTypeCategoryRelationship(s.ctx, wit.ID, *categoryID[1])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[1], relationship.CategoryID)
}

// TestMultipleWorkItemTypeToSingleCategoryRelationship tests creation of relationship between multiple workitemtypes and a single category i.e. many-to-one relationship
func (s *workItemTypeRepoBlackBoxTest) TestMultipleWorkItemTypeToSingleCategoryRelationship() {

	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID) // single category
	wit1, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create workitemtype
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit1)
	require.NotNil(s.T(), wit1.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit1.ID, categoryID) // associate workitemtype with category
	require.Nil(s.T(), err)

	relationship, err := s.loadWorkItemTypeCategoryRelationship(s.ctx, wit1.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit1.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)

	wit2, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create workitemtype
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit2.ID, categoryID) // associate workitemtype with category
	require.Nil(s.T(), err)
	relationship, err = s.loadWorkItemTypeCategoryRelationship(s.ctx, wit2.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit2.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)
}

// TestMultipleWorkItemTypeToMultipleCategoryRelationship tests creation of relationship between multiple workitemtypes and multiple categories i.e. many-to-many relationship
func (s *workItemTypeRepoBlackBoxTest) TestMultipleWorkItemTypeToMultipleCategoryRelationship() {

	categoryID := []*uuid.UUID{}
	categoryID = append(categoryID, &category.PlannerRequirementsID, &category.PlannerPortfolioID) // multiple categories
	wit1, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create workitemtype
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit1)
	require.NotNil(s.T(), wit1.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit1.ID, categoryID) // associate workitemtype with categories
	require.Nil(s.T(), err)
	relationship, err := s.loadWorkItemTypeCategoryRelationship(s.ctx, wit1.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit1.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)

	relationship, err = s.loadWorkItemTypeCategoryRelationship(s.ctx, wit1.ID, *categoryID[1])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit1.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[1], relationship.CategoryID)

	wit2, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	}) // create workitemtype
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.ID)

	err = s.repo.AssociateWithCategories(s.ctx, wit2.ID, categoryID) // associate workitemtype with categories
	require.Nil(s.T(), err)
	relationship, err = s.loadWorkItemTypeCategoryRelationship(s.ctx, wit2.ID, *categoryID[0])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit2.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[0], relationship.CategoryID)

	relationship, err = s.loadWorkItemTypeCategoryRelationship(s.ctx, wit2.ID, *categoryID[1])
	require.Nil(s.T(), err)
	require.NotNil(s.T(), relationship)
	require.Equal(s.T(), wit2.ID, relationship.WorkItemTypeID)
	require.Equal(s.T(), *categoryID[1], relationship.CategoryID)
}
