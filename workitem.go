package main

import (
	"strconv"
	"log"

	"github.com/jinzhu/gorm"
	"github.com/goadesign/goa"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/models"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	db *gorm.DB
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, db *gorm.DB) *WorkitemController {
	ctrl := WorkitemController{Controller: service.NewController("WorkitemController"), db: db}
	if ctrl.db == nil {
		panic("nil db")
	}
	return &ctrl
}

// Show runs the show action.
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	// TBD: implement
	res := models.WorkItem{}
	idVal, error:= strconv.Atoi(ctx.ID);
	if error != nil {
		return error;
	}
	
	log.Printf("looking for id %d", idVal)
	if c.db.First(&res, idVal).RecordNotFound() {
		log.Print("not found, res=%v", res)
		return ctx.NotFound();
	}
	return ctx.OK(&app.WorkItem{
			ID: strconv.FormatUint(uint64(res.ID), 10),
			Name: res.Name,
			Type: res.Type,
			Version: res.Version,
			Fields: res.Fields})
}
