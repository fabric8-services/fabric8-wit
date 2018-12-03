package controller

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
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
		err = appl.Spaces().CheckExists(ctx, *ctx.Payload.Data.Relationships.Space.Data.ID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"space_id": ctx.Payload.Data.Relationships.Space.Data.ID,
			}, "unable to load space")
			return errors.NewBadParameterError("space", ctx.Payload.Data.Relationships.Space.Data.ID.String()).Expected("valid space ID")
		}

		err = appl.Trackers().CheckExists(ctx, ctx.Payload.Data.Relationships.Tracker.Data.ID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":        err,
				"tracker_id": ctx.Payload.Data.Relationships.Tracker.Data.ID,
			}, "unable to load tracker")
			return errors.NewBadParameterError("tracker", ctx.Payload.Data.Relationships.Tracker.Data.ID.String()).Expected("valid tracker ID")
		}
		if ctx.Payload.Data.ID != nil {
			// check if tracker query id exists
			err = appl.TrackerQueries().CheckExists(ctx, *ctx.Payload.Data.ID)
			if err == nil {
				log.Error(ctx, map[string]interface{}{
					"err":             err,
					"trackerquery_id": ctx.Payload.Data.Relationships.Tracker.Data.ID,
				}, "unable to load trackerquery")
				return errors.NewBadParameterError("trackerquery", ctx.Payload.Data.ID.String()).Expected("valid trackerquery ID")
			}

			// check if tracker query id is uuid.Nil
			if *ctx.Payload.Data.ID == uuid.Nil {
				return errors.NewBadParameterError("trackerquery", ctx.Payload.Data.ID.String()).Expected("valid trackerquery ID")
			}
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		trackerQuery := remoteworkitem.TrackerQuery{
			Query:          ctx.Payload.Data.Attributes.Query,
			Schedule:       ctx.Payload.Data.Attributes.Schedule,
			TrackerID:      ctx.Payload.Data.Relationships.Tracker.Data.ID,
			SpaceID:        *ctx.Payload.Data.Relationships.Space.Data.ID,
			WorkItemTypeID: ctx.Payload.Data.Relationships.BaseType.Data.ID,
		}
		if ctx.Payload.Data.ID != nil {
			trackerQuery.ID = *ctx.Payload.Data.ID
		}
		tq, err := appl.TrackerQueries().Create(ctx.Context, trackerQuery)
		if err != nil {
			return errs.Wrapf(err, "failed to create tracker query %s", ctx.Payload.Data)
		}
		res := &app.TrackerQuerySingle{
			Data: convertTrackerQueryToApp(appl, ctx.Request, *tq),
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerqueryHref(tq.ID))
		return ctx.Created(res)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
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
			Data: convertTrackerQueryToApp(appl, ctx.Request, *trackerquery),
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
		if ctx.Payload.Data.Relationships.BaseType.Data.ID == uuid.Nil {
			return errors.NewBadParameterError("Workitemtype_id", nil).Expected("not nil")
		}
		_, err = appl.TrackerQueries().Save(ctx.Context, *tq)
		if err != nil {
			return errs.Wrapf(err, "failed to update tracker query %s", ctx.Payload.Data.ID)
		}
		res := &app.TrackerQuerySingle{
			Data: convertTrackerQueryToApp(appl, ctx.Request, *tq),
		}
		return ctx.OK(res)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
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
		return jsonapi.JSONErrorResponse(ctx, err)
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
		res.Data = ConvertTrackerQueriesToApp(appl, ctx.Request, trackerqueries)
		return ctx.OK(res)
	})
}

// ConvertTrackerQueriesToApp from internal to external REST representation
func ConvertTrackerQueriesToApp(appl application.Application, request *http.Request, trackerqueries []remoteworkitem.TrackerQuery) []*app.TrackerQuery {
	var ls = []*app.TrackerQuery{}
	for _, i := range trackerqueries {
		ls = append(ls, convertTrackerQueryToApp(appl, request, i))
	}
	return ls
}

// ConvertTrackerQueryToApp converts from internal to external REST representation
func convertTrackerQueryToApp(appl application.Application, request *http.Request, trackerquery remoteworkitem.TrackerQuery) *app.TrackerQuery {
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
		return errors.NewBadParameterError("Query", "").Expected("not empty")
	}
	if ctx.Payload.Data.Attributes.Schedule == "" {
		return errors.NewBadParameterError("Schedule", "").Expected("not empty")
	}
	if ctx.Payload.Data.Relationships.Tracker.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("TrackerID", nil).Expected("not nil")
	}
	if *ctx.Payload.Data.Relationships.Space.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("SpaceID", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Relationships.BaseType.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("Workitemtype_id", nil).Expected("not nil")
	}
	return nil
}

func validateUpdateTrackerQueryPayload(ctx *app.UpdateTrackerqueryContext) error {
	if ctx.Payload.Data.ID == nil {
		return errors.NewBadParameterError("ID", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Query == "" {
		return errors.NewBadParameterError("Query", "").Expected("not empty")
	}
	if ctx.Payload.Data.Attributes.Schedule == "" {
		return errors.NewBadParameterError("Schedule", "").Expected("not empty")
	}
	if ctx.Payload.Data.Relationships.Tracker.Data.ID == uuid.Nil {
		return errors.NewBadParameterError("TrackerID", nil).Expected("not nil")
	}
	return nil
}
