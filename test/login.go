package test

import (
	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/login"
	"github.com/goadesign/goa"
)

// ServiceAsUser creates a new service and fill the context with input user
func ServiceAsUser(serviceName string, u account.User) *goa.Service {
	svc := goa.New(serviceName)
	svc.Context = login.WithIdentity(svc.Context, u.ID.String())
	return svc
}
