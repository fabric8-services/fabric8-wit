package main

import (
	"testing"

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
