package main

import (
	"log"
	"strconv"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"

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
	res := models.WorkItem{}
	idVal, error := strconv.Atoi(ctx.ID)
	if error != nil {
		return error
	}

	log.Printf("looking for id %d", idVal)
	if c.db.First(&res, idVal).RecordNotFound() {
		log.Print("not found, res=%v", res)
		return ctx.NotFound()
	}
	result := convertFromModel(res)
	return ctx.OK(&result)
}

func convertFromModel(res models.WorkItem) app.WorkItem {
	return app.WorkItem{
		ID:      strconv.FormatUint(uint64(res.ID), 10),
		Name:    res.Name,
		Type:    res.Type,
		Version: res.Version,
		Fields:  res.Fields}
}

func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
	wiType := loadTypeFromDB(*ctx.Payload.TypeID)
	wi := models.WorkItem{
		Name:   *ctx.Payload.Name,
		Type:   *ctx.Payload.TypeID,
		Fields: models.Fields{},
	}

	for fieldName, _ := range wiType.Fields {
		fieldValue := ctx.Payload.Fields[fieldName]
		wi.Fields[fieldName] = fieldValue
		// TODO: typechecking and conversion for stuff like dates.
	}
	tx:= c.db.Begin();

	if tx.Create(&wi) != nil {
		tx.Rollback();
		return ctx.InternalServerError();
	}
	log.Printf("created item %v\n", wi)
	tx.Commit();

	result := convertFromModel(wi)
	return ctx.OK(&result)
}

func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	var workItem models.WorkItem = models.WorkItem{}
	id, error := strconv.ParseUint(ctx.ID, 10, 64)
	if error != nil {
		return ctx.BadRequest()
	}
	tx:= c.db.Begin();
	
	if c.db.First(&workItem, id).RecordNotFound() {
		log.Print("not found, res=%v", workItem)
		tx.Rollback();
		return ctx.NotFound()
	}
	if c.db.Delete(workItem) != nil {
		tx.Rollback()
		return ctx.InternalServerError()
	}

	tx.Commit();
	return ctx.OK([]byte{})
}
