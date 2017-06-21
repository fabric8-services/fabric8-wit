package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemTypeSuite struct {
	gormtestsupport.DBTestSuite
	clean        func()
	typeCtrl     *WorkitemtypeController
	linkTypeCtrl *WorkItemLinkTypeController
	linkCatCtrl  *WorkItemLinkCategoryController
	spaceCtrl    *SpaceController
	svc          *goa.Service
	testDir      string
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemType(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemTypeSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
	s.testDir = filepath.Join("test-files", "work_item_type")
}

// The SetupTest method will be run before every test in the suite.
func (s *workItemTypeSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)
	s.typeCtrl = NewWorkitemtypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
}

func (s *workItemTypeSuite) TearDownTest() {
	s.clean()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

var (
	animalID = uuid.FromStringOrNil("729431f2-bca4-4062-9087-c751807b569f")
	personID = uuid.FromStringOrNil("22a1e4f1-7e9d-4ce8-ac87-fe7c79356b16")
)

// createWorkItemTypeAnimal defines a work item type "animal" that consists of
// two fields ("animal-type" and "color"). The type is mandatory but the color is not.
func (s *workItemTypeSuite) createWorkItemTypeAnimal() (http.ResponseWriter, *app.WorkItemTypeSingle) {
	return s.createWorkItemTypeAnimalWithDates(nil, nil)
}

func (s *workItemTypeSuite) createWorkItemTypeAnimalWithDates(createdAt, updatedAt *time.Time) (http.ResponseWriter, *app.WorkItemTypeSingle) {
	// Create an enumeration of animal names
	typeStrings := []string{"elephant", "blue whale", "Tyrannosaurus rex"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	stString := "string"

	// Use the goa generated code to create a work item type
	desc := "Description for 'animal'"
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			Type: "workitemtypes",
			ID:   &animalID,
			Attributes: &app.WorkItemTypeAttributes{
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Name:        "animal",
				Description: &desc,
				Icon:        "fa-hand-lizard-o",
				Fields: map[string]*app.FieldDefinition{
					"animal_type": {
						Required: true,
						Type: &app.FieldType{
							BaseType: &stString,
							Kind:     "enum",
							Values:   typeEnum,
						},
					},
					"color": {
						Required: false,
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	s.T().Log("Creating 'animal' work item type...")
	responseWriter, wi := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &payload)
	require.NotNil(s.T(), wi)
	s.T().Log("'animal' work item type created.")
	return responseWriter, wi
}

func (s *workItemTypeSuite) createWorkItemTypePerson() (http.ResponseWriter, *app.WorkItemTypeSingle) {
	return s.createWorkItemTypePersonWithDates(nil, nil)
}

// createWorkItemTypePerson defines a work item type "person" that consists of
// a required "name" field.
func (s *workItemTypeSuite) createWorkItemTypePersonWithDates(createdAt, updatedAt *time.Time) (http.ResponseWriter, *app.WorkItemTypeSingle) {
	// Use the goa generated code to create a work item type
	desc := "Description for 'person'"
	id := personID
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Name:        "person",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"name": {
						Required: true,
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	responseWriter, wi := test.CreateWorkitemtypeCreated(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &payload)
	require.NotNil(s.T(), wi)
	return responseWriter, wi
}

func newCreateWorkItemTypePayload(id uuid.UUID, spaceID uuid.UUID) app.CreateWorkitemtypePayload {
	// Use the goa generated code to create a work item type
	desc := "Description for 'person'"
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(spaceID.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "person",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"test": {
						Required: false,
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: app.NewSpaceRelation(spaceID, spaceSelfURL),
			},
		},
	}

	return payload
}

func lookupWorkItemTypes(witCollection app.WorkItemTypeList, workItemTypes ...app.WorkItemTypeSingle) assert.Comparison {
	return func() bool {
		if len(witCollection.Data) < 2 {
			return false
		}
		toBeFound := len(workItemTypes)
		for i := 0; i < len(witCollection.Data) && toBeFound > 0; i++ {
			id := *witCollection.Data[i].ID
			for _, workItemType := range workItemTypes {
				if uuid.Equal(id, *workItemType.Data.ID) {
					toBeFound--
					break
				}
			}
		}
		return toBeFound == 0
	}
}

//-----------------------------------------------------------------------------
// Test on work item types retrieval (single and list)
//-----------------------------------------------------------------------------

func (s *workItemTypeSuite) TestCreate() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	s.T().Run("ok", func(t *testing.T) {
		res, animal := s.createWorkItemTypeAnimal()
		compareWithGolden(s.T(), filepath.Join(s.testDir, "create", "animal.wit.golden.json"), animal)
		compareWithGolden(s.T(), filepath.Join(s.testDir, "create", "animal.headers.golden.json"), res.Header())
		res, person := s.createWorkItemTypePerson()
		compareWithGolden(s.T(), filepath.Join(s.testDir, "create", "person.golden.json"), person)
		compareWithGolden(s.T(), filepath.Join(s.testDir, "create", "person.headers.golden.json"), res.Header())
	})
}

func (s *workItemTypeSuite) TestValidate() {
	// given
	desc := "Description for 'person'"
	id := personID
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload := app.CreateWorkitemtypePayload{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "",
				Description: &desc,
				Icon:        "fa-user",
				Fields: map[string]*app.FieldDefinition{
					"name": {
						Required: true,
						Type: &app.FieldType{
							Kind: "string",
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}

	s.T().Run("valid", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = "Valid Name 0baa42b5-fa52-4ee2-847d-ef26b23fbb6e"
		// when
		err := p.Validate()
		// then
		require.Nil(t, err)
	})

	s.T().Run("invalid - oversized name", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = testsupport.TestOversizedNameObj
		// when
		err := p.Validate()
		// then
		require.NotNil(t, err)
		gerr, ok := err.(*goa.ErrorResponse)
		require.True(t, ok)
		gerr.ID = "IGNORE_ME"
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "invalid_oversized_name.golden.json"), gerr)
	})

	s.T().Run("invalid - name starts with underscore", func(t *testing.T) {
		// given
		p := payload
		p.Data.Attributes.Name = "_person"
		// when
		err := p.Validate()
		// then
		require.NotNil(t, err)
		gerr, ok := err.(*goa.ErrorResponse)
		require.True(t, ok)
		gerr.ID = "IGNORE_ME"
		compareWithGolden(t, filepath.Join(s.testDir, "validate", "invalid_name_starts_with_underscore.golden.json"), gerr)
	})
}

// TestShowWorkItemType200OK tests if we can fetch the work item type "animal".
func (s *workItemTypeSuite) TestShow() {
	// Disable gorm's automatic setting of "created_at" and "updated_at"
	s.DB.Callback().Create().Remove("gorm:update_time_stamp")
	s.DB.Callback().Update().Remove("gorm:update_time_stamp")

	// given
	date := time.Date(2017, 05, 01, 0, 0, 0, 0, time.UTC)
	_, wit := s.createWorkItemTypeAnimalWithDates(&date, &date)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.Data)
	require.NotNil(s.T(), wit.Data.ID)

	s.T().Run("ok", func(t *testing.T) {
		// when
		res, actual := test.ShowWorkitemtypeOK(t, nil, nil, s.typeCtrl, *wit.Data.Relationships.Space.Data.ID, *wit.Data.ID, nil, nil)
		// then
		require.NotNil(t, actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok.wit.golden.json"), actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok.headers.golden.json"), res.Header())
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(wit.Data.Attributes.CreatedAt.Add(-1 * time.Hour))
		res, actual := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, *wit.Data.ID, &lastModified, nil)
		// then
		require.NotNil(t, actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.wit.golden.json"), actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok_using_expired_lastmodified_header.headers.golden.json"), res.Header())
	})

	s.T().Run("TestShowWorkItemType200UsingExpiredETagHeader", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, actual := test.ShowWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, *wit.Data.Relationships.Space.Data.ID, *wit.Data.ID, nil, &ifNoneMatch)
		// then
		require.NotNil(t, actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.wit.golden.json"), actual)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "ok_using_expired_etag_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when
		lastModified := app.ToHTTPTime(wit.Data.Attributes.UpdatedAt.Add(1 * time.Second))
		res := test.ShowWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, *wit.Data.Relationships.Space.Data.ID, *wit.Data.ID, &lastModified, nil)
		// then
		compareWithGolden(t, filepath.Join(s.testDir, "show", "not_modified_using_if_modified_since_header.headers.golden.json"), res.Header())
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// when
		etag := generateWorkItemTypeTag(*wit)
		res := test.ShowWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, *wit.Data.Relationships.Space.Data.ID, *wit.Data.ID, nil, &etag)
		// then
		compareWithGolden(t, filepath.Join(s.testDir, "show", "not_modified_using_ifnonematch_header.headers.golden.json"), res.Header())
	})
}

