//************************************************************************//
// API "alm": Application Contexts
//
// Generated with goagen v0.0.1, command line:
// $ goagen.exe
// --design=github.com/ALMighty/almighty-core/design
// --out=$(GOPATH)\src\github.com\ALMighty\almighty-core
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import (
	"github.com/goadesign/goa"
	"golang.org/x/net/context"
)

// AuthorizeLoginContext provides the login authorize action context.
type AuthorizeLoginContext struct {
	context.Context
	*goa.ResponseData
	*goa.RequestData
	Service *goa.Service
}

// NewAuthorizeLoginContext parses the incoming request URL and body, performs validations and creates the
// context used by the login controller authorize action.
func NewAuthorizeLoginContext(ctx context.Context, service *goa.Service) (*AuthorizeLoginContext, error) {
	var err error
	req := goa.ContextRequest(ctx)
	rctx := AuthorizeLoginContext{Context: ctx, ResponseData: goa.ContextResponse(ctx), RequestData: req, Service: service}
	return &rctx, err
}

// OK sends a HTTP response with status code 200.
func (ctx *AuthorizeLoginContext) OK(r *AuthToken) error {
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.authtoken+json")
	return ctx.Service.Send(ctx.Context, 200, r)
}

// Unauthorized sends a HTTP response with status code 401.
func (ctx *AuthorizeLoginContext) Unauthorized() error {
	ctx.ResponseData.WriteHeader(401)
	return nil
}

// ShowVersionContext provides the version show action context.
type ShowVersionContext struct {
	context.Context
	*goa.ResponseData
	*goa.RequestData
	Service *goa.Service
}

// NewShowVersionContext parses the incoming request URL and body, performs validations and creates the
// context used by the version controller show action.
func NewShowVersionContext(ctx context.Context, service *goa.Service) (*ShowVersionContext, error) {
	var err error
	req := goa.ContextRequest(ctx)
	rctx := ShowVersionContext{Context: ctx, ResponseData: goa.ContextResponse(ctx), RequestData: req, Service: service}
	return &rctx, err
}

// OK sends a HTTP response with status code 200.
func (ctx *ShowVersionContext) OK(r *Version) error {
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.version+json")
	return ctx.Service.Send(ctx.Context, 200, r)
}
