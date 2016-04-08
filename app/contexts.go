//************************************************************************//
// API "alm": Application Contexts
//
// Generated with goagen v0.0.1, command line:
// $ goagen
// --out=$(GOPATH)/src/github.com/almighty/almighty-design
// --design=github.com/almighty/almighty-design/design
// --pkg=app
//
// The content of this file is auto-generated, DO NOT MODIFY
//************************************************************************//

package app

import (
	"github.com/goadesign/goa"
	"golang.org/x/net/context"
)

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
	ctx.ResponseData.Header().Set("Content-Type", "application/vnd.version")
	return ctx.Service.Send(ctx.Context, 200, r)
}
