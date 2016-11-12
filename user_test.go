package main

import (
	"errors"
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
	almtoken "github.com/almighty/almighty-core/token"
	token "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

func newUserController() *UserController {
	return newUserControllerWithRepo(&TestIdentityRepository{})
}

func newUserControllerWithRepo(repo *TestIdentityRepository) *UserController {
	pub, _ := almtoken.ParsePublicKey([]byte(almtoken.RSAPublicKey))
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	return NewUserController(goa.New("alm-test"), repo, almtoken.NewManager(pub, priv))
}

func TestCurrentAuthorizedMissingUUID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController()
	test.ShowUserBadRequest(t, ctx, nil, controller)
}

func TestCurrentAuthorizedNonUUID(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["uuid"] = "aa"
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController()
	test.ShowUserBadRequest(t, ctx, nil, controller)
}

func TestCurrentAuthorizedMissingIdentity(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["uuid"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	controller := newUserController()
	test.ShowUserUnauthorized(t, ctx, nil, controller)
}

func TestCurrentAuthorizedOK(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["uuid"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	ident := account.Identity{FullName: "Test user", ImageURL: "http://a.com"}
	controller := newUserControllerWithRepo(&TestIdentityRepository{Identity: &ident})
	_, user := test.ShowUserOK(t, ctx, nil, controller)

	if *user.FullName != ident.FullName {
		t.Errorf("Expected FullName %v to match %v", user.FullName, ident.FullName)
	}

	if *user.ImageURL != ident.ImageURL {
		t.Errorf("Expected ImageURL %v to match %v", user.ImageURL, ident.ImageURL)
	}
}

type TestIdentityRepository struct {
	Identity *account.Identity
}

// Load returns a single Identity as a Database Model
func (m *TestIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*account.Identity, error) {
	if m.Identity == nil {
		return nil, errors.New("not found")
	}
	return m.Identity, nil
}

// Create creates a new record.
func (m *TestIdentityRepository) Create(ctx context.Context, model *account.Identity) error {
	m.Identity = model
	return nil
}

// Save modifies a single record.
func (m *TestIdentityRepository) Save(ctx context.Context, model *account.Identity) error {
	return m.Create(ctx, model)
}

// Delete removes a single record.
func (m *TestIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.Identity = nil
	return nil
}

// Query expose an open ended Query model
func (m *TestIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]*account.Identity, error) {
	return []*account.Identity{m.Identity}, nil
}

func (m *TestIdentityRepository) Search(ctx context.Context, q string, start int, limit int) ([]account.Identity, int, error) {
	return []account.Identity{}, 0, nil
}
