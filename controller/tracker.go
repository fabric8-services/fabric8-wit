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
	"github.com/goadesign/goa"
)

type trackerConfiguration interface {
	GetGithubAuthToken() string
}

// TrackerController implements the tracker resource.
type TrackerController struct {
	*goa.Controller
	db            application.DB
	scheduler     *remoteworkitem.Scheduler
	configuration trackerConfiguration
}

func GetAccessTokens(configuration trackerConfiguration) map[string]string {
	tokens := map[string]string{
		remoteworkitem.ProviderGithub: configuration.GetGithubAuthToken(),
		// add tokens for other types
	}
	return tokens
}

// NewTrackerController creates a tracker controller.
func NewTrackerController(service *goa.Service, db application.DB, scheduler *remoteworkitem.Scheduler, configuration trackerConfiguration) *TrackerController {
	return &TrackerController{
		Controller:    service.NewController("TrackerController"),
		db:            db,
		scheduler:     scheduler,
		configuration: configuration}
}

// Create runs the create action.
func (c *TrackerController) Create(ctx *app.CreateTrackerContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateCreateTrackerPayload(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	var tracker *remoteworkitem.Tracker
	err = application.Transactional(c.db, func(appl application.Application) error {
		tracker = &remoteworkitem.Tracker{
			URL:  ctx.Payload.Data.Attributes.URL,
			Type: ctx.Payload.Data.Attributes.Type,
		}
		return appl.Trackers().Create(ctx.Context, tracker)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	res := &app.TrackerSingle{
		Data: convertTracker(ctx.Request, *tracker),
	}
	ctx.ResponseData.Header().Set("Location", app.TrackerHref(res.Data.ID))
	return ctx.Created(res)
}

// Delete runs the delete action.
func (c *TrackerController) Delete(ctx *app.DeleteTrackerContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = application.Transactional(c.db, func(appl application.Application) error {
		tracker, err := appl.Trackers().Load(ctx.Context, ctx.ID)
		if err != nil {
			return err
		}
		return appl.Trackers().Delete(ctx.Context, tracker.ID)
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return ctx.NoContent()
}

// Show runs the show action.
func (c *TrackerController) Show(ctx *app.ShowTrackerContext) error {
	var trkr *remoteworkitem.Tracker
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		trkr, err = appl.Trackers().Load(ctx.Context, ctx.ID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":        err,
				"tracker_id": ctx.ID,
			}, "unable to load the tracker by ID")
		}
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	result := &app.TrackerSingle{
		Data: convertTracker(ctx.Request, *trkr),
	}
	return ctx.OK(result)
}

// List runs the list action.
func (c *TrackerController) List(ctx *app.ListTrackerContext) error {
	var trkrs []remoteworkitem.Tracker
	err := application.Transactional(c.db, func(appl application.Application) error {
		var err error
		trkrs, err = appl.Trackers().List(ctx)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.TrackerList{}
	res.Data = convertTrackers(ctx.Request, trkrs)
	return ctx.OK(res)
}

// ConvertTrackers from internal to external REST representation
func convertTrackers(request *http.Request, trackers []remoteworkitem.Tracker) []*app.Tracker {
	var ls = []*app.Tracker{}
	for _, i := range trackers {
		ls = append(ls, convertTracker(request, i))
	}
	return ls
}

// Update runs the update action.
func (c *TrackerController) Update(ctx *app.UpdateTrackerContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateUpdateTrackerPayload(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	var trkr *remoteworkitem.Tracker
	err = application.Transactional(c.db, func(appl application.Application) error {
		trkr, err = appl.Trackers().Load(ctx.Context, *ctx.Payload.Data.ID)
		if err != nil {
			return err
		}
		if &ctx.Payload.Data.Attributes.URL != nil {
			trkr.URL = ctx.Payload.Data.Attributes.URL
		}
		if &ctx.Payload.Data.Attributes.Type != nil {
			trkr.Type = ctx.Payload.Data.Attributes.Type
		}
		_, err = appl.Trackers().Save(ctx.Context, trkr)
		return err
	})
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	res := &app.TrackerSingle{
		Data: convertTracker(ctx.Request, *trkr),
	}
	return ctx.OK(res)
}

// ConvertTracker converts from internal to external REST representation
func convertTracker(request *http.Request, tracker remoteworkitem.Tracker) *app.Tracker {
	trackerStringType := remoteworkitem.APIStringTypeTrackers
	selfURL := rest.AbsoluteURL(request, app.TrackerHref(tracker.ID))
	t := &app.Tracker{
		Type: trackerStringType,
		ID:   &tracker.ID,
		Attributes: &app.TrackerAttributes{
			URL:  tracker.URL,
			Type: tracker.Type,
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
	return t
}

func validateCreateTrackerPayload(ctx *app.CreateTrackerContext) error {
	if ctx.Payload.Data.Attributes.URL == "" {
		return errors.NewBadParameterError("URL", "").Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Type == "" {
		return errors.NewBadParameterError("Type", "").Expected("not nil")
	}
	return nil
}

func validateUpdateTrackerPayload(ctx *app.UpdateTrackerContext) error {
	if ctx.Payload.Data.ID == nil {
		return errors.NewBadParameterError("ID", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.URL == "" {
		return errors.NewBadParameterError("URL", "").Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Type == "" {
		return errors.NewBadParameterError("Type", "").Expected("not nil")
	}
	return nil
}
