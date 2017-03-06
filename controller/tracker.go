package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	query "github.com/almighty/almighty-core/query/simple"
	"github.com/almighty/almighty-core/remoteworkitem"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

const (
	// APIStringTypeTracker to be used as a TYPE for jsonapi based tracker APIs
	APIStringTypeTracker = "trackers"
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
	return &TrackerController{Controller: service.NewController("TrackerController"), db: db, scheduler: scheduler, configuration: configuration}
}

// Create runs the create action.
func (c *TrackerController) Create(ctx *app.CreateTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		newTracker, err := appl.Trackers().Create(ctx.Context, ctx.Payload.Data.Attributes.URL, ctx.Payload.Data.Attributes.Type)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		trackerData := app.TrackerData{
			ID:   &newTracker.ID,
			Type: APIStringTypeTracker,
			Attributes: &app.TrackerAttributes{
				Type: newTracker.Type,
				URL:  newTracker.URL,
			},
		}
		respTracker := app.TrackerObjectSingle{
			Data: &trackerData,
			Links: &app.TrackerLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerHref(respTracker.Data.ID))
		return ctx.Created(&respTracker)
	})
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(accessTokens)
	return result
}

// Delete runs the delete action.
func (c *TrackerController) Delete(ctx *app.DeleteTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Trackers().Delete(ctx.Context, ctx.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		return ctx.OK([]byte{})
	})
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(accessTokens)
	return result
}

// Show runs the show action.
func (c *TrackerController) Show(ctx *app.ShowTrackerContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		t, err := appl.Trackers().Load(ctx.Context, ctx.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				log.Error(ctx, map[string]interface{}{
					"trackerID": ctx.ID,
				}, "tracker controller not found")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
			}
		}
		jsonapiTrackerObject := app.TrackerObjectSingle{
			Data: convertTracker(t),
			Links: &app.TrackerLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		return ctx.OK(&jsonapiTrackerObject)
	})
}

// List runs the list action.
func (c *TrackerController) List(ctx *app.ListTrackerContext) error {
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse filter: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("could not parse paging: %s", err.Error())))
		return ctx.BadRequest(jerrors)
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.Trackers().List(ctx.Context, exp, start, &limit)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(fmt.Sprintf("Error listing trackers: %s", err.Error())))
			return ctx.InternalServerError(jerrors)
		}
		jsonapiData := make([]*app.TrackerData, len(result))
		for i, tracker := range result {
			jsonapiData[i] = convertTracker(tracker)
		}
		response := app.TrackerObjectList{
			Data: jsonapiData,
			Links: &app.TrackerLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		return ctx.OK(&response)
	})

}

// Update runs the update action.
func (c *TrackerController) Update(ctx *app.UpdateTrackerContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {

		if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == "" {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewBadParameterError("data.id", nil))
			return ctx.NotFound(jerrors)
		}

		tr, err := appl.Trackers().Load(ctx, ctx.Payload.Data.ID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrNotFound(fmt.Sprintf("Error updating tracker item: %s", err.Error())))
			return ctx.NotFound(jerrors)
		}

		if ctx.Payload.Data.Attributes != nil && ctx.Payload.Data.Attributes.Type != nil && *ctx.Payload.Data.Attributes.Type != "" {
			tr.Type = *ctx.Payload.Data.Attributes.Type
		}
		if ctx.Payload.Data.Attributes != nil && ctx.Payload.Data.Attributes.URL != nil && *ctx.Payload.Data.Attributes.URL != "" {
			tr.URL = *ctx.Payload.Data.Attributes.URL
		}
		t, err := appl.Trackers().Save(ctx.Context, *tr)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		jsonapiTrackerObject := app.TrackerObjectSingle{
			Data: convertTracker(t),
			Links: &app.TrackerLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}
		return ctx.OK(&jsonapiTrackerObject)
	})
	accessTokens := GetAccessTokens(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(accessTokens)
	return result
}

// convertTracker converts app.Tracker object into jsonapi based app.TrackerData
func convertTracker(t *app.Tracker) *app.TrackerData {
	return &app.TrackerData{
		ID:   &t.ID,
		Type: APIStringTypeTracker,
		Attributes: &app.TrackerAttributes{
			Type: t.Type,
			URL:  t.URL,
		},
	}
}
