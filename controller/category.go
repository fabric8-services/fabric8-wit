package controller

import (
	"github.com/almighty/almighty-core/application"
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
