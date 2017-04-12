package controller_test

import (
	"strings"
	"testing"

	"context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/auth"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space/authz"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	idnType = "identities"
)

type DummyPolicyManager struct {
	rest *TestCollaboratorsREST
}

type DummySpaceAuthzService struct {
	rest *TestCollaboratorsREST
}

func (s *DummySpaceAuthzService) Authorize(ctx context.Context, endpoint string, spaceID string) (bool, error) {
	jwtToken := goajwt.ContextJWT(ctx)
	if jwtToken == nil {
		return false, errors.NewUnauthorizedError("Missing token")
	}
	id := jwtToken.Claims.(token.MapClaims)["sub"].(string)
	return strings.Contains(s.rest.policy.Config.UserIDs, id), nil
}

func (s *DummySpaceAuthzService) Configuration() authz.AuthzConfiguration {
	return nil
}

func (m *DummyPolicyManager) GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*auth.KeycloakPolicy, *string, error) {
	pat := ""
	return m.rest.policy, &pat, nil
}

func (m *DummyPolicyManager) UpdatePolicy(ctx context.Context, request *goa.RequestData, policy auth.KeycloakPolicy, pat string) error {
	return nil
}

func (m *DummyPolicyManager) AddUserToPolicy(p *auth.KeycloakPolicy, userID string) bool {
	return p.AddUserToPolicy(userID)
}

func (m *DummyPolicyManager) RemoveUserFromPolicy(p *auth.KeycloakPolicy, userID string) bool {
	return p.RemoveUserFromPolicy(userID)
}

type TestCollaboratorsREST struct {
	gormtestsupport.DBTestSuite

	db            *gormapplication.GormDB
	clean         func()
	policy        *auth.KeycloakPolicy
	testIdentity1 account.Identity
	testIdentity2 account.Identity
	spaceID       string
}

func TestRunCollaboratorsREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestCollaboratorsREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestCollaboratorsREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)

	rest.policy = &auth.KeycloakPolicy{
		Name:             "TestCollaborators-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestCollaborators-"+uuid.NewV4().String(), "TestCollaborators")
	require.Nil(rest.T(), err)
	rest.testIdentity1 = testIdentity
	testIdentity, err = testsupport.CreateTestIdentity(rest.DB, "TestCollaborators-"+uuid.NewV4().String(), "TestCollaborators")
	require.Nil(rest.T(), err)
	rest.testIdentity2 = testIdentity
	space := rest.createSpace()
	rest.spaceID = space.ID.String()
}

func (rest *TestCollaboratorsREST) TearDownTest() {
	rest.clean()
}

func (rest *TestCollaboratorsREST) SecuredController() (*goa.Service, *CollaboratorsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))

	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), rest.testIdentity1, &DummySpaceAuthzService{rest})
	return svc, NewCollaboratorsController(svc, rest.db, rest.Configuration, &DummyPolicyManager{rest: rest})
}

func (rest *TestCollaboratorsREST) UnSecuredController() (*goa.Service, *CollaboratorsController) {
	svc := goa.New("Collaborators-Service")
	return svc, NewCollaboratorsController(svc, rest.db, rest.Configuration, &DummyPolicyManager{rest: rest})
}

func (rest *TestCollaboratorsREST) TestListCollaboratorsWithRandomSpaceIDNotFound() {
	svc, ctrl := rest.UnSecuredController()
	test.ListCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestCollaboratorsREST) TestListCollaboratorsWithWrongSpaceIDFormatReturnsBadRequest() {
	svc, ctrl := rest.UnSecuredController()
	test.ListCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, "wrongFormatID", nil, nil)
}

func (rest *TestCollaboratorsREST) TestListCollaboratorsOk() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})

	rest.policy.RemoveUserFromPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsWithRandomSpaceIDNotFound() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.AddCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), uuid.NewV4().String())
}

func (rest *TestCollaboratorsREST) TestAddManyCollaboratorsWithRandomSpaceIDNotFound() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	payload := &app.AddManyCollaboratorsPayload{Data: []*app.UpdateUserID{}}
	test.AddManyCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), payload)
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.AddCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, "wrongFormatID")
}

func (rest *TestCollaboratorsREST) TestAddManyCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	payload := &app.AddManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: "wrongFormatID", Type: idnType}}}
	test.AddManyCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsOk() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})

	test.AddCollaboratorsOK(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})
}

