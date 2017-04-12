package controller_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/almighty/almighty-core/path"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type TestIterationREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunIterationREST(t *testing.T) {
	// given
	suite.Run(t, &TestIterationREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestIterationREST) SecuredController() (*goa.Service, *IterationController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", almtoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewIterationController(svc, rest.db, rest.Configuration)
}

func (rest *TestIterationREST) UnSecuredController() (*goa.Service, *IterationController) {
	svc := goa.New("Iteration-Service")
	return svc, NewIterationController(svc, rest.db, rest.Configuration)
}

func (rest *TestIterationREST) TestSuccessCreateChildIteration() {
	// given
	parent := createSpaceAndIteration(rest.T(), rest.db)
	ri, err := rest.db.Iterations().Root(context.Background(), parent.SpaceID)
	require.Nil(rest.T(), err)
	parentID := parent.ID
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.CreateChildIterationCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	require.NotNil(rest.T(), created)
	assertChildIterationLinking(rest.T(), created.Data)
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	expectedParentPath := parent.Path.String() + path.SepInService + parentID.String()
	expectedResolvedParentPath := path.SepInService + ri.Name + path.SepInService + parent.Name
	assert.Equal(rest.T(), expectedParentPath, *created.Data.Attributes.ParentPath)
	assert.Equal(rest.T(), expectedResolvedParentPath, *created.Data.Attributes.ResolvedParentPath)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingName() {
	// given
	parentID := createSpaceAndIteration(rest.T(), rest.db).ID
	ci := getChildIterationPayload(nil)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildIterationBadRequest(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingParent() {
	// given
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildIterationNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationNotAuthorized() {
	// when
	parentID := createSpaceAndIteration(rest.T(), rest.db).ID
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateChildIterationUnauthorized(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestShowIterationOK() {
	// given
	itrID := createSpaceAndIteration(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itrID.ID.String(), nil, nil)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationOKUsingExpiredIfModifiedSinceHeader() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSinceHeader := app.ToHTTPTime(itr.UpdatedAt.Add(-1 * time.Hour))
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &ifModifiedSinceHeader, nil)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationOKUsingExpiredIfNoneMatchHeader() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := "foo"
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), nil, &ifNoneMatch)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationNotModifiedUsingIfModifiedSinceHeader() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when/then
	rest.T().Log("Iteration:", itr, " updatedAt: ", itr.UpdatedAt)
	ifModifiedSinceHeader := app.ToHTTPTime(itr.UpdatedAt)
	test.ShowIterationNotModified(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &ifModifiedSinceHeader, nil)
}

func (rest *TestIterationREST) TestShowIterationNotModifiedUsingIfNoneMatchHeader() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when/then
	ifNoneMatch := app.GenerateEntityTag(itr)
	test.ShowIterationNotModified(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), nil, &ifNoneMatch)
}

func (rest *TestIterationREST) TestFailShowIterationMissing() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	test.ShowIterationNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestIterationREST) TestSuccessUpdateIteration() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	newName := "Sprint 1001"
	newDesc := "New Description"
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				Name:        &newName,
				Description: &newDesc,
			},
			ID:   &itr.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	// when
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
	// then
	assert.Equal(rest.T(), newName, *updated.Data.Attributes.Name)
	assert.Equal(rest.T(), newDesc, *updated.Data.Attributes.Description)
	require.NotNil(rest.T(), updated.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, updated.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, updated.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestSuccessUpdateIterationWithWICounts() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	newName := "Sprint 1001"
	newDesc := "New Description"
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				Name:        &newName,
				Description: &newDesc,
			},
			ID:   &itr.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	// add WI to this iteration and test counts in the response of update iteration API
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestSuccessUpdateIterationWithWICounts user", "test provider")
	require.Nil(rest.T(), err)
	wirepo := workitem.NewWorkItemRepository(rest.DB)
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	for i := 0; i < 4; i++ {
		wi, err := wirepo.Create(
			ctx, itr.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: itr.ID.String(),
			}, testIdentity.ID)
		require.NotNil(rest.T(), wi)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), wi)
	}
	for i := 0; i < 5; i++ {
		wi, err := wirepo.Create(
			ctx, itr.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: itr.ID.String(),
			}, testIdentity.ID)
		require.NotNil(rest.T(), wi)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), wi)
	}
	svc, ctrl := rest.SecuredController()
	// when
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
	// then
	require.NotNil(rest.T(), updated)
	assert.Equal(rest.T(), newName, *updated.Data.Attributes.Name)
	assert.Equal(rest.T(), newDesc, *updated.Data.Attributes.Description)
	require.NotNil(rest.T(), updated.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 9, updated.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 5, updated.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestFailUpdateIterationNotFound() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	itr.ID = uuid.NewV4()
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	// when/then
	test.UpdateIterationNotFound(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestFailUpdateIterationUnauthorized() {
	// given
	itr := createSpaceAndIteration(rest.T(), rest.db)
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.UpdateIterationUnauthorized(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestIterationStateTransitions() {
	// given
	itr1 := createSpaceAndIteration(rest.T(), rest.db)
	assert.Equal(rest.T(), iteration.IterationStateNew, itr1.State)
	startState := iteration.IterationStateStart
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr1.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(rest.T(), startState, *updated.Data.Attributes.State)
	// create another iteration in same space and then change State to start
	itr2 := iteration.Iteration{
		Name:    "Spring 123",
		SpaceID: itr1.SpaceID,
		Path:    itr1.Path,
	}
	err := rest.db.Iterations().Create(context.Background(), &itr2)
	require.Nil(rest.T(), err)
	payload2 := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr2.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	test.UpdateIterationBadRequest(rest.T(), svc.Context, svc, ctrl, itr2.ID.String(), &payload2)
	// now close first iteration
	closeState := iteration.IterationStateClose
	payload.Data.Attributes.State = &closeState
	_, updated = test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(rest.T(), closeState, *updated.Data.Attributes.State)
	// try to start iteration 2 now
	_, updated2 := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr2.ID.String(), &payload2)
	assert.Equal(rest.T(), startState, *updated2.Data.Attributes.State)
}

func (rest *TestIterationREST) TestRootIterationCanNotStart() {
	// given
	itr1 := createSpaceAndIteration(rest.T(), rest.db)
	var ri *iteration.Iteration
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()
		var err error
		ri, err = repo.Root(context.Background(), itr1.SpaceID)
		return err
	})
	require.Nil(rest.T(), err)
	require.NotNil(rest.T(), ri)

	startState := iteration.IterationStateStart
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &ri.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	test.UpdateIterationBadRequest(rest.T(), svc.Context, svc, ctrl, ri.ID.String(), &payload)
}

func getChildIterationPayload(name *string) *app.CreateChildIterationPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	itType := iteration.APIStringTypeIteration

	return &app.CreateChildIterationPayload{
		Data: &app.Iteration{
			Type: itType,
			Attributes: &app.IterationAttributes{
				Name:    name,
				StartAt: &start,
				EndAt:   &end,
			},
		},
	}
}

