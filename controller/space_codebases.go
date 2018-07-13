package controller

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	gemini "github.com/fabric8-services/fabric8-wit/codebase/analytics-gemini"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"

	"github.com/goadesign/goa"
)

// SpaceCodebasesController implements the space-codebases resource.
type SpaceCodebasesController struct {
	*goa.Controller
	db                    application.DB
	AnalyticsGeminiClient AnalyticsGeminiClientProvider
}

// NewSpaceCodebasesController creates a space-codebases controller.
func NewSpaceCodebasesController(service *goa.Service, db application.DB) *SpaceCodebasesController {
	return &SpaceCodebasesController{Controller: service.NewController("SpaceCodebasesController"), db: db}
}

// Create runs the create action.
func (c *SpaceCodebasesController) Create(ctx *app.CreateSpaceCodebasesContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	// Validate Request
	if ctx.Payload.Data == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data", nil).Expected("not nil"))
	}
	reqIter := ctx.Payload.Data
	if reqIter.Attributes.Type == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.type", nil).Expected("not nil"))
	}
	if reqIter.Attributes.URL == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.url", nil).Expected("not nil"))
	}
	// the default value of cveScan
	cveScan := true
	if reqIter.Attributes.CveScan != nil {
		cveScan = *reqIter.Attributes.CveScan
	}

	var cdb *codebase.Codebase
	err = application.Transactional(c.db, func(appl application.Application) error {
		sp, err := appl.Spaces().Load(ctx, ctx.SpaceID)
		if err != nil {
			return err
		}
		if *identityID != sp.OwnerID {
			return errors.NewForbiddenError("user is not the space owner")
		}
		cdb = &codebase.Codebase{
			SpaceID: ctx.SpaceID,
			Type:    *reqIter.Attributes.Type,
			URL:     *reqIter.Attributes.URL,
			StackID: reqIter.Attributes.StackID,
			CVEScan: cveScan,
		}
		return appl.Codebases().Create(ctx, cdb)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	// new codebase is added register with analytics service
	if err = c.registerCodebaseToGeminiForScan(ctx, cdb.URL); err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	res := &app.CodebaseSingle{
		Data: ConvertCodebase(ctx.Request, *cdb),
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.Request, app.CodebaseHref(res.Data.ID)))
	return ctx.Created(res)
}

// registerCodebaseToGeminiForScan when given the codebase URL, subscribes this codebase
// to enable code scanning to find CVEs with the analytics gemini service
func (c *SpaceCodebasesController) registerCodebaseToGeminiForScan(ctx context.Context, repoURL string) error {
	scanClient := c.AnalyticsGeminiClient()
	req := gemini.NewScanRepoRequest(repoURL)
	return scanClient.Register(ctx, req)
}

// List runs the list action.
func (c *SpaceCodebasesController) List(ctx *app.ListSpaceCodebasesContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	var codebases []codebase.Codebase
	var count int
	err := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Spaces().CheckExists(ctx, ctx.SpaceID)
		if err != nil {
			return err
		}

		codebases, count, err = appl.Codebases().List(ctx, ctx.SpaceID, &offset, &limit)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.CodebaseList{
		Data:  ConvertCodebases(ctx.Request, codebases),
		Meta:  &app.CodebaseListMeta{TotalCount: count},
		Links: &app.PagingLinks{},
	}
	setPagingLinks(res.Links, buildAbsoluteURL(ctx.Request), len(codebases), offset, limit, count)
	return ctx.OK(res)
}
