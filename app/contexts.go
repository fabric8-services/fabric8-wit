//************************************************************************//
// API "alm": Application Contexts
//
// Generated with goagen v0.2.dev, command line:
// $ goagen
// --design=github.com/almighty/almighty-core/design
// --out=$(GOPATH)/src/github.com/almighty/almighty-core
// --version=v0.2.dev
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
}

// NewAuthorizeLoginContext parses the incoming request URL and body, performs validations and creates the
// context used by the login controller authorize action.
func NewAuthorizeLoginContext(ctx context.Context, service *goa.Service) (*AuthorizeLoginContext, error) {
	var err error
	resp := goa.ContextResponse(ctx)
	resp.Service = service
	req := goa.ContextRequest(ctx)
	rctx := AuthorizeLoginContext{Context: ctx, ResponseData: resp, RequestData: req}
	return &rctx, err
}

// OK sends a HTTP response with status code 200.
func (ctx *AuthorizeLoginContext) OK(r *AuthToken) error {
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.authtoken+json")
	return ctx.ResponseData.Service.Send(ctx.Context, 200, r)
}

// Unauthorized sends a HTTP response with status code 401.
func (ctx *AuthorizeLoginContext) Unauthorized() error {
	ctx.ResponseData.WriteHeader(401)
	return nil
}

// GenerateLoginContext provides the login generate action context.
type GenerateLoginContext struct {
	context.Context
	*goa.ResponseData
	*goa.RequestData
}

// NewGenerateLoginContext parses the incoming request URL and body, performs validations and creates the
// context used by the login controller generate action.
func NewGenerateLoginContext(ctx context.Context, service *goa.Service) (*GenerateLoginContext, error) {
	var err error
	resp := goa.ContextResponse(ctx)
	resp.Service = service
	req := goa.ContextRequest(ctx)
	rctx := GenerateLoginContext{Context: ctx, ResponseData: resp, RequestData: req}
	return &rctx, err
}

// OK sends a HTTP response with status code 200.
func (ctx *GenerateLoginContext) OK(r AuthTokenCollection) error {
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.authtoken+json; type=collection")
	return ctx.ResponseData.Service.Send(ctx.Context, 200, r)
}

// Unauthorized sends a HTTP response with status code 401.
func (ctx *GenerateLoginContext) Unauthorized() error {
	ctx.ResponseData.WriteHeader(401)
	return nil
}

// ShowVersionContext provides the version show action context.
type ShowVersionContext struct {
	context.Context
	*goa.ResponseData
	*goa.RequestData
}

// NewShowVersionContext parses the incoming request URL and body, performs validations and creates the
// context used by the version controller show action.
func NewShowVersionContext(ctx context.Context, service *goa.Service) (*ShowVersionContext, error) {
	var err error
	resp := goa.ContextResponse(ctx)
	resp.Service = service
	req := goa.ContextRequest(ctx)
	rctx := ShowVersionContext{Context: ctx, ResponseData: resp, RequestData: req}
	return &rctx, err
}

// OK sends a HTTP response with status code 200.
func (ctx *ShowVersionContext) OK(r *Version) error {
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.version+json")
	return ctx.ResponseData.Service.Send(ctx.Context, 200, r)
}
