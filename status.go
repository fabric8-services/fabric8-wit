package main

import (
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
)

// StatusController implements the status resource.
type StatusController struct {
	*goa.Controller
	db *gorm.DB
}

// NewStatusController creates a status controller.
func NewStatusController(service *goa.Service, db *gorm.DB) *StatusController {
	return &StatusController{
		Controller:	service.NewController("StatusController"),
		db:		db,
	}
}

// Show runs the show action.
func (c *StatusController) Show(ctx *app.ShowStatusContext) error {
	res := &app.Status{}
	res.Commit = Commit
	res.BuildTime = BuildTime
	res.StartTime = StartTime

	_, err := c.db.DB().Exec("select 1")
	if err != nil {
		return ctx.ServiceUnavailable(err);
	}
	return ctx.OK(res)
}