func (s *workItemTypeSuite) TestList() {
	// given
	_, witAnimal := s.createWorkItemTypeAnimalWithDates(nil, nil)
	require.NotNil(s.T(), witAnimal)
	_, witPerson := s.createWorkItemTypePersonWithDates(nil, nil)
	require.NotNil(s.T(), witPerson)

	s.T().Run("ok", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, nil, nil)
		// then
		require.NotNil(s.T(), witCollection)
		require.Nil(s.T(), witCollection.Validate())
		s.T().Log("Response headers:", res.Header())
		assert.Condition(s.T(), lookupWorkItemTypes(*witCollection, *witAnimal, *witPerson),
			"Not all required work item types (animal and person) where found.")
		require.NotNil(s.T(), res.Header()[app.LastModified])
		//assert.Equal(s.T(), app.ToHTTPTime(getWorkItemTypeUpdatedAt(*witPerson)), res.Header()[app.LastModified][0])
		require.NotNil(s.T(), res.Header()[app.CacheControl])
		assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
		require.NotNil(s.T(), res.Header()[app.ETag])
		assert.Equal(s.T(), generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using expired IfModifiedSince header", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		lastModified := app.ToHTTPTime(time.Now().Add(-1 * time.Hour))
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, &lastModified, nil)
		// then
		require.NotNil(s.T(), witCollection)
		require.Nil(s.T(), witCollection.Validate())
		assert.Condition(s.T(), lookupWorkItemTypes(*witCollection, *witAnimal, *witPerson),
			"Not all required work item types (animal and person) where found.")
		require.NotNil(s.T(), res.Header()[app.LastModified])
		//assert.Equal(s.T(), app.ToHTTPTime(getWorkItemTypeUpdatedAt(*witPerson)), res.Header()[app.LastModified][0])
		require.NotNil(s.T(), res.Header()[app.CacheControl])
		assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
		require.NotNil(s.T(), res.Header()[app.ETag])
		assert.Equal(s.T(), generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("ok - using IfNoneMatch header", func(t *testing.T) {
		// when
		// Paging in the format <start>,<limit>"
		etag := "foo"
		page := "0,-1"
		res, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, nil, &etag)
		// then
		require.NotNil(s.T(), witCollection)
		require.Nil(s.T(), witCollection.Validate())
		assert.Condition(s.T(), lookupWorkItemTypes(*witCollection, *witAnimal, *witPerson),
			"Not all required work item types (animal and person) where found.")
		require.NotNil(s.T(), res.Header()[app.LastModified])
		//assert.Equal(s.T(), app.ToHTTPTime(getWorkItemTypeUpdatedAt(*witPerson)), res.Header()[app.LastModified][0])
		require.NotNil(s.T(), res.Header()[app.CacheControl])
		assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
		require.NotNil(s.T(), res.Header()[app.ETag])
		assert.Equal(s.T(), generateWorkItemTypesTag(*witCollection), res.Header()[app.ETag][0])
	})

	s.T().Run("not modified - using IfModifiedSince header", func(t *testing.T) {
		// when/then
		// Paging in the format <start>,<limit>"
		//lastModified := app.ToHTTPTime(getWorkItemTypeUpdatedAt(*witPerson))
		lastModified := app.ToHTTPTime(time.Now())
		page := "0,-1"
		test.ListWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, &lastModified, nil)
	})

	s.T().Run("not modified - using IfNoneMatch header", func(t *testing.T) {
		// given
		// Paging in the format <start>,<limit>"
		page := "0,-1"
		_, witCollection := test.ListWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, nil, nil)
		require.NotNil(s.T(), witCollection)
		// when/then
		ifNoneMatch := generateWorkItemTypesTag(*witCollection)
		test.ListWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, &page, nil, &ifNoneMatch)
	})
}

