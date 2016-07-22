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
	id, err := strconv.ParseUint(ctx.ID, 10, 64)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest("Could not parse id: %s", err.Error()))
	}

	log.Printf("looking for id %d", id)
	if c.db.First(&res, id).RecordNotFound() {
		log.Printf("not found, res=%v", res)
		return ctx.NotFound()
	}
	wiType, err := loadTypeFromDB(res.Type)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(err.Error()))
	}
	result, err := convertFromModel(*wiType, res)
	if err != nil {
		ctx.InternalServerError()
	}
	return ctx.OK(result)
}

func convertFromModel(wiType models.WorkItemType, workItem models.WorkItem) (*app.WorkItem, error) {
	result := app.WorkItem{
		ID:      strconv.FormatUint(workItem.ID, 10),
		Name:    workItem.Name,
		Type:    workItem.Type,
		Version: workItem.Version,
		Fields:  map[string]interface{}{}}

	for name, field := range wiType.Fields {
		var err error
		result.Fields[name], err = field.ConvertFromModel(name, workItem.Fields[name])
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// Create runs the create action.
func (c *WorkitemController) Create(ctx *app.CreateWorkitemContext) error {
	wiType, err := loadTypeFromDB(ctx.Payload.Type)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(err.Error()))
	}
	wi := models.WorkItem{
		Name:   ctx.Payload.Name,
		Type:   ctx.Payload.Type,
		Fields: models.Fields{},
	}

	for fieldName, fieldDef := range wiType.Fields {
		fieldValue := ctx.Payload.Fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return ctx.BadRequest(goa.ErrBadRequest("Could not parse id: %s", err.Error()))
		}
	}
	tx := c.db.Begin()

	if tx.Create(&wi).Error != nil {
		tx.Rollback()
		return ctx.InternalServerError()
	}
	log.Printf("created item %v\n", wi)
	result, err := convertFromModel(*wiType, wi)
	if err != nil {
		tx.Rollback()
		return ctx.InternalServerError()
	}
	tx.Commit()
	return ctx.OK(result)
}

// Delete runs the delete action.
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {
	var workItem models.WorkItem = models.WorkItem{}
	id, err := strconv.ParseUint(ctx.ID, 10, 64)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest("Could not parse id: %s", err.Error()))
	}
	tx := c.db.Begin()

	if tx.First(&workItem, id).RecordNotFound() {
		log.Print("not found, res=%v", workItem)
		tx.Rollback()
		return ctx.NotFound()
	}
	if tx.Delete(workItem).Error != nil {
		tx.Rollback()
		return ctx.InternalServerError()
	}

	tx.Commit()
	return ctx.OK([]byte{})
}

// Update runs the update action.
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	res := models.WorkItem{}
	id, err := strconv.ParseUint(ctx.Payload.ID, 10, 64)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest("Could not parse id: %s", err.Error()))
	}

	log.Printf("looking for id %d", id)
	tx := c.db.Begin()
	if tx.First(&res, id).RecordNotFound() {
		tx.Rollback()
		log.Printf("not found, res=%v", res)
		return ctx.NotFound()
	}
	if res.Version != ctx.Payload.Version {
		tx.Rollback()
		return ctx.BadRequest(goa.ErrBadRequest("version conflict: expected %d but got %d", res.Version, ctx.Payload.Version))
	}

	wiType, err := loadTypeFromDB(res.Type)
	if err != nil {
		return ctx.BadRequest(goa.ErrBadRequest(err.Error()))
	}

	wi := models.WorkItem{
		ID: id,
		Name:    ctx.Payload.Name,
		Type:    ctx.Payload.Type,
		Version: ctx.Payload.Version + 1,
		Fields:  models.Fields{},
	}

	for fieldName, fieldDef := range wiType.Fields {
		fieldValue := ctx.Payload.Fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			tx.Rollback()
			return ctx.BadRequest(goa.ErrBadRequest("Could not convert field %s: %s", fieldName, err.Error()))
		}
	}
	
	if err:= tx.Save(&wi).Error;err != nil {
		tx.Rollback()
		log.Print(err.Error())
		return ctx.InternalServerError()
	}
	log.Printf("updated item to %v\n", wi)
	result, err := convertFromModel(*wiType, wi)
	if err != nil {
		tx.Rollback()
		return ctx.InternalServerError()
	}
	tx.Commit()
	return ctx.OK(result)
}
