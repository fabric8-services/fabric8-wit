package main

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/area"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/iteration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"
	"github.com/almighty/almighty-core/workitem/link"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func newUserController(identity *account.Identity, user *account.User) *UserController {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	return NewUserController(goa.New("alm-test"), newGormTestBase(identity, user), almtoken.NewManagerWithPrivateKey(priv))
}

func TestCurrentAuthorizedMissingUUID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController(nil, nil)
	test.ShowUserBadRequest(t, ctx, nil, controller)
}

func TestCurrentAuthorizedNonUUID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = "aa"
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController(nil, nil)
	test.ShowUserBadRequest(t, ctx, nil, controller)
}

func TestCurrentAuthorizedMissingIdentity(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController(nil, nil)
	test.ShowUserUnauthorized(t, ctx, nil, controller)
}

func TestCurrentAuthorizedOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	usr := account.User{FullName: "Test User", ImageURL: "someURL", Email: "email@domain.com", ID: uuid.NewV4()}
	ident := account.Identity{ID: uuid.NewV4(), Username: "TestUser", ProviderType: account.KeycloakIDP, User: usr, UserID: account.NullUUID{UUID: usr.ID, Valid: true}}
	controller := newUserController(&ident, &usr)
	_, identity := test.ShowUserOK(t, ctx, nil, controller)

	assert.NotNil(t, identity)

	assert.Equal(t, usr.FullName, *identity.Data.Attributes.FullName)
	assert.Equal(t, ident.Username, *identity.Data.Attributes.Username)
	assert.Equal(t, usr.ImageURL, *identity.Data.Attributes.ImageURL)
	assert.Equal(t, usr.Email, *identity.Data.Attributes.Email)
	assert.Equal(t, ident.ProviderType, *identity.Data.Attributes.Provider)
}

type TestIdentityRepository struct {
	Identity *account.Identity
}

// Load returns a single Identity as a Database Model
func (m TestIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*account.Identity, error) {
	if m.Identity == nil {
		return nil, errors.New("not found")
	}
	return m.Identity, nil
}

// Create creates a new record.
func (m TestIdentityRepository) Create(ctx context.Context, model *account.Identity) error {
	m.Identity = model
	return nil
}

// Save modifies a single record.
func (m TestIdentityRepository) Save(ctx context.Context, model *account.Identity) error {
	return m.Create(ctx, model)
}

// Delete removes a single record.
func (m TestIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.Identity = nil
	return nil
}

// Query expose an open ended Query model
func (m TestIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*account.Identity, error) {
	return []*account.Identity{m.Identity}, nil
}

func (m TestIdentityRepository) List(ctx context.Context) (*app.IdentityArray, error) {
	rows := []account.Identity{*m.Identity}

	res := app.IdentityArray{}
	res.Data = make([]*app.IdentityData, len(rows))
	for index, value := range rows {
		ident := value.ConvertIdentityFromModel()
		res.Data[index] = ident.Data
	}
	return &res, nil
}

func (m TestIdentityRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	return true
}

type TestUserRepository struct {
	User *account.User
}

func (m TestUserRepository) Load(ctx context.Context, id uuid.UUID) (*account.User, error) {
	if m.User == nil {
		return nil, errors.New("not found")
	}
	return m.User, nil
}

// Create creates a new record.
func (m TestUserRepository) Create(ctx context.Context, u *account.User) error {
	m.User = u
	return nil
}

// Save modifies a single record
func (m TestUserRepository) Save(ctx context.Context, model *account.User) error {
	return m.Create(ctx, model)
}

// Delete removes a single record.
func (m TestUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.User = nil
	return nil
}

// List return all users
func (m TestUserRepository) List(ctx context.Context) ([]*account.User, error) {
	return []*account.User{m.User}, nil
}

// Query expose an open ended Query model
func (m TestUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*account.User, error) {
	return []*account.User{m.User}, nil
}

type GormTestBase struct {
	IdentityRepository account.IdentityRepository
	UserRepository     account.UserRepository
}

func (g *GormTestBase) WorkItems() workitem.WorkItemRepository {
	return nil
}

func (g *GormTestBase) WorkItemTypes() workitem.WorkItemTypeRepository {
	return nil
}

func (g *GormTestBase) Spaces() space.Repository {
	return nil
}

func (g *GormTestBase) Trackers() application.TrackerRepository {
	return nil
}
func (g *GormTestBase) TrackerQueries() application.TrackerQueryRepository {
	return nil
}

func (g *GormTestBase) SearchItems() application.SearchRepository {
	return nil
}

// Identities creates new Identity repository
func (g *GormTestBase) Identities() account.IdentityRepository {
	return g.IdentityRepository
}

// Users creates new user repository
func (g *GormTestBase) Users() account.UserRepository {
	return g.UserRepository
}

// WorkItemLinkCategories returns a work item link category repository
func (g *GormTestBase) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return nil
}

// WorkItemLinkTypes returns a work item link type repository
func (g *GormTestBase) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return nil
}

// WorkItemLinks returns a work item link repository
func (g *GormTestBase) WorkItemLinks() link.WorkItemLinkRepository {
	return nil
}

// Comments returns a work item comments repository
func (g *GormTestBase) Comments() comment.Repository {
	return nil
}

// Iterations returns a iteration repository
func (g *GormTestBase) Iterations() iteration.Repository {
	return nil
}

// Iterations returns a iteration repository
func (g *GormTestBase) Areas() area.Repository {
	return nil
}

func (g *GormTestBase) DB() *gorm.DB {
	return nil
}

// SetTransactionIsolationLevel sets the isolation level for
// See also https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
func (g *GormTestBase) SetTransactionIsolationLevel(level interface{}) error {
	return nil
}

func (g *GormTestBase) Commit() error {
	return nil
}

func (g *GormTestBase) Rollback() error {
	return nil
}

// Begin implements TransactionSupport
func (g *GormTestBase) BeginTransaction() (application.Transaction, error) {
	return g, nil
}

func newGormTestBase(identity *account.Identity, user *account.User) *GormTestBase {
	return &GormTestBase{IdentityRepository: TestIdentityRepository{Identity: identity}, UserRepository: TestUserRepository{User: user}}
}