//-----------------------------------------------------------------------------
// Test on work item type links retrieval
//-----------------------------------------------------------------------------

const (
	animalLinksToBugStr = "animal-links-to-bug"
	bugLinksToAnimalStr = "bug-links-to-animal"
)

func (s *workItemTypeSuite) createWorkitemtypeLinks() (app.WorkItemLinkTypeSingle, app.WorkItemLinkTypeSingle) {
	// Create the work item type first and try to read it back in
	_, witAnimal := s.createWorkItemTypeAnimal()
	require.NotNil(s.T(), witAnimal)
	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)
	s.T().Log("Created work items")
	// Create work item link category
	linkCatPayload := newCreateWorkItemLinkCategoryPayload("some-link-category-" + uuid.NewV4().String())
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)
	s.T().Log("Created work item link category")
	// Create work item link space
	spacePayload := CreateSpacePayload("some-link-space-"+uuid.NewV4().String(), "description")
	_, sp := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, spacePayload)
	s.T().Log("Created space")
	// Create work item link type
	linkTypePayload := newCreateWorkItemLinkTypePayload(animalLinksToBugStr, animalID, workitem.SystemBug, *linkCat.Data.ID, *sp.Data.ID)
	_, sourceLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *sp.Data.ID, linkTypePayload)
	require.NotNil(s.T(), sourceLinkType)
	s.T().Log("Created work item source link")
	// Create another work item link type
	linkTypePayload = newCreateWorkItemLinkTypePayload(bugLinksToAnimalStr, workitem.SystemBug, animalID, *linkCat.Data.ID, *sp.Data.ID)
	_, targetLinkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *sp.Data.ID, linkTypePayload)
	require.NotNil(s.T(), targetLinkType)
	s.T().Log("Created work item target link")
	return *sourceLinkType, *targetLinkType
}

