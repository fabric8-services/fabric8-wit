package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/goadesign/goa"
)

type trackerQueryConfiguration interface {
	GetGithubAuthToken() string
}

// TrackerqueryController implements the trackerquery resource.
type TrackerqueryController struct {
	*goa.Controller
	db            application.DB
	scheduler     *remoteworkitem.Scheduler
	configuration trackerQueryConfiguration
}

func getAccessTokensForTrackerQuery(configuration trackerQueryConfiguration) map[string]string {
	tokens := map[string]string{
		remoteworkitem.ProviderGithub: configuration.GetGithubAuthToken(),
		// add tokens for other types
	}
	return tokens
}

// NewTrackerqueryController creates a trackerquery controller.
func NewTrackerqueryController(service *goa.Service, db application.DB, scheduler *remoteworkitem.Scheduler, configuration trackerQueryConfiguration) *TrackerqueryController {
	return &TrackerqueryController{Controller: service.NewController("TrackerqueryController"), db: db, scheduler: scheduler, configuration: configuration}
}

// Create runs the create action.
func (c *TrackerqueryController) Create(ctx *app.CreateTrackerqueryContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateCreateTrackerQueryPayload(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		trackerQuery := remoteworkitem.TrackerQuery{
			ID:        *ctx.Payload.Data.ID,
			Query:     ctx.Payload.Data.Attributes.Query,
			Schedule:  ctx.Payload.Data.Attributes.Schedule,
			TrackerID: ctx.Payload.Data.Relationships.Tracker.Data.ID,
			SpaceID:   uuid.FromStringOrNil(*ctx.Payload.Data.Relationships.Space.Data.ID),
		}
		tq, err := appl.TrackerQueries().Create(ctx.Context, trackerQuery)
		if err != nil {
			return errs.Wrapf(err, "failed to create tracker query %s", ctx.Payload.Data)
		}
		res := &app.TrackerQuerySingle{
			Data: convertTrackerQuery(appl, ctx.Request, *tq),
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerqueryHref(tq.ID))
		return ctx.Created(res)
	})
	if err != nil {
		return errs.Wrapf(err, "failed to create tracker query %s", ctx.Payload.Data)
	}
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return nil
}

// Show runs the show action.
func (c *TrackerqueryController) Show(ctx *app.ShowTrackerqueryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		trackerquery, err := appl.TrackerQueries().Load(ctx.Context, ctx.ID)
		if err != nil {
			return errs.Wrapf(err, "failed to load tracker query %s", ctx.ID)
		}
		result := &app.TrackerQuerySingle{
			Data: convertTrackerQuery(appl, ctx.Request, *trackerquery),
		}
		return ctx.OK(result)
	})
	return nil
}

// Update runs the update action.
func (c *TrackerqueryController) Update(ctx *app.UpdateTrackerqueryContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateUpdateTrackerQueryPayload(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	err = application.Transactional(c.db, func(appl application.Application) error {

		tq, err := appl.TrackerQueries().Load(ctx.Context, *ctx.Payload.Data.ID)
		if err != nil {
			return errs.Wrapf(err, "failed to update tracker query %s", ctx.Payload.Data.ID)
		}
		if &ctx.Payload.Data.Attributes.Query != nil {
			tq.Query = ctx.Payload.Data.Attributes.Query
		}
		if &ctx.Payload.Data.Attributes.Schedule != nil {
			tq.Schedule = ctx.Payload.Data.Attributes.Schedule
		}
		if &ctx.Payload.Data.Relationships.Tracker.Data.ID != nil {
			tq.TrackerID = ctx.Payload.Data.Relationships.Tracker.Data.ID
		}
		_, err = appl.TrackerQueries().Save(ctx.Context, *tq)
		if err != nil {
			return errs.Wrapf(err, "failed to update tracker query %s", ctx.Payload.Data.ID)
		}
		res := &app.TrackerQuerySingle{
			Data: convertTrackerQuery(appl, ctx.Request, *tq),
		}
		return ctx.OK(res)
	})
	if err != nil {
		return errs.Wrapf(err, "failed to update tracker query %s", ctx.Payload.Data.ID)
	}
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return nil
}

// Delete runs the delete action.
func (c *TrackerqueryController) Delete(ctx *app.DeleteTrackerqueryContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		tq, err := appl.TrackerQueries().Load(ctx.Context, ctx.ID)
		if err != nil {
			return errs.Wrapf(err, "failed to delete tracker query %s", ctx.ID)
		}
		return appl.TrackerQueries().Delete(ctx.Context, tq.ID)
	})
	if err != nil {
		return errs.Wrapf(err, "failed to delete tracker query %s", ctx.ID)
	}
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return ctx.NoContent()
}

// List runs the list action.
func (c *TrackerqueryController) List(ctx *app.ListTrackerqueryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		trackerqueries, err := appl.TrackerQueries().List(ctx)
		if err != nil {
			return errs.Wrapf(err, "failed to list tracker queries")
		}
		res := &app.TrackerQueryList{}
		res.Data = ConvertTrackerQueries(appl, ctx.Request, trackerqueries)
		return ctx.OK(res)
	})
}

// ConvertTrackerQueries from internal to external REST representation
func ConvertTrackerQueries(appl application.Application, request *http.Request, trackerqueries []remoteworkitem.TrackerQuery) []*app.TrackerQuery {
	var ls = []*app.TrackerQuery{}
	for _, i := range trackerqueries {
		ls = append(ls, convertTrackerQuery(appl, request, i))
	}
	return ls
}

// ConvertTrackerQuery converts from internal to external REST representation
func convertTrackerQuery(appl application.Application, request *http.Request, trackerquery remoteworkitem.TrackerQuery) *app.TrackerQuery {
	trackerQueryStringType := remoteworkitem.APIStringTypeTrackerQuery
	selfURL := rest.AbsoluteURL(request, app.TrackerqueryHref(trackerquery.ID))
	t := &app.TrackerQuery{
		Type: trackerQueryStringType,
		ID:   &trackerquery.ID,
		Attributes: &app.TrackerQueryAttributes{
			Query:    trackerquery.Query,
			Schedule: trackerquery.Schedule,
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
	return t
}

func validateCreateTrackerQueryPayload(ctx *app.CreateTrackerqueryContext) error {
	if ctx.Payload.Data.Attributes.Query == "" {
		return errors.NewBadParameterError("Query", "").Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Schedule == "" {
		return errors.NewBadParameterError("Schedule", "").Expected("not nil")
	}
	if ctx.Payload.Data.Relationships.Tracker.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("TrackerID", nil).Expected("not nil")
	}
	return nil
}

func validateUpdateTrackerQueryPayload(ctx *app.UpdateTrackerqueryContext) error {
	if ctx.Payload.Data.ID == nil {
		return errors.NewBadParameterError("ID", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Query == "" {
		return errors.NewBadParameterError("Query", "").Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Schedule == "" {
		return errors.NewBadParameterError("Schedule", "").Expected("not nil")
	}
	if ctx.Payload.Data.Relationships.Tracker.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("TrackerID", nil).Expected("not nil")
	}
	return nil
}
