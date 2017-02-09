package main

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
)

func TestAuthorizeLoginOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	controller := LoginController{auth: TestLoginService{}}
	test.AuthorizeLoginTemporaryRedirect(t, nil, nil, &controller)
}

type TestLoginService struct{}

func (t TestLoginService) Perform(ctx *app.AuthorizeLoginContext) error {
	return ctx.TemporaryRedirect()
}

func (t TestLoginService) CreateKeycloakUser(accessToken string, ctx context.Context) (*account.Identity, *account.User, error) {
	return nil, nil, nil
}