// TestListWorkItemLinkTypeSources200OK tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListWorkItemLinkTypeSources200OK() {
	// given
	sourceLinkType, _ := s.createWorkitemtypeLinks()
	// when fetch source link types
	res, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, nil)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), animalLinksToBugStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(sourceLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(sourceLinkType), res.Header()[app.ETag][0])
}

// TestListWorkItemLinkTypeTargets200OK tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListWorkItemLinkTypeTargets200OK() {
	// given
	_, targetLinkType := s.createWorkitemtypeLinks()
	// When fetch target link types
	res, wiltCollection := test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, nil)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// Then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), bugLinksToAnimalStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(targetLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(targetLinkType), res.Header()[app.ETag][0])
}

// TestListSourceLinkTypes200UsingExpiredIfModifiedSinceHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceLinkTypes200UsingExpiredIfModifiedSinceHeader() {
	// given
	sourceLinkType, _ := s.createWorkitemtypeLinks()
	// when fetch source link types
	ifModifiedSince := app.ToHTTPTime(sourceLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, &ifModifiedSince, nil)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), animalLinksToBugStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(sourceLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(sourceLinkType), res.Header()[app.ETag][0])
}

// TestListTargetLinkTypes200UsingExpiredIfModifiedSinceHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListTargetLinkTypes200UsingExpiredIfModifiedSinceHeader() {
	// given
	_, targetLinkType := s.createWorkitemtypeLinks()
	// When fetch target link types
	ifModifiedSince := app.ToHTTPTime(targetLinkType.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	res, wiltCollection := test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, &ifModifiedSince, nil)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// Then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), bugLinksToAnimalStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(targetLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(targetLinkType), res.Header()[app.ETag][0])
}