func (rest *TestCollaboratorsREST) TestAddManyCollaboratorsOk() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})

	payload := &app.AddManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity1.ID.String(), Type: idnType}, {ID: rest.testIdentity2.ID.String(), Type: idnType}}}
	test.AddManyCollaboratorsOK(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	test.AddCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestAddManyCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	payload := &app.AddManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity2.ID.String(), Type: idnType}}}
	test.AddManyCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity2.ID.String()})

	test.AddCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity1.ID.String())
}

func (rest *TestCollaboratorsREST) TestAddManyCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity2.ID.String()})

	payload := &app.AddManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity1.ID.String(), Type: idnType}}}
	test.AddManyCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	test.RemoveCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity2.ID.String(), Type: idnType}}}
	test.RemoveManyCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), rest.testIdentity2, &DummySpaceAuthzService{rest})
	ctrl := NewCollaboratorsController(svc, rest.db, rest.Configuration, &DummyPolicyManager{rest: rest})

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})

	test.RemoveCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), rest.testIdentity2, &DummySpaceAuthzService{rest})
	ctrl := NewCollaboratorsController(svc, rest.db, rest.Configuration, &DummyPolicyManager{rest: rest})

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity2.ID.String(), Type: idnType}}}

	test.RemoveManyCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsFailsIfTryToRemoveSpaceOwner() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})

	test.RemoveCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity1.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsFailsIfTryToRemoveSpaceOwner() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity1.ID.String(), Type: idnType}}}

	test.RemoveManyCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsWithRandomSpaceIDNotFound() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.RemoveCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), uuid.NewV4().String())
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsWithRandomSpaceIDNotFound() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: uuid.NewV4().String(), Type: idnType}}}

	test.RemoveManyCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), payload)
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.RemoveCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, "wrongFormatID")
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: "wrongFormatID", Type: idnType}}}

	test.RemoveManyCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) checkCollaborators(userIDs []string) {
	svc, ctrl := rest.UnSecuredController()

	_, users := test.ListCollaboratorsOK(rest.T(), svc.Context, svc, ctrl, rest.spaceID, nil, nil)
	require.NotNil(rest.T(), users)
	require.Equal(rest.T(), len(userIDs), len(users.Data))
	for i, id := range userIDs {
		require.NotNil(rest.T(), users.Data[i].ID)
		require.Equal(rest.T(), id, *users.Data[i].ID)
	}
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsOk() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})

	test.RemoveCollaboratorsOK(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveManyCollaboratorsOk() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})
	payload := &app.RemoveManyCollaboratorsPayload{Data: []*app.UpdateUserID{{ID: rest.testIdentity2.ID.String(), Type: idnType}}}

	test.RemoveManyCollaboratorsOK(rest.T(), svc.Context, svc, ctrl, rest.spaceID, payload)
}

func (rest *TestCollaboratorsREST) createSpace() app.Space {
	svc, _ := rest.SecuredController()
	spaceCtrl := NewSpaceController(svc, rest.db, rest.Configuration, &DummyResourceManager{})
	require.NotNil(rest.T(), spaceCtrl)
	name := "TestCollaborators-space-" + uuid.NewV4().String()
	description := "description"
	spacePayload := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	_, sp := test.CreateSpaceCreated(rest.T(), svc.Context, svc, spaceCtrl, spacePayload)
	require.NotNil(rest.T(), sp)
	require.NotNil(rest.T(), sp.Data)
	return *sp.Data
}

type TestSpaceAuthzService struct {
	owner account.Identity
}

func (s *TestSpaceAuthzService) Authorize(ctx context.Context, endpoint string, spaceID string) (bool, error) {
	jwtToken := goajwt.ContextJWT(ctx)
	if jwtToken == nil {
		return false, errors.NewUnauthorizedError("Missing token")
	}
	id := jwtToken.Claims.(token.MapClaims)["sub"].(string)
	return s.owner.ID.String() == id, nil
}

func (s *TestSpaceAuthzService) Configuration() authz.AuthzConfiguration {
	return nil
}

func CreateSecuredSpace(t *testing.T, db application.DB, config SpaceConfiguration, owner account.Identity) app.Space {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), owner, &TestSpaceAuthzService{owner})
	spaceCtrl := NewSpaceController(svc, db, config, &DummyResourceManager{})
	require.NotNil(t, spaceCtrl)
	name := "TestCollaborators-space-" + uuid.NewV4().String()
	description := "description"
	spacePayload := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	_, sp := test.CreateSpaceCreated(t, svc.Context, svc, spaceCtrl, spacePayload)
	require.NotNil(t, sp)
	require.NotNil(t, sp.Data)
	return *sp.Data
}