func createSpaceAndIteration(t *testing.T, db application.DB) iteration.Iteration {
	var itr iteration.Iteration
	application.Transactional(db, func(app application.Application) error {
		repo := app.Iterations()

		newSpace := space.Space{
			Name: "Test 1" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(context.Background(), &newSpace)
		if err != nil {
			t.Error(err)
		}
		// above space should have a root iteration for itself
		ri := iteration.Iteration{
			Name:    newSpace.Name,
			SpaceID: newSpace.ID,
		}
		repo.Create(context.Background(), &ri)

		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		name := "Sprint #2"

		i := iteration.Iteration{
			Lifecycle: gormsupport.Lifecycle{
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
			},
			Name:    name,
			SpaceID: p.ID,
			StartAt: &start,
			EndAt:   &end,
			Path:    append(ri.Path, ri.ID),
		}
		repo.Create(context.Background(), &i)
		itr = i
		return nil
	})
	return itr
}

func assertIterationLinking(t *testing.T, target *app.Iteration) {
	assert.NotNil(t, target.ID)
	assert.Equal(t, iteration.APIStringTypeIteration, target.Type)
	assert.NotNil(t, target.Links.Self)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Space)
	require.NotNil(t, target.Relationships.Space.Links)
	require.NotNil(t, target.Relationships.Space.Links.Self)
	assert.True(t, strings.Contains(*target.Relationships.Space.Links.Self, "/api/spaces/"))
}

func assertChildIterationLinking(t *testing.T, target *app.Iteration) {
	assertIterationLinking(t, target)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Parent)
	require.NotNil(t, target.Relationships.Parent.Links)
	require.NotNil(t, target.Relationships.Parent.Links.Self)
}
