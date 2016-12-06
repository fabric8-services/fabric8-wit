package main

import (
	"golang.org/x/net/context"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/project"
	"github.com/goadesign/goa"
	satoriuuid "github.com/satori/go.uuid"
)

// ProjectController implements the project resource.
type ProjectController struct {
	*goa.Controller
	db application.DB
}

// NewProjectController creates a project controller.
func NewProjectController(service *goa.Service, db application.DB) *ProjectController {
	return &ProjectController{Controller: service.NewController("ProjectController"), db: db}
}

// Create runs the create action.
func (c *ProjectController) Create(ctx *app.CreateProjectContext) error {
	// ProjectController_Create: start_implement

	return application.Transactional(c.db, func(appl application.Application) error {
		project, err := appl.Projects().Create(ctx, ctx.Payload.Data.Attributes.Name)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}

		return ctx.Created(&app.ProjectResponse{Data: projectToAPI(*project)})
	})
	// ProjectController_Create: end_implement
}

// Delete runs the delete action.
func (c *ProjectController) Delete(ctx *app.DeleteProjectContext) error {
	// ProjectController_Delete: start_implement
	return application.Transactional(c.db, func(appl application.Application) error {
		id, err := satoriuuid.FromString(ctx.ID)
		if err != nil {
			errs, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewNotFoundError("project", ctx.ID))
			return ctx.NotFound(errs)
		}
		err = appl.Projects().Delete(ctx.Context, id)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}

		return ctx.OK([]byte{})
	})
	// ProjectController_Delete: end_implement
}

// List runs the list action.
func (c *ProjectController) List(ctx *app.ListProjectContext) error {
	// ProjectController_List: start_implement

	return application.Transactional(c.db, func(appl application.Application) error {
		offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)
		projects, c, err := appl.Projects().List(ctx.Context, &offset, &limit)
		count := int(c)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}

		result := make([]*app.ProjectData, len(projects))
		for index, value := range projects {
			result[index] = projectToAPI(value)
		}

		response := app.ProjectListResponse{
			Links: &app.PagingLinks{},
			Meta:  &app.ProjectMeta{TotalCount: count},
			Data:  result,
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count)

		return ctx.OK(&response)
	})

}

// Show runs the show action.
func (c *ProjectController) Show(ctx *app.ShowProjectContext) error {
	// ProjectController_Show: start_implement

	return application.Transactional(c.db, func(appl application.Application) error {
		id, err := satoriuuid.FromString(ctx.ID)
		if err != nil {
			errs, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewNotFoundError("project", ctx.ID))
			return ctx.NotFound(errs)
		}
		p, err := appl.Projects().Load(ctx.Context, id)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}

		response := app.ProjectResponse{
			Data: projectToAPI(*p),
		}

		return ctx.OK(&response)
	})

	// ProjectController_Show: end_implement
}

// Update runs the update action.
func (c *ProjectController) Update(ctx *app.UpdateProjectContext) error {
	// ProjectController_Update: start_implement

	return application.Transactional(c.db, func(appl application.Application) error {
		id, err := satoriuuid.FromString(ctx.ID)
		if err != nil {
			errs, _ := jsonapi.ErrorToJSONAPIErrors(errors.NewNotFoundError("project", ctx.ID))
			return ctx.NotFound(errs)
		}
		p, err := appl.Projects().Load(ctx.Context, id)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}
		p.Version = ctx.Payload.Data.Attributes.Version
		if ctx.Payload.Data.Attributes.Name != nil {
			p.Name = *ctx.Payload.Data.Attributes.Name
		}

		p, err = appl.Projects().Save(ctx.Context, *p)
		if err != nil {
			respondError(ctx.ResponseData, ctx.Context, err)
		}

		response := app.ProjectResponse{
			Data: projectToAPI(*p),
		}

		return ctx.OK(&response)
	})
	// ProjectController_Update: end_implement
}

func respondError(resp *goa.ResponseData, ctx context.Context, err error) error {
	jerrors, code := jsonapi.ErrorToJSONAPIErrors(err)
	resp.Header().Set("Content-Type", "application/vnd.api+json")
	return resp.Service.Send(ctx, code, jerrors)
}

func projectToAPI(p project.Project) *app.ProjectData {
	return &app.ProjectData{
		ID:   p.ID.String(),
		Type: "projects",
		Attributes: &app.ProjectAttributes{
			Name:      p.Name,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
	}
}
