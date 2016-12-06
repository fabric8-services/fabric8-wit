package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/login"
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
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateCreateProject(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		project, err := appl.Projects().Create(ctx, *ctx.Payload.Data.Attributes.Name)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.ProjectSingle{
			Data: ConvertProject(ctx.RequestData, project),
		}
		ctx.ResponseData.Header().Set("Location", AbsoluteURL(ctx.RequestData, app.ProjectHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// Delete runs the delete action.
func (c *ProjectController) Delete(ctx *app.DeleteProjectContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	id, err := satoriuuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		err = appl.Projects().Delete(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		return ctx.OK([]byte{})
	})
}

// List runs the list action.
func (c *ProjectController) List(ctx *app.ListProjectContext) error {
	offset, limit := computePagingLimts(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		projects, c, err := appl.Projects().List(ctx.Context, &offset, &limit)
		count := int(c)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.ProjectList{
			Links: &app.PagingLinks{},
			Meta:  &app.ProjectListMeta{TotalCount: count},
			Data:  ConvertProjects(ctx.RequestData, projects),
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(projects), offset, limit, count)

		return ctx.OK(&response)
	})

}

// Show runs the show action.
func (c *ProjectController) Show(ctx *app.ShowProjectContext) error {
	id, err := satoriuuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		p, err := appl.Projects().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		resp := app.ProjectSingle{
			Data: ConvertProject(ctx.RequestData, p),
		}

		return ctx.OK(&resp)
	})
}

// Update runs the update action.
func (c *ProjectController) Update(ctx *app.UpdateProjectContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	id, err := satoriuuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	err = validateUpdateProject(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		p, err := appl.Projects().Load(ctx.Context, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		p.Version = *ctx.Payload.Data.Attributes.Version
		if ctx.Payload.Data.Attributes.Name != nil {
			p.Name = *ctx.Payload.Data.Attributes.Name
		}

		p, err = appl.Projects().Save(ctx.Context, *p)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		response := app.ProjectSingle{
			Data: ConvertProject(ctx.RequestData, p),
		}

		return ctx.OK(&response)
	})
}

func validateCreateProject(ctx *app.CreateProjectContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	return nil
}

func validateUpdateProject(ctx *app.UpdateProjectContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Version == nil {
		return errors.NewBadParameterError("data.attributes.version", nil).Expected("not nil")
	}
	return nil
}

// ProjectConvertFunc is a open ended function to add additional links/data/relations to a Project during
// convertion from internal to API
type ProjectConvertFunc func(*goa.RequestData, *project.Project, *app.Project)

// ConvertProjects converts between internal and external REST representation
func ConvertProjects(request *goa.RequestData, projects []*project.Project, additional ...ProjectConvertFunc) []*app.Project {
	var ps = []*app.Project{}
	for _, p := range projects {
		ps = append(ps, ConvertProject(request, p, additional...))
	}
	return ps
}

// ConvertProject converts between internal and external REST representation
func ConvertProject(request *goa.RequestData, p *project.Project, additional ...ProjectConvertFunc) *app.Project {
	selfURL := AbsoluteURL(request, app.ProjectHref(p.ID))
	return &app.Project{
		ID:   p.ID,
		Type: "projects",
		Attributes: &app.ProjectAttributes{
			Name:      &p.Name,
			CreatedAt: &p.CreatedAt,
			UpdatedAt: &p.UpdatedAt,
			Version:   &p.Version,
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
}