// TestListSourceLinkTypes200UsingExpiredIfNoneMatchHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceLinkTypes200UsingExpiredIfNoneMatchHeader() {
	// given
	sourceLinkType, _ := s.createWorkitemtypeLinks()
	// when fetch source link types
	ifNoneMatch := "foo"
	res, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, &ifNoneMatch)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), animalLinksToBugStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(sourceLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(sourceLinkType), res.Header()[app.ETag][0])
}

// TestListTargetLinkTypes200UsingExpiredIfNoneMatchHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListTargetLinkTypes200UsingExpiredIfNoneMatchHeader() {
	// given
	_, targetLinkType := s.createWorkitemtypeLinks()
	// When fetch target link types
	ifNoneMatch := "foo"
	res, wiltCollection := test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, &ifNoneMatch)
	require.NotNil(s.T(), wiltCollection)
	assert.Nil(s.T(), wiltCollection.Validate())
	// Then check the number of found work item link types
	require.Len(s.T(), wiltCollection.Data, 1)
	assert.Equal(s.T(), bugLinksToAnimalStr, *wiltCollection.Data[0].Attributes.Name)
	require.NotNil(s.T(), res.Header()[app.LastModified])
	assert.Equal(s.T(), app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(targetLinkType)), res.Header()[app.LastModified][0])
	require.NotNil(s.T(), res.Header()[app.CacheControl])
	assert.NotNil(s.T(), res.Header()[app.CacheControl][0])
	require.NotNil(s.T(), res.Header()[app.ETag])
	assert.Equal(s.T(), generateWorkItemLinkTypeTag(targetLinkType), res.Header()[app.ETag][0])
}

// TestListSourceLinkTypes304UsingIfModifiedSinceHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceLinkTypes304UsingIfModifiedSinceHeader() {
	// given
	sourceLinkType, _ := s.createWorkitemtypeLinks()
	// when/then
	ifModifiedSince := app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(sourceLinkType))
	test.ListSourceLinkTypesWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, &ifModifiedSince, nil)
}

// TestListTargetLinkTypes200UsingExpiredIfModifiedSinceHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListTargetLinkTypes304UsingIfModifiedSinceHeader() {
	// given
	_, targetLinkType := s.createWorkitemtypeLinks()
	// When fetch target link types
	ifModifiedSince := app.ToHTTPTime(getWorkItemLinkTypeUpdatedAt(targetLinkType))
	test.ListTargetLinkTypesWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, &ifModifiedSince, nil)
}

// TestListSourceLinkTypes200UsingExpiredIfNoneMatchHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListSourceLinkTypes304UsingIfNoneMatchHeader() {
	// given
	sourceLinkType, _ := s.createWorkitemtypeLinks()
	// when fetch source link types
	ifNoneMatch := generateWorkItemLinkTypeTag(sourceLinkType)
	test.ListSourceLinkTypesWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, &ifNoneMatch)
}

// TestListTargetLinkTypes304UsingIfNoneMatchHeader tests if we can find the work item link
// types for a given WIT.
func (s *workItemTypeSuite) TestListTargetLinkTypes304UsingIfNoneMatchHeader() {
	// given
	_, targetLinkType := s.createWorkitemtypeLinks()
	// When fetch target link types
	ifNoneMatch := generateWorkItemLinkTypeTag(targetLinkType)
	test.ListTargetLinkTypesWorkitemtypeNotModified(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, animalID, nil, &ifNoneMatch)
}

// TestListSourceAndTargetLinkTypesEmpty tests that no link type is returned for
// WITs that don't have link types associated to them
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesEmpty() {
	_, witPerson := s.createWorkItemTypePerson()
	require.NotNil(s.T(), witPerson)

	_, wiltCollection := test.ListSourceLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, personID, nil, nil)
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)

	_, wiltCollection = test.ListTargetLinkTypesWorkitemtypeOK(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, personID, nil, nil)
	require.NotNil(s.T(), wiltCollection)
	require.Nil(s.T(), wiltCollection.Validate())
	require.Len(s.T(), wiltCollection.Data, 0)
}

