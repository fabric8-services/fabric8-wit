package controller

import (
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/category"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/goadesign/goa"
)

// CategoryController implements the category resource.
type CategoryController struct {
	*goa.Controller
	db application.DB
}

// NewCategoryController creates a category controller.
func NewCategoryController(service *goa.Service, db application.DB) *CategoryController {
	return &CategoryController{Controller: service.NewController("CategoryController"), db: db}
}

// List runs the list action.
func (c *CategoryController) List(ctx *app.ListCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {

		categories, err := appl.Categories().List(ctx)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		res := &app.CategoryList{}
		res.Data = ConvertCategories(ctx.RequestData, categories)
		return ctx.OK(res)
	})
}

// ConvertCategories converts between internal and external REST representation
func ConvertCategories(request *goa.RequestData, categories []*category.Category) []*app.Categories {
	var cs = []*app.Categories{}
	for _, c := range categories {
		cs = append(cs, ConvertCategory(request, c))
	}
	return cs
}

// ConvertCategory converts between internal and external REST representation
func ConvertCategory(request *goa.RequestData, cat *category.Category) *app.Categories {
	categoryType := category.APIStringTypeCategory
	c := &app.Categories{
		Type: categoryType,
		ID:   &cat.ID,
		Attributes: &app.CategoryAttributes{
			Name: cat.Name,
		},
	}

	return c
}
