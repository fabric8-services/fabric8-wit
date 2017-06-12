package authz_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/auth"
	"github.com/almighty/almighty-core/codebase"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	"github.com/almighty/almighty-core/space/authz"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	scopes = []string{"read:test", "admin:test"}
)

func TestAuthz(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	suite.Run(t, new(TestAuthzSuite))
}

type TestAuthzSuite struct {
	suite.Suite
	authzService *authz.KeycloakAuthzService
}

func (s *TestAuthzSuite) SetupSuite() {
	var err error
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	var resource *space.Resource
	s.authzService = authz.NewAuthzService(nil, &db{app{resource: resource}})
}

func (s *TestAuthzSuite) TestFailsIfNoTokenInContext() {
	ctx := context.Background()
	spaceID := ""
	_, err := s.authzService.Authorize(ctx, "", spaceID)
	require.NotNil(s.T(), err)
}

func (s *TestAuthzSuite) TestUserAmongSpaceCollaboratorsOK() {
	spaceID := uuid.NewV4().String()
	authzPayload := authz.AuthorizationPayload{Permissions: []authz.Permissions{{ResourceSetName: &spaceID}}}
	ok := s.checkPermissions(authzPayload, spaceID)
	require.True(s.T(), ok)
}

func (s *TestAuthzSuite) TestUserIsNotAmongSpaceCollaboratorsFails() {
	spaceID1 := uuid.NewV4().String()
	spaceID2 := uuid.NewV4().String()
	authzPayload := authz.AuthorizationPayload{Permissions: []authz.Permissions{{ResourceSetName: &spaceID1}}}
	ok := s.checkPermissions(authzPayload, spaceID2)
	require.False(s.T(), ok)
}

func (s *TestAuthzSuite) checkPermissions(authzPayload authz.AuthorizationPayload, spaceID string) bool {
	resource := &space.Resource{}
	authzService := authz.NewAuthzService(nil, &db{app{resource: resource}})
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	testIdentity := testsupport.TestIdentity
	svc := testsupport.ServiceAsUserWithAuthz("SpaceAuthz-Service", almtoken.NewManagerWithPrivateKey(priv), priv, testIdentity, authzPayload)
	resource.UpdatedAt = time.Now()

	ok, err := authzService.Authorize(svc.Context, "", spaceID)
	require.Nil(s.T(), err)
	return ok
}

type app struct {
	resource *space.Resource
}

type db struct {
	app
}

type trx struct {
	app
}

type resourceRepo struct {
	resource *space.Resource
}

func (t *trx) Commit() error {
	return nil
}

func (t *trx) Rollback() error {
	return nil
}

func (d *db) BeginTransaction() (application.Transaction, error) {
	return &trx{}, nil
}

func (a *app) WorkItems() workitem.WorkItemRepository {
	return nil
}

func (a *app) WorkItemTypes() workitem.WorkItemTypeRepository {
	return nil
}

func (a *app) Trackers() application.TrackerRepository {
	return nil
}

func (a *app) TrackerQueries() application.TrackerQueryRepository {
	return nil
}

func (a *app) SearchItems() application.SearchRepository {
	return nil
}

func (a *app) Identities() account.IdentityRepository {
	return nil
}

func (a *app) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return nil
}

func (a *app) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return nil
}

func (a *app) WorkItemLinks() link.WorkItemLinkRepository {
	return nil
}

func (a *app) Comments() comment.Repository {
	return nil
}

func (a *app) Spaces() space.Repository {
	return nil
}

func (a *app) SpaceResources() space.ResourceRepository {
	return &resourceRepo{a.resource}
}

func (a *app) Iterations() iteration.Repository {
	return nil
}

func (a *app) Users() account.UserRepository {
	return nil
}

func (a *app) Areas() area.Repository {
	return nil
}

func (a *app) OauthStates() auth.OauthStateReferenceRepository {
	return nil
}

func (a *app) Codebases() codebase.Repository {
	return nil
}

func (r *resourceRepo) Create(ctx context.Context, s *space.Resource) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Save(ctx context.Context, s *space.Resource) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Load(ctx context.Context, ID uuid.UUID) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	return nil
}

func (r *resourceRepo) LoadBySpace(ctx context.Context, spaceID *uuid.UUID) (*space.Resource, error) {
	resource := &space.Resource{}
	past := time.Now().Unix() - 1000
	resource.UpdatedAt = time.Unix(past, 0)
	return resource, nil
}