// TestListSourceAndTargetLinkTypesNotFound tests that a NotFound error is
// returned when you query a non existing WIT.
func (s *workItemTypeSuite) TestListSourceAndTargetLinkTypesNotFound() {
	_, jerrors := test.ListSourceLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, uuid.Nil, nil, nil)
	require.NotNil(s.T(), jerrors)

	_, jerrors = test.ListTargetLinkTypesWorkitemtypeNotFound(s.T(), nil, nil, s.typeCtrl, space.SystemSpace, uuid.Nil, nil, nil)
	require.NotNil(s.T(), jerrors)
}

// used for testing purpose only
func convertWorkItemTypeToModel(data app.WorkItemTypeData) workitem.WorkItemType {
	return workitem.WorkItemType{
		ID:      *data.ID,
		Version: *data.Attributes.Version,
	}
}

func generateWorkItemTypesTag(entities app.WorkItemTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		modelEntities[i] = convertWorkItemTypeToModel(*entityData)
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemTypeTag(entity app.WorkItemTypeSingle) string {
	return app.GenerateEntityTag(convertWorkItemTypeToModel(*entity.Data))
}

func generateWorkItemLinkTypesTag(entities app.WorkItemLinkTypeList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(entities.Data))
	for i, entityData := range entities.Data {
		e, _ := ConvertWorkItemLinkTypeToModel(app.WorkItemLinkTypeSingle{Data: entityData})
		modelEntities[i] = e
	}
	return app.GenerateEntitiesTag(modelEntities)
}

func generateWorkItemLinkTypeTag(entity app.WorkItemLinkTypeSingle) string {
	e, _ := ConvertWorkItemLinkTypeToModel(entity)
	return app.GenerateEntityTag(e)
}

func convertWorkItemTypesToConditionalEntities(workItemTypeList app.WorkItemTypeList) []app.ConditionalRequestEntity {
	conditionalWorkItemTypes := make([]app.ConditionalRequestEntity, len(workItemTypeList.Data))
	for i, data := range workItemTypeList.Data {
		conditionalWorkItemTypes[i] = convertWorkItemTypeToModel(*data)
	}
	return conditionalWorkItemTypes
}

func getWorkItemTypeUpdatedAt(appWorkItemType app.WorkItemTypeSingle) time.Time {
	return *appWorkItemType.Data.Attributes.UpdatedAt
}

func getWorkItemLinkTypeUpdatedAt(appWorkItemLinkType app.WorkItemLinkTypeSingle) time.Time {
	return *appWorkItemLinkType.Data.Attributes.UpdatedAt
}

//-----------------------------------------------------------------------------
// Test on work item type authorization
//-----------------------------------------------------------------------------

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemTypeSuite) TestUnauthorizeWorkItemTypeCreate() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), s.getWorkItemTypeTestDataFunc(), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWIT-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkitemtypeController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkitemtypeController(service, controller)
		return nil
	})
}

func (s *workItemTypeSuite) getWorkItemTypeTestDataFunc() func(*testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((s.Configuration.GetTokenPrivateKey()))
	return func(t *testing.T) []testSecureAPI {
		if err != nil {
			t.Fatal("Could not parse Key ", err)
		}
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
		require.Nil(t, err)

		createWITPayloadString := bytes.NewBuffer([]byte(`{"fields": {"system.administrator": {"Required": true,"Type": {"Kind": "string"}}},"name": "Epic"}`))

		return []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWITPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWITPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWITPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWITPayloadString,
				jwtToken:           "",
			},
			// Try fetching a random work Item Type
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypes, space.SystemSpace.String()) + "/2e889d4e-49a9-463b-8cd4-6a3a95155103",
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			}, {
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypesSourceLinkTypes, space.SystemSpace, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			}, {
				method:             http.MethodGet,
				url:                fmt.Sprintf(endpointWorkItemTypesTargetLinkTypes, space.SystemSpace, "2e889d4e-49a9-463b-8cd4-6a3a95155103"),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
	}
}
