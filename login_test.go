package main

import (
	"testing"

	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/resource"
)

func TestAuthorizeLoginOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	controller := LoginController{}
	_, res := test.AuthorizeLoginOK(t, nil, nil, &controller)

	if res.Token == "" {
		t.Error("Token not generated")
	}
}
