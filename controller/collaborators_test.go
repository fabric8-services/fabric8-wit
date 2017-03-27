package controller_test

import (
	"strings"
	"testing"

	"context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/auth"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/resource"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// var spaceCollaboratorID = uuid.NewV4().String()

type DummyPolicyManager struct {
	rest *TestCollaboratorsREST
}

func (m *DummyPolicyManager) GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*auth.KeycloakPolicy, *string, error) {
	pat := ""
	return m.rest.policy, &pat, nil
}

func (m *DummyPolicyManager) UpdatePolicy(ctx context.Context, request *goa.RequestData, policy auth.KeycloakPolicy, pat string) error {
	return nil
}

func (m *DummyPolicyManager) VerifyUser(ctx context.Context, request *goa.RequestData, resourceName string) (bool, error) {
	jwtToken := goajwt.ContextJWT(ctx)
	if jwtToken == nil {
		return false, errors.NewUnauthorizedError("Missing token")
	}
	id := jwtToken.Claims.(token.MapClaims)["sub"].(string)
	return strings.Contains(m.rest.policy.Config.UserIDs, id), nil
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

	svc := testsupport.ServiceAsUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), rest.testIdentity1)
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

func (rest *TestCollaboratorsREST) TestAddCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.AddCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, "wrongFormatID")
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

func (rest *TestCollaboratorsREST) TestAddCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	test.AddCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestAddCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity2.ID.String()})

	test.AddCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity1.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsUnauthorizedIfNoToken() {
	svc, ctrl := rest.UnSecuredController()
	test.RemoveCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsUnauthorizedIfCurrentUserIsNotCollaborator() {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Collaborators-Service", almtoken.NewManagerWithPrivateKey(priv), rest.testIdentity2)
	ctrl := NewCollaboratorsController(svc, rest.db, rest.Configuration, &DummyPolicyManager{rest: rest})

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String()})

	test.RemoveCollaboratorsUnauthorized(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity2.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsFailsIfTryToRemoveSpaceOwner() {
	svc, ctrl := rest.SecuredController()

	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	rest.policy.AddUserToPolicy(rest.testIdentity2.ID.String())
	rest.checkCollaborators([]string{rest.testIdentity1.ID.String(), rest.testIdentity2.ID.String()})

	test.RemoveCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, rest.testIdentity1.ID.String())
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsWithRandomSpaceIDNotFound() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.RemoveCollaboratorsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), uuid.NewV4().String())
}

func (rest *TestCollaboratorsREST) TestRemoveCollaboratorsWithWrongUserIDFormatReturnsBadRequest() {
	rest.policy.AddUserToPolicy(rest.testIdentity1.ID.String())
	svc, ctrl := rest.SecuredController()
	test.RemoveCollaboratorsBadRequest(rest.T(), svc.Context, svc, ctrl, rest.spaceID, "wrongFormatID")
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
